package k8s

import (
	"context"
	"fmt"
	"kube-node-manager/internal/cache"
	"kube-node-manager/internal/podcache"
	"kube-node-manager/pkg/logger"
	"strconv"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	metricsclientset "k8s.io/metrics/pkg/client/clientset/versioned"
)

// Service Kubernetes客户端服务
type Service struct {
	logger          *logger.Logger
	clients         map[string]*kubernetes.Clientset
	metricsClients  map[string]*metricsclientset.Clientset
	mu              sync.RWMutex
	cache           *cache.K8sCache          // 旧的K8s API缓存层（仅用于非节点资源）
	podCountCache   *podcache.PodCountCache  // 轻量级 Pod 统计缓存（基于 Informer）
	realtimeManager interface{}              // 实时同步管理器（使用接口避免循环依赖）
}

// NodeInfo Kubernetes节点信息
type NodeInfo struct {
	Name                string             `json:"name"`
	Status              string             `json:"status"`
	Schedulable         bool               `json:"schedulable"`
	UnschedulableReason string             `json:"unschedulable_reason,omitempty"` // 禁止调度原因
	Roles               []string           `json:"roles"`
	Age                 string             `json:"age"`
	Version             string             `json:"version"`
	InternalIP          string             `json:"internal_ip"`
	ExternalIP          string             `json:"external_ip"`
	OS                  string             `json:"os"`
	OSImage             string             `json:"os_image"`
	KernelVersion       string             `json:"kernel_version"`
	ContainerRuntime    string             `json:"container_runtime"`
	Capacity            ResourceInfo       `json:"capacity"`
	Allocatable         ResourceInfo       `json:"allocatable"`
	Usage               *ResourceUsageInfo `json:"usage,omitempty"` // 资源使用情况
	Labels              map[string]string  `json:"labels"`
	Taints              []TaintInfo        `json:"taints"`
	Conditions          []NodeCondition    `json:"conditions"`
	CreatedAt           time.Time          `json:"created_at"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	CPU    string            `json:"cpu"`
	Memory string            `json:"memory"`
	Pods   string            `json:"pods"`
	GPU    map[string]string `json:"gpu,omitempty"` // 支持多种GPU类型
}

// ResourceUsageInfo 资源使用信息
type ResourceUsageInfo struct {
	CPU    string `json:"cpu"`
	Memory string `json:"memory"`
	Pods   string `json:"pods"` // 实际运行的 Pod 数量
}

// TaintInfo 污点信息
type TaintInfo struct {
	Key       string     `json:"key"`
	Value     string     `json:"value"`
	Effect    string     `json:"effect"`
	TimeAdded *time.Time `json:"time_added,omitempty"`
}

// NodeCondition 节点状态
type NodeCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastHeartbeatTime  time.Time `json:"last_heartbeat_time"`
	LastTransitionTime time.Time `json:"last_transition_time"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// LabelUpdateRequest 标签更新请求
type LabelUpdateRequest struct {
	NodeName string            `json:"node_name" binding:"required"`
	Labels   map[string]string `json:"labels" binding:"required"`
}

// TaintUpdateRequest 污点更新请求
type TaintUpdateRequest struct {
	NodeName string      `json:"node_name" binding:"required"`
	Taints   []TaintInfo `json:"taints" binding:"required"`
}

// ClusterInfo 集群信息
type ClusterInfo struct {
	Version   string     `json:"version"`
	NodeCount int        `json:"node_count"`
	Nodes     []NodeInfo `json:"nodes,omitempty"`
	LastSync  time.Time  `json:"last_sync"`
}

// NewService 创建新的Kubernetes服务实例
func NewService(logger *logger.Logger, realtimeMgr interface{}) *Service {
	return &Service{
		logger:          logger,
		clients:         make(map[string]*kubernetes.Clientset),
		metricsClients:  make(map[string]*metricsclientset.Clientset),
		cache:           cache.NewK8sCache(logger),
		podCountCache:   podcache.NewPodCountCache(logger),
		realtimeManager: realtimeMgr,
	}
}

// GetPodCountCache 获取 Pod 统计缓存（供外部注册到 Informer）
func (s *Service) GetPodCountCache() *podcache.PodCountCache {
	return s.podCountCache
}

// CreateClient 根据kubeconfig创建Kubernetes客户端
func (s *Service) CreateClient(clusterName, kubeconfig string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		s.logger.Errorf("Failed to parse kubeconfig for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// 设置超时 - 针对大规模集群增加超时时间
	config.Timeout = 60 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		s.logger.Errorf("Failed to create Kubernetes client for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用基础的API版本检查来验证连接
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		s.logger.Errorf("Failed to test connection for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}

	// 尝试检查节点权限（可选）
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		s.logger.Warningf("Limited permissions for cluster %s: cannot list nodes: %v", clusterName, err)
		// 不阻止客户端创建，只是记录警告
	}

	s.clients[clusterName] = clientset

	// 创建metrics client（可选，用于获取资源使用情况）
	metricsClient, err := metricsclientset.NewForConfig(config)
	if err != nil {
		s.logger.Warningf("Failed to create metrics client for cluster %s (metrics may not be available): %v", clusterName, err)
	} else {
		// 测试metrics API是否可用
		ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel2()
		_, err = metricsClient.MetricsV1beta1().NodeMetricses().List(ctx2, metav1.ListOptions{Limit: 1})
		if err != nil {
			s.logger.Warningf("Metrics API not available for cluster %s: %v", clusterName, err)
		} else {
			s.metricsClients[clusterName] = metricsClient
			s.logger.Infof("Successfully created metrics client for cluster: %s", clusterName)
		}
	}

	// 注册集群到实时同步管理器
	if s.realtimeManager != nil {
		type RealtimeManager interface {
			RegisterCluster(clusterName string, clientset *kubernetes.Clientset) error
		}
		if rtMgr, ok := s.realtimeManager.(RealtimeManager); ok {
			if err := rtMgr.RegisterCluster(clusterName, clientset); err != nil {
				s.logger.Errorf("Failed to register cluster %s to realtime manager: %v", clusterName, err)
				// 不返回错误，允许继续使用传统模式
			}
		}
	}

	s.logger.Infof("Successfully created Kubernetes client for cluster: %s", clusterName)
	return nil
}

// RemoveClient 移除Kubernetes客户端
func (s *Service) RemoveClient(clusterName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clusterName)
	delete(s.metricsClients, clusterName)
	s.logger.Infof("Removed Kubernetes client for cluster: %s", clusterName)
}

// getClient 获取指定集群的客户端
func (s *Service) getClient(clusterName string) (*kubernetes.Clientset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clusterName]
	if !exists {
		s.logger.Errorf("Kubernetes client not found for cluster: %s. Available clusters: %v", clusterName, s.getAvailableClusterNames())
		return nil, fmt.Errorf("kubernetes client not found for cluster: %s. Please check if the cluster is properly configured and connected", clusterName)
	}
	return client, nil
}

// getMetricsClient 获取指定集群的metrics客户端
func (s *Service) getMetricsClient(clusterName string) (*metricsclientset.Clientset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.metricsClients[clusterName]
	if !exists {
		return nil, fmt.Errorf("metrics client not found for cluster: %s (metrics may not be available)", clusterName)
	}
	return client, nil
}

// getAvailableClusterNames 获取可用的集群名称列表（用于调试）
func (s *Service) getAvailableClusterNames() []string {
	var names []string
	for name := range s.clients {
		names = append(names, name)
	}
	return names
}

// GetClusterInfo 获取集群信息
func (s *Service) GetClusterInfo(clusterName string) (*ClusterInfo, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	// 获取集群版本
	version, err := client.Discovery().ServerVersion()
	if err != nil {
		s.logger.Errorf("Failed to get server version for cluster %s: %v", clusterName, err)
		return nil, fmt.Errorf("failed to get server version: %w", err)
	}

	// 获取节点列表
	nodes, err := s.ListNodes(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	return &ClusterInfo{
		Version:   version.GitVersion,
		NodeCount: len(nodes),
		Nodes:     nodes,
		LastSync:  time.Now(),
	}, nil
}

// ListNodes 获取节点列表（带缓存）
func (s *Service) ListNodes(clusterName string) ([]NodeInfo, error) {
	return s.ListNodesWithCache(clusterName, false)
}

// ListNodesWithCache 获取节点列表（可选择是否强制刷新缓存）
func (s *Service) ListNodesWithCache(clusterName string, forceRefresh bool) ([]NodeInfo, error) {
	// 尝试使用智能缓存（由 Informer 实时更新）
	if s.realtimeManager != nil && !forceRefresh {
		type SmartCacheProvider interface {
			GetSmartCache() interface{}
		}
		if rtMgr, ok := s.realtimeManager.(SmartCacheProvider); ok {
			smartCache := rtMgr.GetSmartCache()
			if smartCache != nil {
				type SmartCache interface {
					GetNodes(clusterName string) ([]*corev1.Node, bool)
				}
				if sc, ok := smartCache.(SmartCache); ok {
					k8sNodes, found := sc.GetNodes(clusterName)
					if found && len(k8sNodes) > 0 {
						// 转换 corev1.Node 切片为 NodeInfo 切片
						nodes := make([]NodeInfo, 0, len(k8sNodes))
						for _, k8sNode := range k8sNodes {
							nodeInfo := s.nodeToNodeInfo(k8sNode)
							nodes = append(nodes, nodeInfo)
						}
						s.logger.Infof("Retrieved %d nodes from smart cache for cluster %s", len(nodes), clusterName)
						return nodes, nil
					}
					// SmartCache 未就绪或无数据，回退到传统方式
					s.logger.Infof("SmartCache not ready for cluster %s, falling back to API", clusterName)
				}
			}
		}
	}

	// 传统模式：使用旧的缓存逻辑或直接从 API 获取
	ctx := context.Background()

	// 定义获取函数
	fetchFunc := func() (interface{}, error) {
		return s.fetchNodesFromAPI(clusterName)
	}

	// 使用缓存层
	cachedData, err := s.cache.GetNodeList(ctx, clusterName, forceRefresh, fetchFunc)
	if err != nil {
		return nil, err
	}

	// 类型断言
	nodes, ok := cachedData.([]NodeInfo)
	if !ok {
		return nil, fmt.Errorf("invalid cached data type")
	}

	// 异步预取前20个节点的详情（优化用户体验）
	if len(nodes) > 0 && !forceRefresh {
		go func() {
			nodeNames := make([]string, 0, len(nodes))
			for _, node := range nodes {
				nodeNames = append(nodeNames, node.Name)
			}

			s.cache.PrefetchNodeDetails(clusterName, nodeNames, 20, func(nodeName string) (interface{}, error) {
				return s.fetchNodeFromAPI(clusterName, nodeName)
			})
		}()
	}

	return nodes, nil
}

// fetchNodesFromAPI 从K8s API获取节点列表（无缓存）
func (s *Service) fetchNodesFromAPI(clusterName string) ([]NodeInfo, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		s.logger.Errorf("Failed to get client for cluster %s: %v", clusterName, err)
		return nil, fmt.Errorf("cluster connection not available for %s: %w", clusterName, err)
	}

	// 针对大规模集群增加超时时间
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Errorf("Failed to list nodes for cluster %s: %v", clusterName, err)
		// 提供更详细的错误信息以帮助诊断
		if strings.Contains(err.Error(), "connection refused") {
			return nil, fmt.Errorf("cluster %s is not reachable: connection refused", clusterName)
		}
		if strings.Contains(err.Error(), "forbidden") {
			return nil, fmt.Errorf("insufficient permissions to list nodes in cluster %s: %w", clusterName, err)
		}
		if strings.Contains(err.Error(), "unauthorized") {
			return nil, fmt.Errorf("authentication failed for cluster %s: %w", clusterName, err)
		}
		if strings.Contains(err.Error(), "timeout") {
			return nil, fmt.Errorf("timeout connecting to cluster %s: %w", clusterName, err)
		}
		return nil, fmt.Errorf("failed to list nodes in cluster %s: %w", clusterName, err)
	}

	s.logger.Infof("Successfully retrieved %d nodes from cluster %s (uncached)", len(nodeList.Items), clusterName)

	var nodes []NodeInfo
	gpuNodeCount := 0
	totalGPUCount := 0

	for _, node := range nodeList.Items {
		nodeInfo := s.nodeToNodeInfo(&node)
		nodes = append(nodes, nodeInfo)

		// 统计 GPU 节点数量和总 GPU 数量
		if len(nodeInfo.Capacity.GPU) > 0 {
			gpuNodeCount++
			for _, gpuCount := range nodeInfo.Capacity.GPU {
				// 解析 GPU 数量字符串
				if count, err := strconv.Atoi(gpuCount); err == nil {
					totalGPUCount += count
				}
			}
		}
	}

	// 尝试获取资源使用情况
	s.enrichNodesWithMetrics(clusterName, nodes)

	// 输出 GPU 资源汇总日志
	if gpuNodeCount > 0 {
		s.logger.Infof("Cluster %s: Found %d GPU nodes with total %d GPUs", clusterName, gpuNodeCount, totalGPUCount)
	}

	return nodes, nil
}

