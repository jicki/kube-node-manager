package k8s

import (
	"context"
	"fmt"
	"kube-node-manager/pkg/logger"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// Service Kubernetes客户端服务
type Service struct {
	logger  *logger.Logger
	clients map[string]*kubernetes.Clientset
	mu      sync.RWMutex
}

// NodeInfo Kubernetes节点信息
type NodeInfo struct {
	Name             string            `json:"name"`
	Status           string            `json:"status"`
	Schedulable      bool              `json:"schedulable"`
	Roles            []string          `json:"roles"`
	Age              string            `json:"age"`
	Version          string            `json:"version"`
	InternalIP       string            `json:"internal_ip"`
	ExternalIP       string            `json:"external_ip"`
	OS               string            `json:"os"`
	OSImage          string            `json:"os_image"`
	KernelVersion    string            `json:"kernel_version"`
	ContainerRuntime string            `json:"container_runtime"`
	Capacity         ResourceInfo      `json:"capacity"`
	Allocatable      ResourceInfo      `json:"allocatable"`
	Labels           map[string]string `json:"labels"`
	Taints           []TaintInfo       `json:"taints"`
	Conditions       []NodeCondition   `json:"conditions"`
	CreatedAt        time.Time         `json:"created_at"`
}

// ResourceInfo 资源信息
type ResourceInfo struct {
	CPU    string            `json:"cpu"`
	Memory string            `json:"memory"`
	Pods   string            `json:"pods"`
	GPU    map[string]string `json:"gpu,omitempty"` // 支持多种GPU类型
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
func NewService(logger *logger.Logger) *Service {
	return &Service{
		logger:  logger,
		clients: make(map[string]*kubernetes.Clientset),
	}
}

// CreateClient 根据kubeconfig创建Kubernetes客户端
func (s *Service) CreateClient(clusterName, kubeconfig string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	config, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		s.logger.Error("Failed to parse kubeconfig for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to parse kubeconfig: %w", err)
	}

	// 设置超时
	config.Timeout = 30 * time.Second

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		s.logger.Error("Failed to create Kubernetes client for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用基础的API版本检查来验证连接
	_, err = clientset.Discovery().ServerVersion()
	if err != nil {
		s.logger.Error("Failed to test connection for cluster %s: %v", clusterName, err)
		return fmt.Errorf("failed to connect to kubernetes cluster: %w", err)
	}

	// 尝试检查节点权限（可选）
	_, err = clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{Limit: 1})
	if err != nil {
		s.logger.Warning("Limited permissions for cluster %s: cannot list nodes: %v", clusterName, err)
		// 不阻止客户端创建，只是记录警告
	}

	s.clients[clusterName] = clientset
	s.logger.Info("Successfully created Kubernetes client for cluster: %s", clusterName)
	return nil
}

// RemoveClient 移除Kubernetes客户端
func (s *Service) RemoveClient(clusterName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.clients, clusterName)
	s.logger.Info("Removed Kubernetes client for cluster: %s", clusterName)
}

// getClient 获取指定集群的客户端
func (s *Service) getClient(clusterName string) (*kubernetes.Clientset, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clients[clusterName]
	if !exists {
		return nil, fmt.Errorf("kubernetes client not found for cluster: %s", clusterName)
	}
	return client, nil
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
		s.logger.Error("Failed to get server version for cluster %s: %v", clusterName, err)
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

// ListNodes 获取节点列表
func (s *Service) ListNodes(clusterName string) ([]NodeInfo, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodeList, err := client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		s.logger.Error("Failed to list nodes for cluster %s: %v", clusterName, err)
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var nodes []NodeInfo
	for _, node := range nodeList.Items {
		nodeInfo := s.nodeToNodeInfo(&node)
		nodes = append(nodes, nodeInfo)
	}

	return nodes, nil
}

// GetNode 获取单个节点信息
func (s *Service) GetNode(clusterName, nodeName string) (*NodeInfo, error) {
	client, err := s.getClient(clusterName)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("Failed to get node %s for cluster %s: %v", nodeName, clusterName, err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	nodeInfo := s.nodeToNodeInfo(node)
	return &nodeInfo, nil
}

// UpdateNodeLabels 更新节点标签
func (s *Service) UpdateNodeLabels(clusterName string, req LabelUpdateRequest) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取当前节点
	node, err := client.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("Failed to get node %s: %v", req.NodeName, err)
		return fmt.Errorf("failed to get node: %w", err)
	}

	// 更新标签
	s.logger.Info("Current node labels before update: %+v", node.Labels)
	s.logger.Info("Received labels to apply: %+v", req.Labels)
	
	if node.Labels == nil {
		node.Labels = make(map[string]string)
	}

	for key, value := range req.Labels {
		if value == "" {
			// 删除标签
			s.logger.Info("Deleting label %s (empty value)", key)
			delete(node.Labels, key)
		} else {
			// 添加或更新标签
			s.logger.Info("Setting label %s = %s", key, value)
			node.Labels[key] = value
		}
	}
	
	s.logger.Info("Node labels after update: %+v", node.Labels)

	// 更新节点
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("Failed to update node labels for %s: %v", req.NodeName, err)
		return fmt.Errorf("failed to update node labels: %w", err)
	}

	s.logger.Info("Successfully updated labels for node %s in cluster %s", req.NodeName, clusterName)
	return nil
}

