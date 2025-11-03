package realtime

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/internal/informer"
	"kube-node-manager/internal/smartcache"
	"kube-node-manager/internal/websocket"
	"kube-node-manager/pkg/logger"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// Manager 实时同步管理器
// 统一管理 Informer、SmartCache 和 WebSocket Hub
type Manager struct {
	informerSvc   *informer.Service
	smartCache    *smartcache.SmartCache
	wsHub         *websocket.Hub
	logger        *logger.Logger
	mu            sync.RWMutex
	clusterClients map[string]*kubernetes.Clientset // cluster -> clientset
}

// NewManager 创建实时同步管理器
func NewManager(logger *logger.Logger) *Manager {
	m := &Manager{
		informerSvc:    informer.NewService(logger),
		smartCache:     smartcache.NewSmartCache(logger),
		wsHub:          websocket.NewHub(logger),
		logger:         logger,
		clusterClients: make(map[string]*kubernetes.Clientset),
	}

	// 注册节点事件处理器
	// SmartCache 监听 Informer 事件
	m.informerSvc.RegisterHandler(m.smartCache)

	// WebSocket Hub 监听 Informer 事件
	m.informerSvc.RegisterHandler(m.wsHub)

	logger.Info("Realtime Manager initialized")

	return m
}

// RegisterPodEventHandler 注册 Pod 事件处理器
// 用于将 PodCountCache 注册到 Informer
func (m *Manager) RegisterPodEventHandler(handler informer.PodEventHandler) {
	m.informerSvc.RegisterPodHandler(handler)
	m.logger.Infof("Registered Pod event handler: %T", handler)
}

// Start 启动管理器
func (m *Manager) Start() {
	// 启动 WebSocket Hub
	go m.wsHub.Run()

	m.logger.Info("Realtime Manager started")
}

// RegisterCluster 注册集群并启动 Informer
func (m *Manager) RegisterCluster(clusterName string, clientset *kubernetes.Clientset) error {
	m.mu.Lock()
	m.clusterClients[clusterName] = clientset
	m.mu.Unlock()

	// 启动 Informer 前，先从 K8s API 获取初始数据并填充 SmartCache
	// 这样可以确保在 Informer 完成同步前，用户也能看到数据
	m.logger.Infof("Fetching initial node list for cluster %s", clusterName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		m.logger.Warningf("Failed to fetch initial nodes for cluster %s: %v", clusterName, err)
		// 继续启动 Informer，即使初始加载失败
	} else {
		// 将节点添加到 SmartCache
		for i := range nodeList.Items {
			m.smartCache.SetNode(clusterName, &nodeList.Items[i])
		}
		m.logger.Infof("Initialized SmartCache with %d nodes for cluster %s", len(nodeList.Items), clusterName)
	}

	// 启动节点 Informer
	if err := m.informerSvc.StartInformer(clusterName, clientset); err != nil {
		return fmt.Errorf("failed to start node informer for cluster %s: %w", clusterName, err)
	}

	// 延迟启动 Pod Informer（避免启动时资源竞争）
	// 在后台延迟10秒后启动，给系统一些时间完成初始化
	go func() {
		// 等待10秒，让服务先完全启动
		time.Sleep(10 * time.Second)
		
		m.logger.Infof("Starting Pod Informer for cluster: %s (delayed start)", clusterName)
		
		if err := m.informerSvc.StartPodInformer(clusterName); err != nil {
			m.logger.Warningf("Failed to start Pod Informer for cluster %s: %v", clusterName, err)
			m.logger.Infof("Pod count will fall back to API query mode for cluster %s", clusterName)
		} else {
			m.logger.Infof("✓ Pod Informer ready for cluster: %s", clusterName)
		}
	}()

	m.logger.Infof("Cluster registered: %s (Pod Informer will start in 10s)", clusterName)
	return nil
}

// UnregisterCluster 注销集群并停止 Informer
func (m *Manager) UnregisterCluster(clusterName string) {
	m.mu.Lock()
	delete(m.clusterClients, clusterName)
	m.mu.Unlock()

	m.informerSvc.StopInformer(clusterName)
	m.smartCache.InvalidateCluster(clusterName)

	m.logger.Infof("Cluster unregistered: %s", clusterName)
}

// GetSmartCache 获取智能缓存
func (m *Manager) GetSmartCache() *smartcache.SmartCache {
	return m.smartCache
}

// GetWebSocketHub 获取 WebSocket Hub
func (m *Manager) GetWebSocketHub() *websocket.Hub {
	return m.wsHub
}

// GetInformerService 获取 Informer 服务
func (m *Manager) GetInformerService() *informer.Service {
	return m.informerSvc
}

// Shutdown 关闭管理器
func (m *Manager) Shutdown() {
	m.logger.Info("Shutting down Realtime Manager")

	// 停止所有 Informer
	m.informerSvc.StopAll()

	m.logger.Info("Realtime Manager shut down")
}

// GetStatus 获取管理器状态
func (m *Manager) GetStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})

	// Informer 状态
	status["informers"] = m.informerSvc.GetInformerStatus()

	// SmartCache 状态
	status["cache"] = m.smartCache.GetCacheStats()

	// WebSocket Hub 状态
	status["websocket"] = m.wsHub.GetStats()

	// 注册的集群数
	status["cluster_count"] = len(m.clusterClients)

	return status
}

// IsClusterRegistered 检查集群是否已注册
func (m *Manager) IsClusterRegistered(clusterName string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.clusterClients[clusterName]
	return exists
}

// GetRegisteredClusters 获取所有已注册的集群名称
func (m *Manager) GetRegisteredClusters() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clusters := make([]string, 0, len(m.clusterClients))
	for clusterName := range m.clusterClients {
		clusters = append(clusters, clusterName)
	}
	return clusters
}

