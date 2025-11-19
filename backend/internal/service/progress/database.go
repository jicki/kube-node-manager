package progress

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// DatabaseProgressService 基于数据库的进度服务，支持多副本
type DatabaseProgressService struct {
	db                *gorm.DB
	logger            *logger.Logger
	wsService         *Service // 原有的WebSocket服务
	stopPolling       chan struct{}
	pollingWg         sync.WaitGroup
	lastProcessedTime time.Time
	pollInterval      time.Duration
}

// NewDatabaseProgressService 创建数据库进度服务
func NewDatabaseProgressService(db *gorm.DB, logger *logger.Logger, wsService *Service) *DatabaseProgressService {
	dps := &DatabaseProgressService{
		db:           db,
		logger:       logger,
		wsService:    wsService,
		stopPolling:  make(chan struct{}),
		pollInterval: 1 * time.Second, // 每秒检查一次新消息
	}

	// 启动消息轮询
	go dps.startMessagePolling()

	return dps
}

// CreateTask 创建任务
func (dps *DatabaseProgressService) CreateTask(taskID, action string, total int, userID uint) error {
	task := &model.ProgressTask{
		TaskID:    taskID,
		UserID:    userID,
		Action:    action,
		Status:    model.TaskStatusRunning,
		Current:   0,
		Total:     total,
		Progress:  0,
		Message:   "任务已创建，准备开始处理",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := dps.db.Create(task).Error; err != nil {
		dps.logger.Errorf("Failed to create task %s: %v", taskID, err)
		return err
	}

	dps.logger.Infof("Created database task %s with %d total items for user %d", taskID, total, userID)
	return nil
}

// UpdateProgress 更新任务进度
func (dps *DatabaseProgressService) UpdateProgress(taskID string, current int, currentNode string, userID uint) error {
	var task model.ProgressTask
	if err := dps.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		dps.logger.Warningf("Task %s not found for progress update: %v", taskID, err)
		return err
	}

	// 更新进度
	task.UpdateProgress(current, currentNode)
	task.Message = fmt.Sprintf("正在处理节点 %s (%d/%d)", currentNode, current, task.Total)

	if err := dps.db.Save(&task).Error; err != nil {
		dps.logger.Errorf("Failed to update task %s progress: %v", taskID, err)
		return err
	}

	// 创建进度消息
	return dps.createProgressMessage(&task, "progress")
}

// CompleteTask 完成任务
func (dps *DatabaseProgressService) CompleteTask(taskID string, userID uint) error {
	var task model.ProgressTask
	if err := dps.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		dps.logger.Warningf("Task %s not found for completion: %v", taskID, err)
		return err
	}

	// 标记完成
	task.MarkCompleted()
	task.Message = fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total)

	if err := dps.db.Save(&task).Error; err != nil {
		dps.logger.Errorf("Failed to complete task %s: %v", taskID, err)
		return err
	}

	// 创建完成消息
	if err := dps.createProgressMessage(&task, "complete"); err != nil {
		return err
	}

	dps.logger.Infof("Task %s completed successfully in database", taskID)

	// 立即尝试推送完成消息，不等待轮询
	go func() {
		for i := 0; i < 5; i++ { // 重试5次
			time.Sleep(200 * time.Millisecond * time.Duration(i)) // 递增延迟

			// 检查是否有WebSocket连接
			dps.wsService.connMutex.RLock()
			hasConnection := false
			if _, exists := dps.wsService.connections[userID]; exists {
				hasConnection = true
			}
			dps.wsService.connMutex.RUnlock()

			if hasConnection {
				// 立即处理未发送的消息
				dps.processUnsentMessages()
				dps.logger.Infof("Force pushed completion message for task %s, user %d", taskID, userID)
				break
			} else {
				dps.logger.Infof("Waiting for WebSocket connection to push completion message (attempt %d/5)", i+1)
			}
		}
	}()

	// 设置延时清理
	go func() {
		time.Sleep(60 * time.Second)
		dps.cleanupTask(taskID)
	}()

	return nil
}

// ErrorTask 标记任务失败
func (dps *DatabaseProgressService) ErrorTask(taskID string, err error, userID uint) error {
	var task model.ProgressTask
	if dbErr := dps.db.Where("task_id = ?", taskID).First(&task).Error; dbErr != nil {
		dps.logger.Warningf("Task %s not found for error marking: %v", taskID, dbErr)
		return dbErr
	}

	// 标记失败
	task.MarkFailed(err.Error())
	task.Message = "批量操作失败"

	if dbErr := dps.db.Save(&task).Error; dbErr != nil {
		dps.logger.Errorf("Failed to mark task %s as failed: %v", taskID, dbErr)
		return dbErr
	}

	// 创建错误消息
	return dps.createProgressMessage(&task, "error")
}