// GetNode 获取单个节点信息（带缓存）
func (s *Service) GetNode(clusterName, nodeName string) (*NodeInfo, error) {
	return s.GetNodeWithCache(clusterName, nodeName, false)
}

// GetNodeWithCache 获取单个节点信息（可选择是否强制刷新缓存）
func (s *Service) GetNodeWithCache(clusterName, nodeName string, forceRefresh bool) (*NodeInfo, error) {
	// 尝试使用智能缓存（由 Informer 实时更新）
	if s.realtimeManager != nil && !forceRefresh {
		type SmartCacheProvider interface {
			GetSmartCache() interface{}
		}
		if rtMgr, ok := s.realtimeManager.(SmartCacheProvider); ok {
			smartCache := rtMgr.GetSmartCache()
			if smartCache != nil {
				type SmartCache interface {
					GetNode(clusterName, nodeName string) (*corev1.Node, bool)
				}
				if sc, ok := smartCache.(SmartCache); ok {
					k8sNode, found := sc.GetNode(clusterName, nodeName)
					if found && k8sNode != nil {
						// 转换 corev1.Node 为 NodeInfo
						nodeInfo := s.nodeToNodeInfo(k8sNode)
						s.logger.Infof("Retrieved node %s from smart cache for cluster %s", nodeName, clusterName)
						return &nodeInfo, nil
					}
					// SmartCache 未就绪或无数据，回退到传统方式
					s.logger.Infof("SmartCache not ready for node %s/%s, falling back to API", clusterName, nodeName)
				}
			}
		}
	}

	// 传统模式：使用旧的缓存逻辑或直接从 API 获取
	ctx := context.Background()

	// 定义获取函数
	fetchFunc := func() (interface{}, error) {
		return s.fetchNodeFromAPI(clusterName, nodeName)
	}

	// 使用缓存层
	cachedData, err := s.cache.GetNodeDetail(ctx, clusterName, nodeName, forceRefresh, fetchFunc)
	if err != nil {
		return nil, err
	}

	// 类型断言
	node, ok := cachedData.(*NodeInfo)
	if !ok {
		return nil, fmt.Errorf("invalid cached data type")
	}

	return node, nil
}