// UpdateNodeTaints 更新节点污点
func (s *Service) UpdateNodeTaints(clusterName string, req TaintUpdateRequest) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 获取当前节点
	node, err := client.CoreV1().Nodes().Get(ctx, req.NodeName, metav1.GetOptions{})
	if err != nil {
		s.logger.Error("Failed to get node %s: %v", req.NodeName, err)
		return fmt.Errorf("failed to get node: %w", err)
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

	// 更新节点
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		s.logger.Error("Failed to update node taints for %s: %v", req.NodeName, err)
		return fmt.Errorf("failed to update node taints: %w", err)
	}

	s.logger.Info("Successfully updated taints for node %s in cluster %s", req.NodeName, clusterName)
	return nil
}

// DrainNode 驱逐节点
func (s *Service) DrainNode(clusterName, nodeName string) error {
	client, err := s.getClient(clusterName)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// 首先标记节点为不可调度
	node, err := client.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	node.Spec.Unschedulable = true
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	// 获取节点上的pods
	pods, err := client.CoreV1().Pods("").List(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("spec.nodeName=%s", nodeName),
	})
	if err != nil {
		return fmt.Errorf("failed to list pods on node: %w", err)
	}

	// 驱逐pods
	for _, pod := range pods.Items {
		// 跳过DaemonSet管理的pods和已经在删除的pods
		if pod.DeletionTimestamp != nil {
			continue
		}

		// 检查是否是DaemonSet
		isDaemonSet := false
		for _, owner := range pod.OwnerReferences {
			if owner.Kind == "DaemonSet" {
				isDaemonSet = true
				break
			}
		}
		if isDaemonSet {
			continue
		}

		// 删除pod
		err = client.CoreV1().Pods(pod.Namespace).Delete(ctx, pod.Name, metav1.DeleteOptions{
			GracePeriodSeconds: func() *int64 { i := int64(30); return &i }(),
		})
		if err != nil {
			s.logger.Warning("Failed to delete pod %s/%s: %v", pod.Namespace, pod.Name, err)
		}
	}

	s.logger.Info("Successfully drained node %s in cluster %s", nodeName, clusterName)
	return nil
}

// UncordonNode 取消节点驱逐
func (s *Service) UncordonNode(clusterName, nodeName string) error {
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

	node.Spec.Unschedulable = false
	_, err = client.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to uncordon node: %w", err)
	}

	s.logger.Info("Successfully uncordoned node %s in cluster %s", nodeName, clusterName)
	return nil
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

	return NodeInfo{
		Name:             node.Name,
		Status:           status,
		Schedulable:      !node.Spec.Unschedulable,
		Roles:            roles,
		Age:              s.getAge(node.CreationTimestamp.Time),
		Version:          node.Status.NodeInfo.KubeletVersion,
		InternalIP:       internalIP,
		ExternalIP:       externalIP,
		OS:               node.Status.NodeInfo.OperatingSystem,
		OSImage:          node.Status.NodeInfo.OSImage,
		KernelVersion:    node.Status.NodeInfo.KernelVersion,
		ContainerRuntime: node.Status.NodeInfo.ContainerRuntimeVersion,
		Capacity: ResourceInfo{
			CPU:    node.Status.Capacity.Cpu().String(),
			Memory: node.Status.Capacity.Memory().String(),
			Pods:   node.Status.Capacity.Pods().String(),
			GPU:    s.extractGPUResources(node.Status.Capacity),
		},
		Allocatable: ResourceInfo{
			CPU:    node.Status.Allocatable.Cpu().String(),
			Memory: node.Status.Allocatable.Memory().String(),
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
	if node.Spec.Unschedulable {
		return "SchedulingDisabled"
	}

	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			if condition.Status == corev1.ConditionTrue {
				return "Ready"
			}
			return "NotReady"
		}
	}
	return "Unknown"
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
		s.logger.Warning("Cannot list nodes with this kubeconfig (limited permissions): %v", err)
		// 不返回错误，只是记录警告
		// 对于只有特定权限的service account，这是正常的

		// 3. 尝试列出命名空间（更基础的权限）
		_, err = clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
		if err != nil {
			s.logger.Warning("Cannot list namespaces with this kubeconfig: %v", err)
			// 仍然不返回错误，继续验证其他权限
		}
	}

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
	}

	for _, key := range gpuResourceKeys {
		if quantity, exists := resources[corev1.ResourceName(key)]; exists {
			gpuResources[key] = quantity.String()
		}
	}

	return gpuResources
}
