package informer

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"kube-node-manager/pkg/logger"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
)

// NodeEvent 节点事件类型
type NodeEvent struct {
	Type        EventType       // 事件类型：Add/Update/Delete
	ClusterName string          // 集群名称
	Node        *corev1.Node    // 节点对象
	OldNode     *corev1.Node    // 旧节点对象（仅 Update 事件）
	Timestamp   time.Time       // 事件时间
	Changes     []string        // 变化的字段列表
}

// EventType 事件类型
type EventType string

const (
	EventTypeAdd    EventType = "Add"
	EventTypeUpdate EventType = "Update"
	EventTypeDelete EventType = "Delete"
)

// NodeEventHandler 节点事件处理器接口
type NodeEventHandler interface {
	OnNodeEvent(event NodeEvent)
}

// PodEvent Pod 事件类型
type PodEvent struct {
	Type        EventType    // 事件类型：Add/Update/Delete
	ClusterName string       // 集群名称
	Pod         *corev1.Pod  // Pod 对象
	OldPod      *corev1.Pod  // 旧 Pod 对象（仅 Update 事件）
	Timestamp   time.Time    // 事件时间
}

// PodEventHandler Pod 事件处理器接口
type PodEventHandler interface {
	OnPodEvent(event PodEvent)
}

// Service Informer 服务
type Service struct {
	logger      *logger.Logger
	informers   map[string]informers.SharedInformerFactory // cluster -> informer
	stoppers    map[string]chan struct{}                   // cluster -> stop channel
	clients     map[string]*kubernetes.Clientset           // cluster -> clientset (用于自动清理操作)
	handlers    []NodeEventHandler                         // 节点事件处理器列表
	podHandlers []PodEventHandler                          // Pod 事件处理器列表
	mu          sync.RWMutex
}

// NewService 创建 Informer 服务
func NewService(logger *logger.Logger) *Service {
	return &Service{
		logger:      logger,
		informers:   make(map[string]informers.SharedInformerFactory),
		stoppers:    make(map[string]chan struct{}),
		clients:     make(map[string]*kubernetes.Clientset),
		handlers:    make([]NodeEventHandler, 0),
		podHandlers: make([]PodEventHandler, 0),
	}
}

// RegisterHandler 注册节点事件处理器
func (s *Service) RegisterHandler(handler NodeEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers = append(s.handlers, handler)
	s.logger.Infof("Registered node event handler: %T", handler)
}

// RegisterPodHandler 注册 Pod 事件处理器
func (s *Service) RegisterPodHandler(handler PodEventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.podHandlers = append(s.podHandlers, handler)
	s.logger.Infof("Registered pod event handler: %T", handler)
}

// StartInformer 为指定集群启动 Informer
func (s *Service) StartInformer(clusterName string, clientset *kubernetes.Clientset) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查是否已经存在
	if _, exists := s.informers[clusterName]; exists {
		s.logger.Warningf("Informer for cluster %s already exists", clusterName)
		return nil
	}

	// 创建 SharedInformerFactory
	// resyncPeriod: 每 30 分钟全量同步一次，防止事件丢失
	factory := informers.NewSharedInformerFactory(clientset, 30*time.Minute)

	// 获取 NodeInformer
	nodeInformer := factory.Core().V1().Nodes().Informer()

	// 注册事件处理器
	nodeInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			s.handleNodeAdd(clusterName, node)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldNode := oldObj.(*corev1.Node)
			newNode := newObj.(*corev1.Node)
			s.handleNodeUpdate(clusterName, oldNode, newNode)
		},
		DeleteFunc: func(obj interface{}) {
			node := obj.(*corev1.Node)
			s.handleNodeDelete(clusterName, node)
		},
	})

	// 创建停止通道
	stopCh := make(chan struct{})
	s.stoppers[clusterName] = stopCh

	// 启动 Informer
	go factory.Start(stopCh)

	// 等待缓存同步
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if !cache.WaitForCacheSync(ctx.Done(), nodeInformer.HasSynced) {
		close(stopCh)
		delete(s.informers, clusterName)
		delete(s.stoppers, clusterName)
		return fmt.Errorf("failed to sync cache for cluster %s", clusterName)
	}

	s.informers[clusterName] = factory
	s.clients[clusterName] = clientset
	s.logger.Infof("Successfully started Informer for cluster: %s", clusterName)

	return nil
}

// StopInformer 停止指定集群的 Informer
func (s *Service) StopInformer(clusterName string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if stopCh, exists := s.stoppers[clusterName]; exists {
		close(stopCh)
		delete(s.informers, clusterName)
		delete(s.stoppers, clusterName)
		delete(s.clients, clusterName)
		s.logger.Infof("Stopped Informer for cluster: %s", clusterName)
	}
}