// fetchNodeFromAPI 从K8s API获取单个节点信息（无缓存）
func (s *Service) fetchNodeFromAPI(clusterName, nodeName string) (*NodeInfo, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		s.logger.Errorf("Failed to get node %s for cluster %s: %v", nodeName, clusterName, err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	nodeInfo := s.nodeToNodeInfo(node)

	// 尝试获取单个节点的资源使用情况
	s.enrichNodeWithMetrics(clusterName, &nodeInfo)

	return &nodeInfo, nil
}

// UpdateNodeLabels 更新节点标签（带重试机制）
func (s *Service) UpdateNodeLabels(clusterName string, req LabelUpdateRequest) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// 获取当前节点（每次重试都重新获取最新版本）
		node, err := client.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
		if err != nil {
			cancel()
			s.logger.Errorf("Failed to get node %s (attempt %d/%d): %v", req.NodeName, attempt+1, maxRetries+1, err)
			if attempt == maxRetries {
				return fmt.Errorf("failed to get node after %d attempts: %w", maxRetries+1, err)
			}
			// 等待后重试
			s.waitWithBackoff(attempt, baseDelay)
			continue
		}

		// 更新标签
		if attempt == 0 {
			s.logger.Infof("Current node labels before update: %+v", node.Labels)
			s.logger.Infof("Received labels to apply: %+v", req.Labels)
		} else {
			s.logger.Infof("Retry attempt %d: Current node labels: %+v", attempt+1, node.Labels)
		}

		// 不要完全清空标签，因为这会删除系统必需的标签
		// 相反，我们直接应用传递过来的标签映射，它们已经在上层服务中被正确处理了
		if node.Labels == nil {
			node.Labels = make(map[string]string)
		}

		// 直接应用传递的标签（标签服务已经处理了保留系统标签的逻辑）
		node.Labels = req.Labels

		s.logger.Infof("Node labels after update: %+v", node.Labels)

		// 尝试更新节点
		_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		cancel()

		if err != nil {
			// 检查是否是资源版本冲突错误
			if strings.Contains(err.Error(), "the object has been modified") && attempt < maxRetries {
				s.logger.Warningf("Resource version conflict for node %s (attempt %d/%d), retrying: %v",
					req.NodeName, attempt+1, maxRetries+1, err)

				// 指数退避等待
				s.waitWithBackoff(attempt, baseDelay)
				continue
			}

			s.logger.Errorf("Failed to update node labels for %s (attempt %d/%d): %v", req.NodeName, attempt+1, maxRetries+1, err)
			if attempt == maxRetries {
				return fmt.Errorf("failed to update node labels after %d attempts: %w", maxRetries+1, err)
			}

			// 其他错误也尝试重试
			s.waitWithBackoff(attempt, baseDelay)
			continue
		}

		// 成功更新
		if attempt > 0 {
			s.logger.Infof("Successfully updated labels for node %s in cluster %s (succeeded after %d retries)",
				req.NodeName, clusterName, attempt)
		} else {
			s.logger.Infof("Successfully updated labels for node %s in cluster %s", req.NodeName, clusterName)
		}

		// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
		return nil
	}

	return fmt.Errorf("failed to update node labels after %d attempts", maxRetries+1)
}

// waitWithBackoff 实现指数退避等待
func (s *Service) waitWithBackoff(attempt int, baseDelay time.Duration) {
	// 指数退避: baseDelay * (2^attempt) + 随机抖动
	delay := baseDelay * time.Duration(1<<uint(attempt))

	// 添加最大延迟限制（最多等待2秒）
	maxDelay := 2 * time.Second
	if delay > maxDelay {
		delay = maxDelay
	}

	// 添加随机抖动 (±25%)
	jitterPercent := (float64(time.Now().UnixNano()%1000) / 1000.0 * 2.0) - 1.0 // -1 到 1 之间的随机数
	jitter := time.Duration(float64(delay) * 0.25 * jitterPercent)
	delay = delay + jitter

	s.logger.Infof("Waiting %v before retry (attempt %d)", delay, attempt+1)
	time.Sleep(delay)
}

