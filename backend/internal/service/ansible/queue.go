package ansible

import (
	"container/heap"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"sync"
	"time"

	"gorm.io/gorm"
)

// QueueService 任务队列服务
type QueueService struct {
	db     *gorm.DB
	logger *logger.Logger
	mu     sync.RWMutex
}

// NewQueueService 创建 QueueService 实例
func NewQueueService(db *gorm.DB, logger *logger.Logger) *QueueService {
	return &QueueService{
		db:     db,
		logger: logger,
	}
}

// TaskQueueItem 队列任务项
type TaskQueueItem struct {
	TaskID     uint
	Priority   string
	QueuedAt   time.Time
	UserID     uint
	index      int // heap 索引
}

// TaskPriorityQueue 任务优先级队列（实现 heap.Interface）
type TaskPriorityQueue []*TaskQueueItem

func (pq TaskPriorityQueue) Len() int { return len(pq) }

func (pq TaskPriorityQueue) Less(i, j int) bool {
	// 优先级权重：high=3, medium=2, low=1
	priorityWeight := map[string]int{
		string(model.TaskPriorityHigh):   3,
		string(model.TaskPriorityMedium): 2,
		string(model.TaskPriorityLow):    1,
	}
	
	wi := priorityWeight[pq[i].Priority]
	wj := priorityWeight[pq[j].Priority]
	
	// 1. 优先级更高的任务优先
	if wi != wj {
		return wi > wj
	}
	
	// 2. 同优先级按入队时间（FIFO）
	return pq[i].QueuedAt.Before(pq[j].QueuedAt)
}

func (pq TaskPriorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *TaskPriorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*TaskQueueItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *TaskPriorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // 避免内存泄漏
	item.index = -1 // 标记为已移除
	*pq = old[0 : n-1]
	return item
}

// GetNextTask 获取下一个要执行的任务（优先级队列 + 公平调度）
func (s *QueueService) GetNextTask(maxConcurrentPerUser int) (*model.AnsibleTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 1. 获取所有待执行的任务
	var pendingTasks []model.AnsibleTask
	if err := s.db.Where("status = ?", model.AnsibleTaskStatusPending).
		Order("priority DESC, queued_at ASC").
		Find(&pendingTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}

	if len(pendingTasks) == 0 {
		return nil, nil // 没有待执行任务
	}

	// 2. 统计每个用户当前正在运行的任务数
	userRunningCount := make(map[uint]int)
	var runningTasks []model.AnsibleTask
	if err := s.db.Where("status = ?", model.AnsibleTaskStatusRunning).
		Select("user_id").
		Find(&runningTasks).Error; err != nil {
		s.logger.Errorf("Failed to query running tasks: %v", err)
	} else {
		for _, task := range runningTasks {
			userRunningCount[task.UserID]++
		}
	}

	// 3. 构建优先级队列
	pq := make(TaskPriorityQueue, 0, len(pendingTasks))
	for _, task := range pendingTasks {
		// 公平调度：如果用户已达到并发上限，跳过该用户的任务
		if maxConcurrentPerUser > 0 && userRunningCount[task.UserID] >= maxConcurrentPerUser {
			s.logger.Debugf("User %d has reached max concurrent tasks (%d), skipping task %d", 
				task.UserID, maxConcurrentPerUser, task.ID)
			continue
		}

		queuedAt := task.CreatedAt
		if task.QueuedAt != nil {
			queuedAt = *task.QueuedAt
		}

		pq = append(pq, &TaskQueueItem{
			TaskID:   task.ID,
			Priority: task.Priority,
			QueuedAt: queuedAt,
			UserID:   task.UserID,
		})
	}

	if len(pq) == 0 {
		return nil, nil // 所有任务都被公平调度策略阻止
	}

	// 4. 初始化堆并取出最高优先级任务
	heap.Init(&pq)
	nextItem := heap.Pop(&pq).(*TaskQueueItem)

	// 5. 获取完整的任务信息
	var nextTask model.AnsibleTask
	if err := s.db.First(&nextTask, nextItem.TaskID).Error; err != nil {
		return nil, fmt.Errorf("failed to get task %d: %w", nextItem.TaskID, err)
	}

	s.logger.Infof("Next task selected: ID=%d, Priority=%s, QueuedAt=%s, UserID=%d",
		nextTask.ID, nextTask.Priority, nextItem.QueuedAt.Format(time.RFC3339), nextTask.UserID)

	return &nextTask, nil
}

// GetQueueStats 获取队列统计信息
func (s *QueueService) GetQueueStats() (*QueueStats, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := &QueueStats{
		ByPriority: make(map[string]int),
		ByUser:     make(map[uint]int),
	}

	// 统计待执行任务
	var pendingTasks []model.AnsibleTask
	if err := s.db.Where("status = ?", model.AnsibleTaskStatusPending).
		Find(&pendingTasks).Error; err != nil {
		return nil, fmt.Errorf("failed to query pending tasks: %w", err)
	}

	stats.TotalPending = len(pendingTasks)

	for _, task := range pendingTasks {
		// 按优先级统计
		stats.ByPriority[task.Priority]++

		// 按用户统计
		stats.ByUser[task.UserID]++

		// 计算平均等待时间
		queuedAt := task.CreatedAt
		if task.QueuedAt != nil {
			queuedAt = *task.QueuedAt
		}
		waitDuration := time.Since(queuedAt)
		stats.TotalWaitDuration += waitDuration

		// 最长等待时间
		if waitDuration > stats.MaxWaitDuration {
			stats.MaxWaitDuration = waitDuration
			stats.MaxWaitTaskID = task.ID
		}
	}

	// 计算平均等待时间
	if stats.TotalPending > 0 {
		stats.AvgWaitDuration = stats.TotalWaitDuration / time.Duration(stats.TotalPending)
	}

	// 统计正在运行的任务
	if err := s.db.Model(&model.AnsibleTask{}).
		Where("status = ?", model.AnsibleTaskStatusRunning).
		Count(&stats.TotalRunning).Error; err != nil {
		return nil, fmt.Errorf("failed to count running tasks: %w", err)
	}

	return stats, nil
}

// QueueStats 队列统计信息
type QueueStats struct {
	TotalPending      int                `json:"total_pending"`       // 待执行任务总数
	TotalRunning      int64              `json:"total_running"`       // 正在运行任务总数
	ByPriority        map[string]int     `json:"by_priority"`         // 按优先级统计
	ByUser            map[uint]int       `json:"by_user"`             // 按用户统计
	AvgWaitDuration   time.Duration      `json:"avg_wait_duration"`   // 平均等待时间
	MaxWaitDuration   time.Duration      `json:"max_wait_duration"`   // 最长等待时间
	MaxWaitTaskID     uint               `json:"max_wait_task_id"`    // 等待最久的任务ID
	TotalWaitDuration time.Duration      `json:"total_wait_duration"` // 总等待时间
}

// UpdateWaitDuration 更新任务的等待时长（在任务开始执行时调用）
func (s *QueueService) UpdateWaitDuration(taskID uint) error {
	var task model.AnsibleTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	if task.QueuedAt != nil {
		waitDuration := int(time.Since(*task.QueuedAt).Seconds())
		if err := s.db.Model(&task).Update("wait_duration", waitDuration).Error; err != nil {
			return fmt.Errorf("failed to update wait duration: %w", err)
		}
		s.logger.Infof("Task %d wait duration updated: %d seconds", taskID, waitDuration)
	}

	return nil
}