// StopAll 停止所有 Informer
func (s *Service) StopAll() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for clusterName, stopCh := range s.stoppers {
		close(stopCh)
		s.logger.Infof("Stopped Informer for cluster: %s", clusterName)
	}

	s.informers = make(map[string]informers.SharedInformerFactory)
	s.stoppers = make(map[string]chan struct{})
	s.clients = make(map[string]*kubernetes.Clientset)
	s.logger.Info("Stopped all Informers")
}

// handleNodeAdd 处理节点添加事件
func (s *Service) handleNodeAdd(clusterName string, node *corev1.Node) {
	s.logger.Infof("Node added: cluster=%s, node=%s", clusterName, node.Name)

	event := NodeEvent{
		Type:        EventTypeAdd,
		ClusterName: clusterName,
		Node:        node,
		Timestamp:   time.Now(),
		Changes:     []string{"*"}, // 新增节点，所有字段都是新的
	}

	s.notifyHandlers(event)
}

// handleNodeUpdate 处理节点更新事件
func (s *Service) handleNodeUpdate(clusterName string, oldNode, newNode *corev1.Node) {
	// 检测关键字段变化
	changes := s.detectChanges(oldNode, newNode)

	// 如果没有关键变化，忽略此事件（例如只是 ResourceVersion 变化）
	if len(changes) == 0 {
		return
	}

	// 自动清理遗留的 annotations：当节点从 Unschedulable 变为 Schedulable 时
	// 检查是否存在我们的 annotations，如果存在则自动清理
	s.autoCleanOrphanedAnnotations(clusterName, oldNode, newNode)

	// 只对重要变化输出日志，减少日志噪音
	// 重要变化：status、schedulable、taints
	// 频繁但不重要的变化：annotations、labels（除非同时有其他重要变化）
	if s.shouldLogUpdate(changes) {
		s.logger.Infof("Node updated: cluster=%s, node=%s, changes=%v", clusterName, newNode.Name, changes)
	}

	event := NodeEvent{
		Type:        EventTypeUpdate,
		ClusterName: clusterName,
		Node:        newNode,
		OldNode:     oldNode,
		Timestamp:   time.Now(),
		Changes:     changes,
	}

	s.notifyHandlers(event)
}

// handleNodeDelete 处理节点删除事件
func (s *Service) handleNodeDelete(clusterName string, node *corev1.Node) {
	s.logger.Infof("Node deleted: cluster=%s, node=%s", clusterName, node.Name)

	event := NodeEvent{
		Type:        EventTypeDelete,
		ClusterName: clusterName,
		Node:        node,
		Timestamp:   time.Now(),
		Changes:     []string{"*"}, // 删除节点，标记所有字段
	}

	s.notifyHandlers(event)
}

// shouldLogUpdate 判断是否应该输出节点更新日志
// 只对重要变化输出日志，减少日志噪音
func (s *Service) shouldLogUpdate(changes []string) bool {
	// 重要变化：status、schedulable、taints、conditions
	importantChanges := []string{"status", "schedulable", "taints", "conditions"}
	
	for _, change := range changes {
		for _, important := range importantChanges {
			if change == important {
				return true
			}
		}
	}
	
	// 如果只有 annotations 或 labels 变化，不输出日志
	return false
}

// detectChanges 检测节点关键字段的变化
func (s *Service) detectChanges(oldNode, newNode *corev1.Node) []string {
	changes := make([]string, 0)

	// 检查 Labels 变化
	if !equalMaps(oldNode.Labels, newNode.Labels) {
		changes = append(changes, "labels")
	}

	// 检查 Taints 变化
	if !equalTaints(oldNode.Spec.Taints, newNode.Spec.Taints) {
		changes = append(changes, "taints")
	}

	// 检查 Schedulable 变化
	if oldNode.Spec.Unschedulable != newNode.Spec.Unschedulable {
		changes = append(changes, "schedulable")
	}

	// 检查 Annotations 变化（包含禁止调度原因）
	if !equalMaps(oldNode.Annotations, newNode.Annotations) {
		changes = append(changes, "annotations")
	}

	// 检查状态变化（Ready/NotReady）
	if getNodeStatus(oldNode) != getNodeStatus(newNode) {
		changes = append(changes, "status")
	}

	// 检查 Conditions 变化
	if !equalConditions(oldNode.Status.Conditions, newNode.Status.Conditions) {
		changes = append(changes, "conditions")
	}

	return changes
}

// notifyHandlers 通知所有注册的事件处理器
func (s *Service) notifyHandlers(event NodeEvent) {
	s.mu.RLock()
	handlers := make([]NodeEventHandler, len(s.handlers))
	copy(handlers, s.handlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		// 异步通知，避免阻塞
		go func(h NodeEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Errorf("Event handler panic: %v", r)
				}
			}()
			h.OnNodeEvent(event)
		}(handler)
	}
}