// UpdateNodeTaints 更新节点污点（带重试机制）
func (s *Service) UpdateNodeTaints(clusterName string, req TaintUpdateRequest) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	const maxRetries = 5
	const baseDelay = 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

		// 获取当前节点（每次重试都重新获取最新版本）
		node, err := client.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
		if err != nil {
			cancel()
			s.logger.Errorf("Failed to get node %s (attempt %d/%d): %v", req.NodeName, attempt+1, maxRetries+1, err)
			if attempt == maxRetries {
				return fmt.Errorf("failed to get node after %d attempts: %w", maxRetries+1, err)
			}
			// 等待后重试
			s.waitWithBackoff(attempt, baseDelay)
			continue
		}

		// 转换污点
		var taints []corev1.Taint
		for _, taint := range req.Taints {
			k8sTaint := corev1.Taint{
				Key:    taint.Key,
				Value:  taint.Value,
				Effect: corev1.TaintEffect(taint.Effect),
			}
			if taint.TimeAdded != nil {
				k8sTaint.TimeAdded = &metav1.Time{Time: *taint.TimeAdded}
			}
			taints = append(taints, k8sTaint)
		}

		// 更新污点
		node.Spec.Taints = taints

		// 尝试更新节点
		_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		cancel()

		if err != nil {
			// 检查是否是资源版本冲突错误
			if strings.Contains(err.Error(), "the object has been modified") && attempt < maxRetries {
				s.logger.Warningf("Resource version conflict for node %s taints (attempt %d/%d), retrying: %v",
					req.NodeName, attempt+1, maxRetries+1, err)

				// 指数退避等待
				s.waitWithBackoff(attempt, baseDelay)
				continue
			}

			s.logger.Errorf("Failed to update node taints for %s (attempt %d/%d): %v", req.NodeName, attempt+1, maxRetries+1, err)
			if attempt == maxRetries {
				return fmt.Errorf("failed to update node taints after %d attempts: %w", maxRetries+1, err)
			}

			// 其他错误也尝试重试
			s.waitWithBackoff(attempt, baseDelay)
			continue
		}

		// 成功更新
		if attempt > 0 {
			s.logger.Infof("Successfully updated taints for node %s in cluster %s (succeeded after %d retries)",
				req.NodeName, clusterName, attempt)
		} else {
			s.logger.Infof("Successfully updated taints for node %s in cluster %s", req.NodeName, clusterName)
		}

		// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
		return nil
	}

	return fmt.Errorf("failed to update node taints after %d attempts", maxRetries+1)
}

// CordonNode 禁止调度节点（仅设置不可调度，不删除pod）
func (s *Service) CordonNode(clusterName, nodeName string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// 如果节点已经不可调度，直接返回成功
	if node.Spec.Unschedulable {
		s.logger.Infof("Node %s in cluster %s is already cordoned", nodeName, clusterName)
		return nil
	}

	node.Spec.Unschedulable = true
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	s.logger.Infof("Successfully cordoned node %s in cluster %s", nodeName, clusterName)

	// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
	return nil
}

// CordonNodeWithReason 禁止调度节点并添加原因注释（带重试机制处理资源冲突）
func (s *Service) CordonNodeWithReason(clusterName, nodeName, reason string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用重试机制处理资源冲突错误
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 使用指数退避策略
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			s.logger.Infof("Retrying cordon node %s (attempt %d/%d) after %v", nodeName, attempt+1, maxRetries+1, backoff)
			time.Sleep(backoff)
		}

		// 每次重试都重新获取节点以获取最新的 ResourceVersion
		node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			lastErr = fmt.Errorf("failed to get node: %w", err)
			continue
		}

		// 设置不可调度
		node.Spec.Unschedulable = true

		// 添加或更新annotation
		if node.Annotations == nil {
			node.Annotations = make(map[string]string)
		}

		// 添加禁止调度的原因注释
		if reason != "" {
			node.Annotations["deeproute.cn/kube-node-mgr"] = reason
		}

		// 添加禁止调度的时间戳（ISO 8601格式）
		node.Annotations["deeproute.cn/kube-node-mgr-timestamp"] = time.Now().UTC().Format(time.RFC3339)

		_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			// 检查是否是资源冲突错误（状态码 409）
			if strings.Contains(err.Error(), "the object has been modified") || 
			   strings.Contains(err.Error(), "Operation cannot be fulfilled") {
				lastErr = err
				s.logger.Warningf("Node %s resource conflict on attempt %d: %v", nodeName, attempt+1, err)
				continue // 重试
			}
			// 其他类型的错误直接返回
			return fmt.Errorf("failed to cordon node with reason: %w", err)
		}

		// 成功
		s.logger.Infof("Successfully cordoned node %s in cluster %s with reason: %s (attempt %d/%d)", nodeName, clusterName, reason, attempt+1, maxRetries+1)

		// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
		return nil
	}

	// 所有重试都失败了
	return fmt.Errorf("failed to cordon node after %d attempts: %w", maxRetries+1, lastErr)
}

// UncordonNode 取消节点驱逐（带重试机制处理资源冲突）
func (s *Service) UncordonNode(clusterName, nodeName string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 使用重试机制处理资源冲突错误
	maxRetries := 3
	var lastErr error
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 使用指数退避策略
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			s.logger.Infof("Retrying uncordon node %s (attempt %d/%d) after %v", nodeName, attempt+1, maxRetries+1, backoff)
			time.Sleep(backoff)
		}

		// 每次重试都重新获取节点以获取最新的 ResourceVersion
		node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			lastErr = fmt.Errorf("failed to get node: %w", err)
			continue
		}

		// 如果节点已经可调度，直接返回成功
		if !node.Spec.Unschedulable {
			s.logger.Infof("Node %s in cluster %s is already uncordoned", nodeName, clusterName)
			// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
			return nil
		}

		node.Spec.Unschedulable = false

		// 删除相关的annotations
		if node.Annotations != nil {
			delete(node.Annotations, "deeproute.cn/kube-node-mgr")
			delete(node.Annotations, "deeproute.cn/kube-node-mgr-timestamp")
		}

		_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			// 检查是否是资源冲突错误（状态码 409）
			if strings.Contains(err.Error(), "the object has been modified") || 
			   strings.Contains(err.Error(), "Operation cannot be fulfilled") {
				lastErr = err
				s.logger.Warningf("Node %s resource conflict on attempt %d: %v", nodeName, attempt+1, err)
				continue // 重试
			}
			// 其他类型的错误直接返回
			return fmt.Errorf("failed to uncordon node: %w", err)
		}

		// 成功
		s.logger.Infof("Successfully uncordoned node %s in cluster %s (attempt %d/%d)", nodeName, clusterName, attempt+1, maxRetries+1)

		// 注意：使用智能缓存 + Informer 后，缓存会自动更新，无需手动清除
		return nil
	}

	// 所有重试都失败了
	return fmt.Errorf("failed to uncordon node after %d attempts: %w", maxRetries+1, lastErr)
}