// createProgressMessage 创建进度消息
func (dps *DatabaseProgressService) createProgressMessage(task *model.ProgressTask, msgType string) error {
	msg := &model.ProgressMessage{
		UserID:   task.UserID,
		TaskID:   task.TaskID,
		Type:     msgType,
		Action:   task.Action,
		Current:  task.Current,
		Total:    task.Total,
		Progress: task.Progress,
		Message:  task.Message,
		ErrorMsg: task.ErrorMsg,
	}

	if err := dps.db.Create(msg).Error; err != nil {
		dps.logger.Errorf("Failed to create progress message for task %s: %v", task.TaskID, err)
		return err
	}

	dps.logger.Infof("Created %s message for task %s, user %d", msgType, task.TaskID, task.UserID)
	return nil
}

// startMessagePolling 启动消息轮询，处理未发送的消息
func (dps *DatabaseProgressService) startMessagePolling() {
	dps.pollingWg.Add(1)
	defer dps.pollingWg.Done()

	// 对于完成消息使用更短的轮询间隔
	ticker := time.NewTicker(500 * time.Millisecond) // 缩短到500ms
	defer ticker.Stop()

	for {
		select {
		case <-dps.stopPolling:
			dps.logger.Infof("Message polling stopped")
			return
		case <-ticker.C:
			dps.processUnsentMessages()
		}
	}
}

// processUnsentMessages 处理未发送的消息
func (dps *DatabaseProgressService) processUnsentMessages() {
	var messages []model.ProgressMessage

	// 优先处理完成和错误消息，然后处理普通进度消息
	query := dps.db.Where("processed = ? AND created_at > ?", false, dps.lastProcessedTime).
		Order("CASE WHEN type IN ('complete', 'error') THEN 0 ELSE 1 END, created_at ASC").
		Limit(100) // 限制批次大小

	if err := query.Find(&messages).Error; err != nil {
		dps.logger.Errorf("Failed to query unsent messages: %v", err)
		return
	}

	if len(messages) == 0 {
		return
	}

	dps.logger.Infof("Processing %d unsent messages", len(messages))

	completedCount := 0
	for _, msg := range messages {
		// 转换为WebSocket消息格式
		wsMessage := ProgressMessage{
			TaskID:      msg.TaskID,
			Type:        msg.Type,
			Action:      msg.Action,
			Current:     msg.Current,
			Total:       msg.Total,
			Progress:    msg.Progress,
			CurrentNode: "", // 这个字段在数据库中没有存储
			Message:     msg.Message,
			Error:       msg.ErrorMsg,
			Timestamp:   msg.CreatedAt,
		}

		// 检查WebSocket连接状态
		dps.wsService.connMutex.RLock()
		hasConnection := false
		if _, exists := dps.wsService.connections[msg.UserID]; exists {
			hasConnection = true
		}
		dps.wsService.connMutex.RUnlock()

		if hasConnection {
			// 有连接时直接发送
			dps.wsService.sendToUser(msg.UserID, wsMessage)
			dps.logger.Infof("Sent %s message to connected user %d for task %s", msg.Type, msg.UserID, msg.TaskID)
		} else if msg.Type == "complete" || msg.Type == "error" {
			// 没有连接但是重要消息，等待一下再重试
			dps.logger.Warningf("No connection for important message type %s, will retry", msg.Type)
			time.Sleep(100 * time.Millisecond)
			// 再次检查连接
			dps.wsService.sendToUser(msg.UserID, wsMessage)
		} else {
			// 普通进度消息，没有连接就跳过
			dps.logger.Infof("Skipping progress message for disconnected user %d", msg.UserID)
		}

		// 标记为已处理
		if err := dps.db.Model(&msg).Update("processed", true).Error; err != nil {
			dps.logger.Errorf("Failed to mark message %d as processed: %v", msg.ID, err)
		} else {
			dps.logger.Infof("Marked message %d (%s) as processed for user %d", msg.ID, msg.Type, msg.UserID)
			if msg.Type == "complete" {
				completedCount++
			}
		}
	}

	// 特别记录完成消息的处理
	if completedCount > 0 {
		dps.logger.Infof("Processed %d completion messages", completedCount)
	}

	// 更新最后处理时间
	if len(messages) > 0 {
		dps.lastProcessedTime = messages[len(messages)-1].CreatedAt
	}
}

