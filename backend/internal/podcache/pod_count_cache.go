package podcache

import (
	"sync"
	"sync/atomic"

	corev1 "k8s.io/api/core/v1"
	"kube-node-manager/internal/informer"
	"kube-node-manager/pkg/logger"
)

// PodCountCache 轻量级 Pod 统计缓存
// 设计原则：
// 1. 只存储必要信息（UID -> nodeName），不存储完整 Pod 对象
// 2. 使用 atomic 操作保证并发安全
// 3. 内存占用：~100 bytes/pod（相比完整对象的 50KB）
type PodCountCache struct {
	// 每个节点的 Pod 计数: "cluster:node" -> *int32
	nodePodCounts sync.Map

	// Pod 索引: "cluster:podUID" -> nodeName
	// 用于处理 Pod 删除和迁移场景
	podToNode sync.Map

	logger *logger.Logger
	mu     sync.RWMutex
}

// NewPodCountCache 创建 Pod 统计缓存
func NewPodCountCache(logger *logger.Logger) *PodCountCache {
	return &PodCountCache{
		logger: logger,
	}
}

// OnPodEvent 处理 Pod 事件（实现 informer.PodEventHandler 接口）
func (pc *PodCountCache) OnPodEvent(event informer.PodEvent) {
	// 过滤空节点（Pod 尚未调度）
	if event.Pod.Spec.NodeName == "" && event.Type != informer.EventTypeDelete {
		return
	}

	switch event.Type {
	case informer.EventTypeAdd:
		pc.handlePodAdd(event)
	case informer.EventTypeDelete:
		pc.handlePodDelete(event)
	case informer.EventTypeUpdate:
		pc.handlePodUpdate(event)
	}
}

// handlePodAdd 处理 Pod 添加事件
func (pc *PodCountCache) handlePodAdd(event informer.PodEvent) {
	// 过滤终止状态的 Pod
	if isTerminated(event.Pod.Status.Phase) {
		return
	}

	cluster := event.ClusterName
	podUID := string(event.Pod.UID)
	nodeName := event.Pod.Spec.NodeName

	if nodeName == "" {
		return // Pod 尚未调度到节点
	}

	// 递增节点 Pod 计数
	pc.incrementPodCount(cluster, nodeName)

	// 记录 Pod 到节点的映射
	pc.podToNode.Store(makeKey(cluster, podUID), nodeName)

	pc.logger.Debugf("Pod added: cluster=%s, pod=%s/%s, node=%s",
		cluster, event.Pod.Namespace, event.Pod.Name, nodeName)
}

// handlePodDelete 处理 Pod 删除事件
func (pc *PodCountCache) handlePodDelete(event informer.PodEvent) {
	cluster := event.ClusterName
	podUID := string(event.Pod.UID)

	// 获取 Pod 所在节点
	key := makeKey(cluster, podUID)
	if nodeNameInterface, ok := pc.podToNode.LoadAndDelete(key); ok {
		nodeName := nodeNameInterface.(string)
		pc.decrementPodCount(cluster, nodeName)

		pc.logger.Debugf("Pod deleted: cluster=%s, pod=%s/%s, node=%s",
			cluster, event.Pod.Namespace, event.Pod.Name, nodeName)
	}
}

// handlePodUpdate 处理 Pod 更新事件
func (pc *PodCountCache) handlePodUpdate(event informer.PodEvent) {
	cluster := event.ClusterName
	podUID := string(event.Pod.UID)
	newNodeName := event.Pod.Spec.NodeName
	newPhase := event.Pod.Status.Phase

	key := makeKey(cluster, podUID)

	// 场景1：Pod 迁移到其他节点
	if oldNodeInterface, ok := pc.podToNode.Load(key); ok {
		oldNodeName := oldNodeInterface.(string)

		if oldNodeName != newNodeName && newNodeName != "" {
			// Pod 迁移：旧节点 -1，新节点 +1
			pc.decrementPodCount(cluster, oldNodeName)
			pc.incrementPodCount(cluster, newNodeName)
			pc.podToNode.Store(key, newNodeName)

			pc.logger.Infof("Pod migrated: cluster=%s, pod=%s/%s, %s -> %s",
				cluster, event.Pod.Namespace, event.Pod.Name, oldNodeName, newNodeName)
		}
	} else {
		// 场景2：Pod 从 Pending 变为 Running（首次调度）
		if newNodeName != "" && !isTerminated(newPhase) {
			pc.incrementPodCount(cluster, newNodeName)
			pc.podToNode.Store(key, newNodeName)

			pc.logger.Debugf("Pod scheduled: cluster=%s, pod=%s/%s, node=%s",
				cluster, event.Pod.Namespace, event.Pod.Name, newNodeName)
		}
	}

	// 场景3：Pod 变为终止状态
	if isTerminated(newPhase) {
		if nodeNameInterface, ok := pc.podToNode.LoadAndDelete(key); ok {
			nodeName := nodeNameInterface.(string)
			pc.decrementPodCount(cluster, nodeName)

			pc.logger.Debugf("Pod terminated: cluster=%s, pod=%s/%s, node=%s, phase=%s",
				cluster, event.Pod.Namespace, event.Pod.Name, nodeName, newPhase)
		}
	}
}