// 辅助函数：比较两个 map
func equalMaps(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if b[k] != v {
			return false
		}
	}
	return true
}

// 辅助函数：比较 Taints
func equalTaints(a, b []corev1.Taint) bool {
	if len(a) != len(b) {
		return false
	}

	// 创建 map 用于比较（忽略顺序）
	aMap := make(map[string]corev1.Taint)
	for _, taint := range a {
		key := fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect)
		aMap[key] = taint
	}

	for _, taint := range b {
		key := fmt.Sprintf("%s=%s:%s", taint.Key, taint.Value, taint.Effect)
		if _, exists := aMap[key]; !exists {
			return false
		}
	}

	return true
}

// 辅助函数：获取节点状态
func getNodeStatus(node *corev1.Node) string {
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

// 辅助函数：比较 Conditions
func equalConditions(a, b []corev1.NodeCondition) bool {
	if len(a) != len(b) {
		return false
	}

	aMap := make(map[corev1.NodeConditionType]corev1.ConditionStatus)
	for _, cond := range a {
		aMap[cond.Type] = cond.Status
	}

	for _, cond := range b {
		if status, exists := aMap[cond.Type]; !exists || status != cond.Status {
			return false
		}
	}

	return true
}

// GetInformerStatus 获取 Informer 状态
func (s *Service) GetInformerStatus() map[string]bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := make(map[string]bool)
	for clusterName := range s.informers {
		status[clusterName] = true
	}
	return status
}

// StartPodInformer 为指定集群启动 Pod Informer
// 注意：必须在 StartInformer 之后调用，因为需要复用 SharedInformerFactory
func (s *Service) StartPodInformer(clusterName string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查节点 Informer 是否已启动
	factory, exists := s.informers[clusterName]
	if !exists {
		return fmt.Errorf("node informer not started for cluster %s, please call StartInformer first", clusterName)
	}

	// 获取 PodInformer（复用现有 factory）
	podInformer := factory.Core().V1().Pods().Informer()

	// 注册事件处理器
	podInformer.AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			s.handlePodAdd(clusterName, pod)
		},
		UpdateFunc: func(oldObj, newObj interface{}) {
			oldPod := oldObj.(*corev1.Pod)
			newPod := newObj.(*corev1.Pod)
			s.handlePodUpdate(clusterName, oldPod, newPod)
		},
		DeleteFunc: func(obj interface{}) {
			pod := obj.(*corev1.Pod)
			s.handlePodDelete(clusterName, pod)
		},
	})

	// 等待缓存同步（增加超时时间以适应大规模集群）
	// 对于大规模集群（如10k+ pods），初始同步可能需要较长时间
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	s.logger.Infof("Waiting for Pod Informer cache sync for cluster: %s (timeout: 120s)", clusterName)

	if !cache.WaitForCacheSync(ctx.Done(), podInformer.HasSynced) {
		return fmt.Errorf("failed to sync pod cache for cluster %s within 120s (cluster may have too many pods)", clusterName)
	}

	s.logger.Infof("Successfully started Pod Informer for cluster: %s", clusterName)

	return nil
}

// handlePodAdd 处理 Pod 添加事件
func (s *Service) handlePodAdd(clusterName string, pod *corev1.Pod) {
	event := PodEvent{
		Type:        EventTypeAdd,
		ClusterName: clusterName,
		Pod:         pod,
		Timestamp:   time.Now(),
	}

	s.notifyPodHandlers(event)
}

// handlePodUpdate 处理 Pod 更新事件
func (s *Service) handlePodUpdate(clusterName string, oldPod, newPod *corev1.Pod) {
	// 只关注关键字段变化：nodeName、phase
	if oldPod.Spec.NodeName == newPod.Spec.NodeName &&
		oldPod.Status.Phase == newPod.Status.Phase {
		return // 无关键变化，忽略
	}

	event := PodEvent{
		Type:        EventTypeUpdate,
		ClusterName: clusterName,
		Pod:         newPod,
		OldPod:      oldPod,
		Timestamp:   time.Now(),
	}

	s.notifyPodHandlers(event)
}

// handlePodDelete 处理 Pod 删除事件
func (s *Service) handlePodDelete(clusterName string, pod *corev1.Pod) {
	event := PodEvent{
		Type:        EventTypeDelete,
		ClusterName: clusterName,
		Pod:         pod,
		Timestamp:   time.Now(),
	}

	s.notifyPodHandlers(event)
}