// cleanupTask 清理旧任务
func (dps *DatabaseProgressService) cleanupTask(taskID string) {
	// 软删除任务
	if err := dps.db.Where("task_id = ?", taskID).Delete(&model.ProgressTask{}).Error; err != nil {
		dps.logger.Errorf("Failed to cleanup task %s: %v", taskID, err)
	} else {
		dps.logger.Infof("Cleaned up task %s", taskID)
	}

	// 清理相关的已处理消息
	if err := dps.db.Where("task_id = ? AND processed = ?", taskID, true).Delete(&model.ProgressMessage{}).Error; err != nil {
		dps.logger.Errorf("Failed to cleanup messages for task %s: %v", taskID, err)
	}
}

// GetUserTasks 获取用户的任务列表
func (dps *DatabaseProgressService) GetUserTasks(userID uint, status model.TaskStatus) ([]model.ProgressTask, error) {
	var tasks []model.ProgressTask
	query := dps.db.Where("user_id = ?", userID)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&tasks).Error; err != nil {
		return nil, err
	}

	return tasks, nil
}

// Stop 停止服务
func (dps *DatabaseProgressService) Stop() {
	close(dps.stopPolling)
	dps.pollingWg.Wait()
	dps.logger.Infof("Database progress service stopped")
}

// ProcessBatchWithProgress 带数据库持久化的批量处理
func (dps *DatabaseProgressService) ProcessBatchWithProgress(
	ctx context.Context,
	taskID string,
	action string,
	nodeNames []string,
	userID uint,
	maxConcurrency int,
	processor BatchProcessor,
) error {
	total := len(nodeNames)

	// 创建数据库任务
	if err := dps.CreateTask(taskID, action, total, userID); err != nil {
		return err
	}

	// 使用信号量控制并发
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string
	processed := 0

	for i, nodeName := range nodeNames {
		wg.Add(1)
		go func(index int, node string) {
			defer func() {
				if r := recover(); r != nil {
					dps.logger.Errorf("Panic while processing node %s: %v", node, r)
					mu.Lock()
					errors = append(errors, fmt.Sprintf("%s: panic: %v", node, r))
					mu.Unlock()
				}
				wg.Done()
				dps.logger.Infof("Goroutine for node %s completed", node)
			}()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			dps.logger.Infof("Starting to process node %s (index %d)", node, index)

			// 先获取当前索引用于日志
			mu.Lock()
			processed++
			currentIndex := processed
			mu.Unlock()

			dps.logger.Infof("Node %s assigned index %d/%d", node, currentIndex, total)

			// 发送开始处理的进度消息
			dps.UpdateProgress(taskID, currentIndex, node, userID)

			// 处理节点
			dps.logger.Infof("Calling ProcessNode for %s", node)
			if err := processor.ProcessNode(ctx, node, index); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", node, err))
				mu.Unlock()
				dps.logger.Errorf("Failed to process node %s: %v", node, err)
			} else {
				dps.logger.Infof("Successfully processed node %s (%d/%d)", node, currentIndex, total)
			}

			// 处理完成后再次更新进度，确保前端收到最新状态
			dps.logger.Infof("Sending final progress update for node %s", node)
			dps.UpdateProgress(taskID, currentIndex, node, userID)
		}(i, nodeName)
	}

	// 等待所有任务完成
	wg.Wait()

	dps.logger.Infof("All nodes processed for task %s, processed=%d, errors=%d", taskID, processed, len(errors))

	// 确保最后一次进度更新显示 100%
	if len(errors) == 0 {
		dps.logger.Infof("Sending final 100%% progress update for task %s", taskID)
		dps.UpdateProgress(taskID, total, "完成", userID)
	}

	// 处理结果
	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("部分节点处理失败: %s", errors[0])
		if len(errors) > 1 {
			errorMsg = fmt.Sprintf("部分节点处理失败: %s 等 %d 个错误", errors[0], len(errors))
		}
		dps.logger.Errorf("Task %s failed with %d errors, calling ErrorTask", taskID, len(errors))
		err := fmt.Errorf("%s", errorMsg)
		dps.ErrorTask(taskID, err, userID)
		return err
	}

	dps.logger.Infof("Task %s completed successfully, calling CompleteTask for user %d", taskID, userID)
	if err := dps.CompleteTask(taskID, userID); err != nil {
		dps.logger.Errorf("Failed to mark task %s as completed: %v", taskID, err)
		return err
	}
	return nil
}
