package smartcache

import (
	"sync"
	"time"

	"kube-node-manager/internal/informer"
	"kube-node-manager/pkg/logger"

	corev1 "k8s.io/api/core/v1"
)

// NodeCacheEntry 节点缓存条目
type NodeCacheEntry struct {
	Node      *corev1.Node // 完整的节点对象
	UpdatedAt time.Time    // 更新时间
	mu        sync.RWMutex
}

// SmartCache 智能缓存
// 区分静态属性和动态属性：
// - 静态属性（CPU/内存容量、OS等）：缓存1小时
// - 动态属性（Labels/Taints/Schedulable）：由 Informer 实时更新，无 TTL
type SmartCache struct {
	// 节点缓存: "cluster:node" -> NodeCacheEntry
	nodes sync.Map

	// 集群节点列表: cluster -> []nodeName
	clusterNodes sync.Map

	logger *logger.Logger

	// 静态属性缓存 TTL
	staticTTL time.Duration
}

// NewSmartCache 创建智能缓存
func NewSmartCache(logger *logger.Logger) *SmartCache {
	return &SmartCache{
		logger:    logger,
		staticTTL: 1 * time.Hour, // 静态属性缓存1小时
	}
}

// OnNodeEvent 实现 NodeEventHandler 接口，接收 Informer 事件
func (sc *SmartCache) OnNodeEvent(event informer.NodeEvent) {
	switch event.Type {
	case informer.EventTypeAdd:
		sc.handleNodeAdd(event)
	case informer.EventTypeUpdate:
		sc.handleNodeUpdate(event)
	case informer.EventTypeDelete:
		sc.handleNodeDelete(event)
	}
}

// handleNodeAdd 处理节点添加事件
func (sc *SmartCache) handleNodeAdd(event informer.NodeEvent) {
	key := makeKey(event.ClusterName, event.Node.Name)

	entry := &NodeCacheEntry{
		Node:      event.Node.DeepCopy(),
		UpdatedAt: time.Now(),
	}

	sc.nodes.Store(key, entry)

	// 更新集群节点列表
	sc.addNodeToCluster(event.ClusterName, event.Node.Name)

	sc.logger.Debugf("SmartCache: Added node %s to cluster %s", event.Node.Name, event.ClusterName)
}

// handleNodeUpdate 处理节点更新事件
func (sc *SmartCache) handleNodeUpdate(event informer.NodeEvent) {
	key := makeKey(event.ClusterName, event.Node.Name)

	if cached, ok := sc.nodes.Load(key); ok {
		entry := cached.(*NodeCacheEntry)
		entry.mu.Lock()
		entry.Node = event.Node.DeepCopy()
		entry.UpdatedAt = time.Now()
		entry.mu.Unlock()
		// 日志已在 Informer 中输出，此处不再重复记录
	} else {
		// 如果缓存中不存在，则添加
		sc.handleNodeAdd(event)
	}
}

// handleNodeDelete 处理节点删除事件
func (sc *SmartCache) handleNodeDelete(event informer.NodeEvent) {
	key := makeKey(event.ClusterName, event.Node.Name)
	sc.nodes.Delete(key)

	// 从集群节点列表中移除
	sc.removeNodeFromCluster(event.ClusterName, event.Node.Name)

	sc.logger.Debugf("SmartCache: Deleted node %s from cluster %s", event.Node.Name, event.ClusterName)
}

// GetNode 获取单个节点
func (sc *SmartCache) GetNode(clusterName, nodeName string) (*corev1.Node, bool) {
	key := makeKey(clusterName, nodeName)

	if cached, ok := sc.nodes.Load(key); ok {
		entry := cached.(*NodeCacheEntry)
		entry.mu.RLock()
		node := entry.Node.DeepCopy()
		entry.mu.RUnlock()
		return node, true
	}

	return nil, false
}