// notifyPodHandlers 通知所有注册的 Pod 事件处理器
func (s *Service) notifyPodHandlers(event PodEvent) {
	s.mu.RLock()
	handlers := make([]PodEventHandler, len(s.podHandlers))
	copy(handlers, s.podHandlers)
	s.mu.RUnlock()

	for _, handler := range handlers {
		// 异步通知，避免阻塞
		go func(h PodEventHandler) {
			defer func() {
				if r := recover(); r != nil {
					s.logger.Errorf("Pod event handler panic: %v", r)
				}
			}()
			h.OnPodEvent(event)
		}(handler)
	}
}

// autoCleanOrphanedAnnotations 自动清理遗留的 annotations
// 当检测到节点从 Unschedulable 变为 Schedulable 时，如果存在我们的 annotations，则自动清理
// 这解决了使用原生 kubectl uncordon 命令时不会清理自定义 annotations 的问题
func (s *Service) autoCleanOrphanedAnnotations(clusterName string, oldNode, newNode *corev1.Node) {
	// 只在节点从 Unschedulable 变为 Schedulable 时检查
	if !oldNode.Spec.Unschedulable || newNode.Spec.Unschedulable {
		return // 不是 uncordon 操作，跳过
	}

	// 检查是否存在我们的 annotations
	hasAnnotations := false
	if newNode.Annotations != nil {
		_, hasReasonAnnotation := newNode.Annotations["deeproute.cn/kube-node-mgr"]
		_, hasTimestampAnnotation := newNode.Annotations["deeproute.cn/kube-node-mgr-timestamp"]
		hasAnnotations = hasReasonAnnotation || hasTimestampAnnotation
	}

	// 如果没有我们的 annotations，无需清理
	if !hasAnnotations {
		return
	}

	// 获取集群的 clientset
	s.mu.RLock()
	clientset, exists := s.clients[clusterName]
	s.mu.RUnlock()

	if !exists {
		s.logger.Warningf("Cannot clean annotations for node %s: clientset not found for cluster %s", newNode.Name, clusterName)
		return
	}

	// 异步清理，避免阻塞 informer 事件处理
	go s.cleanNodeAnnotations(clusterName, newNode.Name, clientset)
}

// cleanNodeAnnotations 清理节点上遗留的 kube-node-mgr annotations
func (s *Service) cleanNodeAnnotations(clusterName, nodeName string, clientset *kubernetes.Clientset) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 使用重试机制处理资源冲突
	maxRetries := 3
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// 指数退避
			backoff := time.Duration(100*(1<<uint(attempt-1))) * time.Millisecond
			time.Sleep(backoff)
		}

		// 重新获取节点（获取最新的 ResourceVersion）
		node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
		if err != nil {
			s.logger.Errorf("Failed to get node %s for annotation cleanup: %v", nodeName, err)
			return
		}

		// 再次检查节点是否可调度（可能在重试期间状态又变了）
		if node.Spec.Unschedulable {
			s.logger.Infof("Node %s is cordoned again, skipping annotation cleanup", nodeName)
			return
		}

		// 检查 annotations 是否还存在
		if node.Annotations == nil {
			return // 已经被清理了
		}

		_, hasReason := node.Annotations["deeproute.cn/kube-node-mgr"]
		_, hasTimestamp := node.Annotations["deeproute.cn/kube-node-mgr-timestamp"]

		if !hasReason && !hasTimestamp {
			return // 已经被清理了
		}

		// 删除遗留的 annotations
		needsUpdate := false
		if hasReason {
			delete(node.Annotations, "deeproute.cn/kube-node-mgr")
			needsUpdate = true
		}
		if hasTimestamp {
			delete(node.Annotations, "deeproute.cn/kube-node-mgr-timestamp")
			needsUpdate = true
		}

		if !needsUpdate {
			return
		}

		// 更新节点
		_, err = clientset.CoreV1().Nodes().Update(ctx, node, metav1.UpdateOptions{})
		if err != nil {
			// 检查是否是资源冲突错误
			if strings.Contains(err.Error(), "the object has been modified") ||
				strings.Contains(err.Error(), "Operation cannot be fulfilled") {
				s.logger.Warningf("Resource conflict when cleaning annotations for node %s (attempt %d/%d): %v", nodeName, attempt+1, maxRetries+1, err)
				continue // 重试
			}
			// 其他错误
			s.logger.Errorf("Failed to clean annotations for node %s: %v", nodeName, err)
			return
		}

		// 成功
		s.logger.Infof("✓ Auto-cleaned orphaned annotations for node %s in cluster %s (uncordoned via kubectl)", nodeName, clusterName)
		return
	}

	s.logger.Errorf("Failed to clean annotations for node %s after %d attempts", nodeName, maxRetries+1)
}