// GetNodeCordonInfo 获取节点的禁止调度信息
func (s *Service) GetNodeCordonInfo(clusterName, nodeName string) (map[string]interface{}, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	info := make(map[string]interface{})
	info["cordoned"] = node.Spec.Unschedulable

	if node.Spec.Unschedulable {
		// 首先从我们的自定义 annotations 获取信息
		if node.Annotations != nil {
			if reason, exists := node.Annotations["deeproute.cn/kube-node-mgr"]; exists {
				info["reason"] = reason
			}
			if timestamp, exists := node.Annotations["deeproute.cn/kube-node-mgr-timestamp"]; exists {
				info["timestamp"] = timestamp
			}
		}

		// 如果没有自定义 timestamp，但有 reason annotation，尝试从 unschedulable taint 获取 timeAdded
		if _, hasCustomTimestamp := info["timestamp"]; !hasCustomTimestamp {
			// 只有在存在我们的 reason annotation 时才去获取 taint 时间戳
			if _, hasReason := info["reason"]; hasReason {
				for _, taint := range node.Spec.Taints {
					if taint.Key == "node.kubernetes.io/unschedulable" && taint.TimeAdded != nil {
						info["timestamp"] = taint.TimeAdded.Format(time.RFC3339)
						info["timestamp_source"] = "kubernetes_taint"
						s.logger.Infof("Found unschedulable taint timestamp for node %s with deeproute annotation: %s", node.Name, taint.TimeAdded.Format(time.RFC3339))
						break
					}
				}
			}
		} else {
			info["timestamp_source"] = "kubectl_plugin"
		}
	}

	return info, nil
}

// DrainNode 驱逐节点上的Pod（类似kubectl drain）
func (s *Service) DrainNode(clusterName, nodeName, reason string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second) // 5分钟超时
	defer cancel()

	s.logger.Infof("Starting to drain node %s in cluster %s", nodeName, clusterName)

	// 首先cordon节点，防止新的Pod调度到此节点
	if err := s.CordonNodeWithReason(clusterName, nodeName, reason); err != nil {
		return fmt.Errorf("failed to cordon node before draining: %w", err)
	}

	// 获取节点上的所有Pod
	fieldSelector := fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName})
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fieldSelector.String(),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	s.logger.Infof("Found %d pods on node %s", len(pods.Items), nodeName)

	// 过滤需要驱逐的Pod
	var podsToEvict []corev1.Pod
	for _, pod := range pods.Items {
		if s.shouldEvictPod(&pod) {
			podsToEvict = append(podsToEvict, pod)
		}
	}

	s.logger.Infof("Will evict %d pods from node %s", len(podsToEvict), nodeName)

	// 如果没有需要驱逐的Pod，直接返回成功
	if len(podsToEvict) == 0 {
		s.logger.Infof("No pods to evict on node %s", nodeName)
		return nil
	}

	// 驱逐Pod
	var evictionErrors []string
	for _, pod := range podsToEvict {
		if err := s.evictPod(ctx, client, &pod); err != nil {
			evictionErrors = append(evictionErrors, fmt.Sprintf("Pod %s/%s: %v", pod.Namespace, pod.Name, err))
			s.logger.Errorf("Failed to evict pod %s/%s: %v", pod.Namespace, pod.Name, err)
		} else {
			s.logger.Infof("Successfully evicted pod %s/%s", pod.Namespace, pod.Name)
		}
	}

	// 等待Pod驱逐完成
	if len(evictionErrors) == 0 {
		if err := s.waitForPodsEvicted(ctx, client, nodeName, podsToEvict); err != nil {
			s.logger.Warningf("Some pods may still be terminating on node %s: %v", nodeName, err)
		}
	}

	if len(evictionErrors) > 0 {
		return fmt.Errorf("failed to evict some pods: %s", strings.Join(evictionErrors, "; "))
	}

	s.logger.Infof("Successfully drained node %s in cluster %s", nodeName, clusterName)
	return nil
}

// shouldEvictPod 判断是否应该驱逐Pod
func (s *Service) shouldEvictPod(pod *corev1.Pod) bool {
	// 跳过已经完成或正在删除的Pod
	if pod.DeletionTimestamp != nil || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
		return false
	}

	// 跳过DaemonSet管理的Pod（根据requirement要求忽略daemonsets）
	if s.isDaemonSetPod(pod) {
		s.logger.Infof("Skipping DaemonSet pod %s/%s", pod.Namespace, pod.Name)
		return false
	}

	// 跳过静态Pod（由kubelet直接管理）
	if s.isStaticPod(pod) {
		s.logger.Infof("Skipping static pod %s/%s", pod.Namespace, pod.Name)
		return false
	}

	return true
}

// isDaemonSetPod 检查Pod是否由DaemonSet管理
func (s *Service) isDaemonSetPod(pod *corev1.Pod) bool {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "DaemonSet" {
			return true
		}
	}
	return false
}

// isStaticPod 检查是否为静态Pod
func (s *Service) isStaticPod(pod *corev1.Pod) bool {
	for _, ownerRef := range pod.OwnerReferences {
		if ownerRef.Kind == "Node" {
			return true
		}
	}
	// 静态Pod通常在mirror pod annotation中有标记
	if pod.Annotations != nil {
		if _, exists := pod.Annotations["kubernetes.io/config.mirror"]; exists {
			return true
		}
	}
	return false
}