// GetNodePodCount 获取单个节点的 Pod 数量（O(1) 时间复杂度）
func (pc *PodCountCache) GetNodePodCount(cluster, nodeName string) int {
	key := makeKey(cluster, nodeName)
	if countInterface, ok := pc.nodePodCounts.Load(key); ok {
		count := countInterface.(*int32)
		return int(atomic.LoadInt32(count))
	}
	return 0
}

// GetAllNodePodCounts 获取集群所有节点的 Pod 数量
func (pc *PodCountCache) GetAllNodePodCounts(cluster string) map[string]int {
	result := make(map[string]int)

	prefix := cluster + ":"
	pc.nodePodCounts.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(prefix) && keyStr[:len(prefix)] == prefix {
			nodeName := keyStr[len(prefix):]
			count := value.(*int32)
			result[nodeName] = int(atomic.LoadInt32(count))
		}
		return true
	})

	return result
}

// GetCacheStats 获取缓存统计信息
func (pc *PodCountCache) GetCacheStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 统计 Pod 总数
	podCount := 0
	pc.podToNode.Range(func(_, _ interface{}) bool {
		podCount++
		return true
	})
	stats["total_pods"] = podCount

	// 统计节点数
	nodeCount := 0
	pc.nodePodCounts.Range(func(_, _ interface{}) bool {
		nodeCount++
		return true
	})
	stats["total_nodes"] = nodeCount

	// 估算内存占用（每个 Pod ~100 bytes）
	estimatedMemoryMB := float64(podCount) * 100 / 1024 / 1024
	stats["estimated_memory_mb"] = estimatedMemoryMB

	return stats
}

// InvalidateCluster 清除指定集群的所有缓存
func (pc *PodCountCache) InvalidateCluster(cluster string) {
	prefix := cluster + ":"

	// 清除 Pod 索引
	keysToDelete := make([]string, 0)
	pc.podToNode.Range(func(key, _ interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(prefix) && keyStr[:len(prefix)] == prefix {
			keysToDelete = append(keysToDelete, keyStr)
		}
		return true
	})

	for _, key := range keysToDelete {
		pc.podToNode.Delete(key)
	}

	// 清除节点计数
	keysToDelete = keysToDelete[:0]
	pc.nodePodCounts.Range(func(key, _ interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(prefix) && keyStr[:len(prefix)] == prefix {
			keysToDelete = append(keysToDelete, keyStr)
		}
		return true
	})

	for _, key := range keysToDelete {
		pc.nodePodCounts.Delete(key)
	}

	pc.logger.Infof("Invalidated pod count cache for cluster: %s", cluster)
}

// IsReady 检查缓存是否就绪（至少有一些数据）
func (pc *PodCountCache) IsReady(cluster string) bool {
	prefix := cluster + ":"
	ready := false

	pc.nodePodCounts.Range(func(key, _ interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(prefix) && keyStr[:len(prefix)] == prefix {
			ready = true
			return false // 找到一个即可
		}
		return true
	})

	return ready
}

// incrementPodCount 递增节点 Pod 计数（原子操作）
func (pc *PodCountCache) incrementPodCount(cluster, nodeName string) {
	key := makeKey(cluster, nodeName)

	countInterface, _ := pc.nodePodCounts.LoadOrStore(key, new(int32))
	count := countInterface.(*int32)
	atomic.AddInt32(count, 1)
}

// decrementPodCount 递减节点 Pod 计数（原子操作）
func (pc *PodCountCache) decrementPodCount(cluster, nodeName string) {
	key := makeKey(cluster, nodeName)

	if countInterface, ok := pc.nodePodCounts.Load(key); ok {
		count := countInterface.(*int32)
		newCount := atomic.AddInt32(count, -1)

		// 如果计数降为 0 或负数，删除键（节省内存）
		if newCount <= 0 {
			pc.nodePodCounts.Delete(key)
		}
	}
}

// 辅助函数：生成缓存键
func makeKey(cluster, identifier string) string {
	return cluster + ":" + identifier
}

// 辅助函数：判断 Pod 是否处于终止状态
func isTerminated(phase corev1.PodPhase) bool {
	return phase == corev1.PodSucceeded || phase == corev1.PodFailed
}