// GetNodes 获取集群的所有节点
func (sc *SmartCache) GetNodes(clusterName string) ([]*corev1.Node, bool) {
	// 获取节点名称列表
	nodeNames := sc.getClusterNodeNames(clusterName)
	if len(nodeNames) == 0 {
		return nil, false
	}

	nodes := make([]*corev1.Node, 0, len(nodeNames))
	for _, nodeName := range nodeNames {
		if node, ok := sc.GetNode(clusterName, nodeName); ok {
			nodes = append(nodes, node)
		}
	}

	return nodes, len(nodes) > 0
}

// SetNode 设置节点（用于初始化或手动更新）
func (sc *SmartCache) SetNode(clusterName string, node *corev1.Node) {
	key := makeKey(clusterName, node.Name)

	entry := &NodeCacheEntry{
		Node:      node.DeepCopy(),
		UpdatedAt: time.Now(),
	}

	sc.nodes.Store(key, entry)
	sc.addNodeToCluster(clusterName, node.Name)
}

// SetNodes 批量设置节点（用于初始化）
func (sc *SmartCache) SetNodes(clusterName string, nodes []*corev1.Node) {
	for _, node := range nodes {
		sc.SetNode(clusterName, node)
	}
	sc.logger.Infof("SmartCache: Initialized %d nodes for cluster %s", len(nodes), clusterName)
}

// InvalidateCluster 清除指定集群的所有缓存
func (sc *SmartCache) InvalidateCluster(clusterName string) {
	nodeNames := sc.getClusterNodeNames(clusterName)

	for _, nodeName := range nodeNames {
		key := makeKey(clusterName, nodeName)
		sc.nodes.Delete(key)
	}

	sc.clusterNodes.Delete(clusterName)
	sc.logger.Infof("SmartCache: Invalidated all nodes for cluster %s", clusterName)
}

// InvalidateNode 清除指定节点的缓存
func (sc *SmartCache) InvalidateNode(clusterName, nodeName string) {
	key := makeKey(clusterName, nodeName)
	sc.nodes.Delete(key)
	sc.removeNodeFromCluster(clusterName, nodeName)
}

// GetCacheStats 获取缓存统计信息
func (sc *SmartCache) GetCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 统计节点数量
	nodeCount := 0
	sc.nodes.Range(func(_, _ interface{}) bool {
		nodeCount++
		return true
	})
	stats["node_count"] = nodeCount

	// 统计集群数量
	clusterCount := 0
	sc.clusterNodes.Range(func(_, _ interface{}) bool {
		clusterCount++
		return true
	})
	stats["cluster_count"] = clusterCount

	stats["static_ttl"] = sc.staticTTL.String()

	return stats
}

// 辅助方法：添加节点到集群列表
func (sc *SmartCache) addNodeToCluster(clusterName, nodeName string) {
	nodesInterface, _ := sc.clusterNodes.LoadOrStore(clusterName, &sync.Map{})
	nodesMap := nodesInterface.(*sync.Map)
	nodesMap.Store(nodeName, true)
}

// 辅助方法：从集群列表中移除节点
func (sc *SmartCache) removeNodeFromCluster(clusterName, nodeName string) {
	if nodesInterface, ok := sc.clusterNodes.Load(clusterName); ok {
		nodesMap := nodesInterface.(*sync.Map)
		nodesMap.Delete(nodeName)
	}
}

// 辅助方法：获取集群的所有节点名称
func (sc *SmartCache) getClusterNodeNames(clusterName string) []string {
	if nodesInterface, ok := sc.clusterNodes.Load(clusterName); ok {
		nodesMap := nodesInterface.(*sync.Map)
		names := make([]string, 0)
		nodesMap.Range(func(key, _ interface{}) bool {
			names = append(names, key.(string))
			return true
		})
		return names
	}
	return nil
}

// 辅助函数：生成缓存键
func makeKey(clusterName, nodeName string) string {
	return clusterName + ":" + nodeName
}