// evictPod 驱逐单个Pod
func (s *Service) evictPod(ctx context.Context, client kubernetes.Interface, pod *corev1.Pod) error {
	// 检查是否支持Eviction API（优先使用）
	eviction := &policyv1.Eviction{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pod.Name,
			Namespace: pod.Namespace,
		},
		DeleteOptions: &metav1.DeleteOptions{},
	}

	err := client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, eviction)
	if err == nil {
		return nil
	}

	// 如果Eviction API不可用，回退到直接删除Pod
	if apierrors.IsNotFound(err) || apierrors.IsTooManyRequests(err) {
		s.logger.Warningf("Eviction API failed for pod %s/%s (%v), falling back to direct deletion",
			pod.Namespace, pod.Name, err)

		deleteOptions := metav1.DeleteOptions{
			GracePeriodSeconds: func() *int64 { var i int64 = 30; return &i }(), // 30秒优雅关闭
		}

		return client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, deleteOptions)
	}

	return err
}

// waitForPodsEvicted 等待Pod驱逐完成
func (s *Service) waitForPodsEvicted(ctx context.Context, client kubernetes.Interface, nodeName string, podsToWait []corev1.Pod) error {
	return wait.PollImmediate(5*time.Second, 120*time.Second, func() (bool, error) {
		fieldSelector := fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName})
		currentPods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			FieldSelector: fieldSelector.String(),
		})
		if err != nil {
			return false, err
		}

		// 检查是否还有需要等待的Pod
		waitingPods := make(map[string]bool)
		for _, pod := range podsToWait {
			waitingPods[pod.Namespace+"/"+pod.Name] = true
		}

		stillWaiting := 0
		for _, pod := range currentPods.Items {
			podKey := pod.Namespace + "/" + pod.Name
			if waitingPods[podKey] && pod.DeletionTimestamp == nil {
				stillWaiting++
			}
		}

		s.logger.Infof("Still waiting for %d pods to be evicted from node %s", stillWaiting, nodeName)
		return stillWaiting == 0, nil
	})
}

// nodeToNodeInfo 转换节点信息
func (s *Service) nodeToNodeInfo(node *corev1.Node) NodeInfo {
	// 获取节点角色
	roles := s.getNodeRoles(node)

	// 获取IP地址
	internalIP, externalIP := s.getNodeIPs(node)

	// 获取节点状态
	status := s.getNodeStatus(node)

	// 转换污点
	var taints []TaintInfo
	for _, taint := range node.Spec.Taints {
		taintInfo := TaintInfo{
			Key:    taint.Key,
			Value:  taint.Value,
			Effect: string(taint.Effect),
		}
		if taint.TimeAdded != nil {
			taintInfo.TimeAdded = &taint.TimeAdded.Time
		}
		taints = append(taints, taintInfo)
	}

	// 转换条件
	var conditions []NodeCondition
	for _, condition := range node.Status.Conditions {
		conditions = append(conditions, NodeCondition{
			Type:               string(condition.Type),
			Status:             string(condition.Status),
			LastHeartbeatTime:  condition.LastHeartbeatTime.Time,
			LastTransitionTime: condition.LastTransitionTime.Time,
			Reason:             condition.Reason,
			Message:            condition.Message,
		})
	}

	// 获取禁止调度原因（如果节点被禁止调度）
	var unschedulableReason string
	if node.Spec.Unschedulable && node.Annotations != nil {
		if reason, exists := node.Annotations["deeproute.cn/kube-node-mgr"]; exists && reason != "" {
			unschedulableReason = reason
		}
	}

	return NodeInfo{
		Name:                node.Name,
		Status:              status,
		Schedulable:         !node.Spec.Unschedulable,
		UnschedulableReason: unschedulableReason,
		Roles:               roles,
		Age:                 s.getAge(node.CreationTimestamp.Time),
		Version:             node.Status.NodeInfo.KubeletVersion,
		InternalIP:          internalIP,
		ExternalIP:          externalIP,
		OS:                  node.Status.NodeInfo.OperatingSystem,
		OSImage:             node.Status.NodeInfo.OSImage,
		KernelVersion:       node.Status.NodeInfo.KernelVersion,
		ContainerRuntime:    node.Status.NodeInfo.ContainerRuntimeVersion,
		Capacity: ResourceInfo{
			CPU:    s.formatCPU(node.Status.Capacity.Cpu().MilliValue()),
			Memory: s.formatMemory(node.Status.Capacity.Memory().Value()),
			Pods:   node.Status.Capacity.Pods().String(),
			GPU:    s.extractGPUResources(node.Status.Capacity),
		},
		Allocatable: ResourceInfo{
			CPU:    s.formatCPU(node.Status.Allocatable.Cpu().MilliValue()),
			Memory: s.formatMemory(node.Status.Allocatable.Memory().Value()),
			Pods:   node.Status.Allocatable.Pods().String(),
			GPU:    s.extractGPUResources(node.Status.Allocatable),
		},
		Labels:     node.Labels,
		Taints:     taints,
		Conditions: conditions,
		CreatedAt:  node.CreationTimestamp.Time,
	}
}

// getNodeRoles 获取节点角色
func (s *Service) getNodeRoles(node *corev1.Node) []string {
	var roles []string
	for label := range node.Labels {
		if strings.HasPrefix(label, "node-role.kubernetes.io/") {
			role := strings.TrimPrefix(label, "node-role.kubernetes.io/")
			if role == "" {
				role = "master"
			}
			roles = append(roles, role)
		}
	}
	if len(roles) == 0 {
		roles = []string{"worker"}
	}
	return roles
}

// getNodeIPs 获取节点IP地址
func (s *Service) getNodeIPs(node *corev1.Node) (internalIP, externalIP string) {
	for _, address := range node.Status.Addresses {
		switch address.Type {
		case corev1.NodeInternalIP:
			internalIP = address.Address
		case corev1.NodeExternalIP:
			externalIP = address.Address
		}
	}
	return
}

// getNodeStatus 获取节点状态
func (s *Service) getNodeStatus(node *corev1.Node) string {
	// 首先检查节点的 Ready 状态
	var readyStatus string
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				readyStatus = "Ready"
			} else {
				readyStatus = "NotReady"
			}
			break
		}
	}

	// 如果没有找到 Ready condition，默认为 Unknown
	if readyStatus == "" {
		readyStatus = "Unknown"
	}

	// 如果节点被禁止调度，添加 SchedulingDisabled 标记
	if node.Spec.Unschedulable {
		return readyStatus + ",SchedulingDisabled"
	}

	return readyStatus
}

// getAge 计算年龄
func (s *Service) getAge(creationTime time.Time) string {
	duration := time.Since(creationTime)
	days := int(duration.Hours() / 24)
	if days > 0 {
		return fmt.Sprintf("%dd", days)
	}
	hours := int(duration.Hours())
	if hours > 0 {
		return fmt.Sprintf("%dh", hours)
	}
	minutes := int(duration.Minutes())
	return fmt.Sprintf("%dm", minutes)
}

// TestConnection 测试集群连接
func (s *Service) TestConnection(kubeconfig string) error {
	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	config.Timeout = 10 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 尝试多种权限验证方法，从最基础开始
	// 1. 首先尝试获取API版本信息（几乎所有用户都有此权限）
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}

	// 2. 尝试列出节点（需要nodes权限，可选）
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		s.logger.Warningf("Cannot list nodes with this kubeconfig (limited permissions): %v", err)
		// 不返回错误，只是记录警告
		// 对于只有特定权限的service account，这是正常的

		// 3. 尝试列出命名空间（更基础的权限）
		_, err = clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			s.logger.Warningf("Cannot list namespaces with this kubeconfig: %v", err)
			// 仍然不返回错误，继续验证其他权限
		}
	}

	return nil
}

// CheckClusterConnection 检查集群连接状态
func (s *Service) CheckClusterConnection(clusterName string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	// 尝试简单的API调用来验证连接
	_, err = client.Discovery().ServerVersion()
	if err != nil {
		s.logger.Errorf("Failed to check cluster connection for %s: %v", clusterName, err)
		return fmt.Errorf("cluster connection check failed for %s: %w", clusterName, err)
	}

	s.logger.Infof("Cluster connection check successful for %s", clusterName)
	return nil
}

// extractGPUResources 提取GPU资源信息
func (s *Service) extractGPUResources(resources corev1.ResourceList) map[string]string {
	gpuResources := make(map[string]string)

	// 常见的GPU资源类型
	gpuResourceKeys := []string{
		"nvidia.com/gpu",
		"amd.com/gpu",
		"intel.com/gpu",
		"gpu",
		"kubernetes.io/gpu",
	}

	// 首先检查常见的GPU资源类型
	for _, key := range gpuResourceKeys {
		if quantity, exists := resources[corev1.ResourceName(key)]; exists && !quantity.IsZero() {
			gpuResources[key] = quantity.String()
		}
	}

	// 检查所有以 nvidia.com/mig- 开头的资源（Multi-Instance GPU）
	for resourceName, quantity := range resources {
		resourceKey := string(resourceName)
		if strings.HasPrefix(resourceKey, "nvidia.com/mig-") && !quantity.IsZero() {
			gpuResources[resourceKey] = quantity.String()
		}
	}

	return gpuResources
}

// enrichNodesWithMetrics 为节点列表添加资源使用情况
// 大规模集群优化（v3 - Pod Informer）：
// - 优先使用 Pod Informer 缓存（实时，<1ms响应）
// - 降级策略：Informer 未就绪时使用旧的分页查询+缓存方案
// - CPU/内存指标保持同步获取（响应快）
func (s *Service) enrichNodesWithMetrics(clusterName string, nodes []NodeInfo) {
	metricsClient, err := s.getMetricsClient(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get metrics client for cluster %s: %v", clusterName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取所有节点的metrics
	nodeMetricsList, err := metricsClient.MetricsV1beta1().NodeMetricses().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Warningf("Failed to get node metrics for cluster %s: %v", clusterName, err)
		return
	}

	// 创建metrics映射
	metricsMap := make(map[string]*metricsv1beta1.NodeMetrics)
	for i := range nodeMetricsList.Items {
		metric := &nodeMetricsList.Items[i]
		metricsMap[metric.Name] = metric
	}

	// 批量获取所有节点的 Pod 数量（优先使用 Informer）
	nodeNames := make([]string, len(nodes))
	for i := range nodes {
		nodeNames[i] = nodes[i].Name
	}
	
	podCounts := s.getPodCountsWithFallback(clusterName, nodeNames)

	// 为每个节点添加使用情况
	for i := range nodes {
		if metric, exists := metricsMap[nodes[i].Name]; exists {
			podCount := podCounts[nodes[i].Name]
			nodes[i].Usage = &ResourceUsageInfo{
				CPU:    s.formatCPU(metric.Usage.Cpu().MilliValue()),
				Memory: s.formatMemory(metric.Usage.Memory().Value()),
				Pods:   fmt.Sprintf("%d", podCount),
			}
		}
	}

	s.logger.Infof("Successfully enriched %d nodes with metrics for cluster %s", len(nodes), clusterName)
}

// getPodCountsWithFallback 获取 Pod 数量（带降级策略）
// 优先级：Pod Informer 缓存 > 旧的分页查询+缓存方案
func (s *Service) getPodCountsWithFallback(clusterName string, nodeNames []string) map[string]int {
	// 策略1：尝试从 Pod Informer 缓存获取（最优，实时且快速）
	if s.podCountCache != nil && s.podCountCache.IsReady(clusterName) {
		podCounts := s.podCountCache.GetAllNodePodCounts(clusterName)
		if len(podCounts) > 0 {
			s.logger.Debugf("Using Pod Informer cache for cluster %s (fast path)", clusterName)
			return podCounts
		}
	}

	// 策略2：降级到旧的分页查询+缓存方案
	s.logger.Debugf("Pod Informer not ready for cluster %s, falling back to paginated query", clusterName)
	
	fetchFunc := func() map[string]int {
		return s.getNodesPodCounts(clusterName, nodeNames)
	}
	return s.cache.GetPodCounts(clusterName, nodeNames, fetchFunc)
}

// enrichNodeWithMetrics 为单个节点添加资源使用情况
func (s *Service) enrichNodeWithMetrics(clusterName string, node *NodeInfo) {
	metricsClient, err := s.getMetricsClient(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get metrics client for cluster %s: %v", clusterName, err)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 获取单个节点的metrics
	nodeMetrics, err := metricsClient.MetricsV1beta1().NodeMetricses().Get(ctx, node.Name, metav1.GetOptions{})
	if err != nil {
		s.logger.Warningf("Failed to get metrics for node %s in cluster %s: %v", node.Name, clusterName, err)
		return
	}

	// 获取节点上的 Pod 数量
	podCount, err := s.getNodePodCount(clusterName, node.Name)
	if err != nil {
		s.logger.Warningf("Failed to get pod count for node %s in cluster %s: %v", node.Name, clusterName, err)
		podCount = 0 // 如果获取失败，设置为 0
	}

	// 添加使用情况
	node.Usage = &ResourceUsageInfo{
		CPU:    s.formatCPU(nodeMetrics.Usage.Cpu().MilliValue()),
		Memory: s.formatMemory(nodeMetrics.Usage.Memory().Value()),
		Pods:   fmt.Sprintf("%d", podCount),
	}

	s.logger.Infof("Successfully enriched node %s with metrics for cluster %s", node.Name, clusterName)
}

// formatCPU 格式化 CPU 资源，将毫核转换为核心数
// milliValue: CPU 的毫核数（1核 = 1000毫核）
func (s *Service) formatCPU(milliValue int64) string {
	// 将毫核转换为核心数，保留2位小数
	cores := float64(milliValue) / 1000.0

	// 如果小于0.01核，显示为毫核
	if cores < 0.01 {
		return fmt.Sprintf("%dm", milliValue)
	}

	// 如果是整数核心，不显示小数
	if cores == float64(int64(cores)) {
		return fmt.Sprintf("%d", int64(cores))
	}

	// 保留2位小数
	return fmt.Sprintf("%.2f", cores)
}

// formatMemory 格式化内存资源，将字节转换为人类可读格式
// bytes: 内存字节数
func (s *Service) formatMemory(bytes int64) string {
	const (
		Ki = 1024
		Mi = 1024 * Ki
		Gi = 1024 * Mi
		Ti = 1024 * Gi
	)

	switch {
	case bytes >= Ti:
		value := float64(bytes) / float64(Ti)
		if value == float64(int64(value)) {
			return fmt.Sprintf("%dTi", int64(value))
		}
		return fmt.Sprintf("%.2fTi", value)
	case bytes >= Gi:
		value := float64(bytes) / float64(Gi)
		if value == float64(int64(value)) {
			return fmt.Sprintf("%dGi", int64(value))
		}
		return fmt.Sprintf("%.2fGi", value)
	case bytes >= Mi:
		value := float64(bytes) / float64(Mi)
		if value == float64(int64(value)) {
			return fmt.Sprintf("%dMi", int64(value))
		}
		return fmt.Sprintf("%.2fMi", value)
	case bytes >= Ki:
		value := float64(bytes) / float64(Ki)
		if value == float64(int64(value)) {
			return fmt.Sprintf("%dKi", int64(value))
		}
		return fmt.Sprintf("%.2fKi", value)
	default:
		return fmt.Sprintf("%d", bytes)
	}
}

// getNodePodCount 获取节点上运行的 Pod 数量（Non-terminated Pods）
func (s *Service) getNodePodCount(clusterName, nodeName string) (int, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return 0, err
	}

	// 针对大规模集群增加超时时间到 20 秒
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	// 使用 FieldSelector 获取指定节点上的 Pods
	// 排除已终止的 Pod (status.phase != Succeeded && status.phase != Failed)
	podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fields.SelectorFromSet(fields.Set{"spec.nodeName": nodeName}).String(),
	})
	if err != nil {
		return 0, fmt.Errorf("failed to list pods on node %s: %w", nodeName, err)
	}

	// 统计非终止状态的 Pod
	nonTerminatedCount := 0
	for _, pod := range podList.Items {
		// 排除 Succeeded 和 Failed 状态的 Pod
		if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
			nonTerminatedCount++
		}
	}

	return nonTerminatedCount, nil
}

// getNodesPodCounts 批量获取多个节点的 Pod 数量（使用分页查询优化）
// 大规模集群优化（v2）：
// - 页面大小从 500 增加到 1000（减少请求次数）
// - 单页超时从 30 秒增加到 60 秒（更宽松的超时策略）
// - 支持 partial data 早期返回（遇到错误时返回已统计的部分结果）
func (s *Service) getNodesPodCounts(clusterName string, nodeNames []string) map[string]int {
	client, err := s.getClient(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get client for cluster %s: %v", clusterName, err)
		return make(map[string]int)
	}

	// 初始化计数器
	podCounts := make(map[string]int)
	nodeSet := make(map[string]bool)
	for _, node := range nodeNames {
		podCounts[node] = 0
		nodeSet[node] = true
	}

	// 使用分页查询，避免一次性加载大量 Pod 导致超时
	// 优化：页面大小从 500 增加到 1000，减少请求次数
	const pageSize = 1000
	continueToken := ""
	totalPods := 0
	pageCount := 0
	maxPages := 50 // 最多查询 50 页（50k pods），避免无限循环

	s.logger.Infof("Starting paginated pod count for cluster %s with %d nodes", clusterName, len(nodeNames))

	for pageCount < maxPages {
		pageCount++
		// 优化：每次分页请求超时从 30 秒增加到 60 秒
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)

		podList, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
			Limit:    pageSize,
			Continue: continueToken,
		})
		cancel()

		if err != nil {
			s.logger.Warningf("Failed to list pods (page %d) for cluster %s: %v", pageCount, clusterName, err)
			// Partial data 策略：即使部分失败，也返回已统计的结果
			if totalPods > 0 {
				s.logger.Infof("Returning partial pod count data: cluster=%s, pods=%d, pages=%d", 
					clusterName, totalPods, pageCount-1)
			}
			break
		}

		// 统计此批次的 Pod
		for _, pod := range podList.Items {
			// 只统计非终止状态的 Pod
			if pod.Status.Phase != corev1.PodSucceeded && pod.Status.Phase != corev1.PodFailed {
				if nodeSet[pod.Spec.NodeName] {
					podCounts[pod.Spec.NodeName]++
					totalPods++
				}
			}
		}

		s.logger.Debugf("Processed page %d for cluster %s: %d pods in this page", pageCount, clusterName, len(podList.Items))

		// 检查是否还有更多数据
		if podList.Continue == "" {
			break
		}
		continueToken = podList.Continue
	}

	if pageCount >= maxPages {
		s.logger.Warningf("Reached max pages limit (%d) for cluster %s, pod count may be incomplete", 
			maxPages, clusterName)
	}

	s.logger.Infof("Completed paginated pod count for cluster %s: %d total active pods across %d pages", 
		clusterName, totalPods, pageCount)

	return podCounts
}

// InvalidateClusterCache 清除指定集群的所有缓存
func (s *Service) InvalidateClusterCache(clusterName string) {
	s.cache.InvalidateCluster(clusterName)
}
