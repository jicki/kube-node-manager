package progress

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/internal/config"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// DatabaseProgressService 基于数据库的进度服务，支持多副本
type DatabaseProgressService struct {
	db                *gorm.DB
	logger            *logger.Logger
	wsService         *Service // 原有的WebSocket服务
	notifier          ProgressNotifier // 通知器（PostgreSQL/Redis/Polling）
	stopPolling       chan struct{}
	pollingWg         sync.WaitGroup
	lastProcessedTime time.Time
	pollInterval      time.Duration
	usePolling        bool // 是否使用轮询模式
}

// NewDatabaseProgressService 创建数据库进度服务
func NewDatabaseProgressService(db *gorm.DB, dbConfig *config.DatabaseConfig, logger *logger.Logger, wsService *Service, notifyType string, pollInterval int, redisAddr, redisPassword string, redisDB int) *DatabaseProgressService {
	dps := &DatabaseProgressService{
		db:           db,
		logger:       logger,
		wsService:    wsService,
		stopPolling:  make(chan struct{}),
		pollInterval: time.Duration(pollInterval) * time.Millisecond,
		usePolling:   false,
	}

	// 根据配置创建相应的通知器
	var notifier ProgressNotifier
	var err error
	
	switch notifyType {
	case "postgres":
		notifier, err = NewPostgresNotifier(db, dbConfig, logger)
		if err != nil {
			logger.Errorf("Failed to create PostgreSQL notifier, falling back to polling: %v", err)
			notifier = NewPollingNotifier(dps.pollInterval, logger)
			dps.usePolling = true
		} else {
			logger.Info("Using PostgreSQL LISTEN/NOTIFY for real-time progress updates")
		}
		
	case "redis":
		notifier, err = NewRedisNotifier(redisAddr, redisPassword, redisDB, logger)
		if err != nil {
			logger.Errorf("Failed to create Redis notifier, falling back to polling: %v", err)
			notifier = NewPollingNotifier(dps.pollInterval, logger)
			dps.usePolling = true
		} else {
			logger.Info("Using Redis Pub/Sub for real-time progress updates")
		}
		
	case "polling":
		notifier = NewPollingNotifier(dps.pollInterval, logger)
		dps.usePolling = true
		logger.Info("Using polling mode for progress updates")
		
	default:
		logger.Warningf("Unknown notify type '%s', falling back to polling", notifyType)
		notifier = NewPollingNotifier(dps.pollInterval, logger)
		dps.usePolling = true
	}
	
	dps.notifier = notifier

	// 如果使用实时通知（非轮询），启动订阅处理
	if !dps.usePolling {
		go dps.startNotificationSubscription()
	}
	
	// 无论使用哪种模式，都启动轮询作为降级方案
	// 但如果使用实时通知，轮询间隔会更长（作为备份）
	if dps.usePolling {
		go dps.startMessagePolling()
	} else {
		// 实时通知模式下，使用更长的轮询间隔作为降级（每 10 秒）
		go dps.startFallbackPolling()
	}

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

// UpdateNodeLists 更新成功和失败节点列表
func (dps *DatabaseProgressService) UpdateNodeLists(taskID string, successNodes []string, failedNodes []model.NodeError) error {
	var task model.ProgressTask
	if err := dps.db.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		dps.logger.Errorf("Failed to find task %s for node list update: %v", taskID, err)
		return err
	}

	// 转换为JSON
	if len(successNodes) > 0 {
		successJSON, err := json.Marshal(successNodes)
		if err == nil {
			task.SuccessNodes = string(successJSON)
		} else {
			dps.logger.Errorf("Failed to marshal success nodes for task %s: %v", taskID, err)
		}
	}

	if len(failedNodes) > 0 {
		failedJSON, err := json.Marshal(failedNodes)
		if err == nil {
			task.FailedNodes = string(failedJSON)
		} else {
			dps.logger.Errorf("Failed to marshal failed nodes for task %s: %v", taskID, err)
		}
	}

	if err := dps.db.Save(&task).Error; err != nil {
		dps.logger.Errorf("Failed to save task %s with updated node lists: %v", taskID, err)
		return err
	}

	// 不记录每次更新的日志，避免日志轰炸
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

	// 创建进度消息并通知
	if err := dps.createProgressMessage(&task, "progress"); err != nil {
		return err
	}
	
	// 使用通知器发送实时通知
	// 根据任务规模决定通知频率：
	// - 小规模任务（<10个节点）：每个节点都通知
	// - 中等规模（10-50个节点）：每3个节点通知一次
	// - 大规模任务（>50个节点）：每10个节点通知一次
	shouldNotify := false
	if task.Total < 10 {
		// 小规模：每个节点都通知
		shouldNotify = true
	} else if task.Total <= 50 {
		// 中等规模：每3个节点或最后一个
		shouldNotify = (task.Current%3 == 0 || task.Current == task.Total)
	} else {
		// 大规模：每10个节点或最后一个
		shouldNotify = (task.Current%10 == 0 || task.Current == task.Total)
	}
	
	if !dps.usePolling && shouldNotify {
		// 解析成功和失败节点列表
		var successNodes []string
		var failedNodes []model.NodeError
		if task.SuccessNodes != "" {
			json.Unmarshal([]byte(task.SuccessNodes), &successNodes)
		}
		if task.FailedNodes != "" {
			json.Unmarshal([]byte(task.FailedNodes), &failedNodes)
		}
		
		progressMsg := ProgressMessage{
			TaskID:       task.TaskID,
			UserID:       userID,
			Type:         "progress",
			Action:       task.Action,
			Current:      task.Current,
			Total:        task.Total,
			Progress:     task.Progress,
			CurrentNode:  task.CurrentNode,
			Message:      task.Message,
			Error:        task.ErrorMsg,
			Timestamp:    time.Now(),
			SuccessNodes: successNodes,
			FailedNodes:  failedNodes,
		}
		if err := dps.notifier.Notify(context.Background(), progressMsg); err != nil {
			// 通知失败不返回错误，会通过轮询降级（不记录警告避免日志轰炸）
		}
	}
	
	return nil
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

	// 创建完成消息并通知
	if err := dps.createProgressMessage(&task, "complete"); err != nil {
		return err
	}
	
	// 解析成功和失败节点列表（用于日志）
	var successNodes []string
	var failedNodes []model.NodeError
	if task.SuccessNodes != "" {
		json.Unmarshal([]byte(task.SuccessNodes), &successNodes)
	}
	if task.FailedNodes != "" {
		json.Unmarshal([]byte(task.FailedNodes), &failedNodes)
	}
	
	// 使用通知器发送实时通知
	if !dps.usePolling {
		// 重新解析以确保数据完整
		var notifySuccessNodes []string
		var notifyFailedNodes []model.NodeError
		if task.SuccessNodes != "" {
		if err := json.Unmarshal([]byte(task.SuccessNodes), &notifySuccessNodes); err != nil {
			dps.logger.Errorf("Failed to unmarshal success nodes for task %s: %v (data: %s)", 
				taskID, err, task.SuccessNodes[:min(100, len(task.SuccessNodes))])
		}
		// 移除成功解析的 DEBUG 日志，避免日志轰炸
		}
		if task.FailedNodes != "" {
		if err := json.Unmarshal([]byte(task.FailedNodes), &notifyFailedNodes); err != nil {
			dps.logger.Errorf("Failed to unmarshal failed nodes for task %s: %v (data: %s)", 
				taskID, err, task.FailedNodes[:min(100, len(task.FailedNodes))])
		}
		// 移除成功解析的 DEBUG 日志，避免日志轰炸
		}
		
		progressMsg := ProgressMessage{
			TaskID:       task.TaskID,
			UserID:       userID,
			Type:         "complete",
			Action:       task.Action,
			Current:      task.Current,
			Total:        task.Total,
			Progress:     100.0,
			CurrentNode:  task.CurrentNode,
			Message:      task.Message,
			Error:        task.ErrorMsg,
			Timestamp:    time.Now(),
			SuccessNodes: notifySuccessNodes,
			FailedNodes:  notifyFailedNodes,
		}
		if err := dps.notifier.Notify(context.Background(), progressMsg); err != nil {
			dps.logger.Errorf("Failed to send completion notification for task %s: %v", taskID, err)
		} else {
			dps.logger.Infof("Completion notification sent for task %s (success=%d, failed=%d)", 
				taskID, len(notifySuccessNodes), len(notifyFailedNodes))
		}
	}

	dps.logger.Infof("Task %s completed successfully in database (success=%d, failed=%d)", 
		taskID, len(successNodes), len(failedNodes))

	// 注释掉强制推送逻辑，避免重复发送完成消息
	// PostgreSQL LISTEN/NOTIFY 或轮询机制已经会处理消息推送
	// 如果 LISTEN/NOTIFY 工作正常，消息会立即推送
	// 如果失败，轮询机制（10秒间隔）会作为降级方案
	
	// 旧的强制推送逻辑会导致重复消息：
	// - LISTEN/NOTIFY 发送一次
	// - 强制推送 processUnsentMessages() 再发送一次
	// 结果是前端收到多个重复的完成消息

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

	// 创建错误消息并通知
	if err := dps.createProgressMessage(&task, "error"); err != nil {
		return err
	}
	
	// 使用通知器发送实时通知
	if !dps.usePolling {
		// 解析成功和失败节点列表
		var successNodes []string
		var failedNodes []model.NodeError
		if task.SuccessNodes != "" {
			json.Unmarshal([]byte(task.SuccessNodes), &successNodes)
		}
		if task.FailedNodes != "" {
			json.Unmarshal([]byte(task.FailedNodes), &failedNodes)
		}
		
		progressMsg := ProgressMessage{
			TaskID:       task.TaskID,
			UserID:       userID,
			Type:         "error",
			Action:       task.Action,
			Current:      task.Current,
			Total:        task.Total,
			Progress:     task.Progress,
			CurrentNode:  task.CurrentNode,
			Message:      task.Message,
			Error:        task.ErrorMsg,
			Timestamp:    time.Now(),
			SuccessNodes: successNodes,
			FailedNodes:  failedNodes,
		}
		if err := dps.notifier.Notify(context.Background(), progressMsg); err != nil {
			dps.logger.Warningf("Failed to send error notification: %v", err)
		}
	}
	
	return nil
}

// createProgressMessage 创建进度消息
func (dps *DatabaseProgressService) createProgressMessage(task *model.ProgressTask, msgType string) error {
	msg := &model.ProgressMessage{
		UserID:       task.UserID,
		TaskID:       task.TaskID,
		Type:         msgType,
		Action:       task.Action,
		Current:      task.Current,
		Total:        task.Total,
		Progress:     task.Progress,
		CurrentNode:  task.CurrentNode,
		SuccessNodes: task.SuccessNodes,
		FailedNodes:  task.FailedNodes,
		Message:      task.Message,
		ErrorMsg:     task.ErrorMsg,
	}

	if err := dps.db.Create(msg).Error; err != nil {
		dps.logger.Errorf("Failed to create progress message for task %s: %v", task.TaskID, err)
		return err
	}

	// 只记录完成和错误消息，避免进度消息日志噪音
	if msgType == "complete" || msgType == "error" {
		dps.logger.Infof("Created %s message for task %s, user %d", msgType, task.TaskID, task.UserID)
	}
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

	// 只记录有重要消息时的日志
	importantCount := 0
	for _, msg := range messages {
		if msg.Type == "complete" || msg.Type == "error" {
			importantCount++
		}
	}
	if importantCount > 0 {
		dps.logger.Infof("Processing %d unsent messages (%d important)", len(messages), importantCount)
	}

	completedCount := 0
	for _, msg := range messages {
		// 解析成功和失败节点列表
		var successNodes []string
		var failedNodes []model.NodeError
		
		if msg.SuccessNodes != "" {
			json.Unmarshal([]byte(msg.SuccessNodes), &successNodes)
		}
		
		if msg.FailedNodes != "" {
			json.Unmarshal([]byte(msg.FailedNodes), &failedNodes)
		}
		
		// 转换为WebSocket消息格式
		wsMessage := ProgressMessage{
			TaskID:       msg.TaskID,
			Type:         msg.Type,
			Action:       msg.Action,
			Current:      msg.Current,
			Total:        msg.Total,
			Progress:     msg.Progress,
			CurrentNode:  msg.CurrentNode,
			SuccessNodes: successNodes,
			FailedNodes:  failedNodes,
			Message:      msg.Message,
			Error:        msg.ErrorMsg,
			Timestamp:    msg.CreatedAt,
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
			// 只记录完成消息的发送日志
			if msg.Type == "complete" {
				dps.logger.Infof("Sent completion message for task %s to user %d", msg.TaskID, msg.UserID)
			}
		} else if msg.Type == "complete" || msg.Type == "error" {
			// 没有连接但是重要消息，等待一下再重试
			time.Sleep(100 * time.Millisecond)
			// 再次检查连接
			dps.wsService.sendToUser(msg.UserID, wsMessage)
		}

		// 标记为已处理
		if err := dps.db.Model(&msg).Update("processed", true).Error; err != nil {
			dps.logger.Errorf("Failed to mark message %d as processed: %v", msg.ID, err)
		} else {
			// 统计完成消息
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
	
	// 关闭通知器
	if dps.notifier != nil {
		if err := dps.notifier.Close(); err != nil {
			dps.logger.Errorf("Failed to close notifier: %v", err)
		}
	}
	
	dps.logger.Infof("Database progress service stopped")
}

// startNotificationSubscription 启动实时通知订阅处理
func (dps *DatabaseProgressService) startNotificationSubscription() {
	dps.pollingWg.Add(1)
	defer dps.pollingWg.Done()
	
	ctx := context.Background()
	messageChan, err := dps.notifier.Subscribe(ctx)
	if err != nil {
		dps.logger.Errorf("Failed to subscribe to notifications: %v, falling back to polling", err)
		dps.usePolling = true
		go dps.startMessagePolling()
		return
	}
	
	dps.logger.Infof("Started %s notification subscription", dps.notifier.Type())
	
	for {
		select {
		case <-dps.stopPolling:
			dps.logger.Info("Notification subscription stopped")
			return
			
		case msg, ok := <-messageChan:
			if !ok {
				dps.logger.Warning("Notification channel closed, restarting subscription")
				time.Sleep(5 * time.Second)
				
				// 尝试重新订阅
				messageChan, err = dps.notifier.Subscribe(ctx)
				if err != nil {
					dps.logger.Errorf("Failed to resubscribe: %v, falling back to polling", err)
					dps.usePolling = true
					go dps.startMessagePolling()
					return
				}
				continue
			}
			
			// 检查用户是否有活跃连接
			dps.wsService.connMutex.RLock()
			hasConnection := false
			if _, exists := dps.wsService.connections[msg.UserID]; exists {
				hasConnection = true
			}
			dps.wsService.connMutex.RUnlock()
			
		if hasConnection {
			// 记录转发的通知（使用合适的日志级别）
			if msg.Type == "complete" || msg.Type == "error" {
				dps.logger.Infof("Forwarding %s notification for task %s to user %d", msg.Type, msg.TaskID, msg.UserID)
			} else if msg.Type == "progress" {
				progress := int(msg.Progress)
				// 只在整十百分比时记录
				if progress%10 == 0 || msg.Current == 1 || msg.Current == msg.Total {
					dps.logger.Infof("Forwarding progress for task %s to user %d: %d/%d (%.0f%%)", 
						msg.TaskID, msg.UserID, msg.Current, msg.Total, msg.Progress)
				}
			}
			
			// 直接通过 WebSocket 推送消息
			dps.wsService.sendToUser(msg.UserID, msg)
		}
		}
	}
}

// startFallbackPolling 启动降级轮询（仅在使用实时通知时作为备份）
func (dps *DatabaseProgressService) startFallbackPolling() {
	dps.pollingWg.Add(1)
	defer dps.pollingWg.Done()
	
	// 使用更长的轮询间隔（10 秒）作为降级
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	
	dps.logger.Info("Started fallback polling (10s interval)")
	
	for {
		select {
		case <-dps.stopPolling:
			dps.logger.Info("Fallback polling stopped")
			return
		case <-ticker.C:
			// 只处理重要消息（complete, error）
			dps.processFallbackMessages()
		}
	}
}

// processFallbackMessages 处理降级消息（仅处理重要消息）
func (dps *DatabaseProgressService) processFallbackMessages() {
	var messages []model.ProgressMessage
	
	// 只查询完成和错误消息
	cutoff := time.Now().Add(-30 * time.Second) // 只处理最近 30 秒的消息
	query := dps.db.Where("processed = ? AND type IN (?) AND created_at > ?", 
		false, 
		[]string{"complete", "error"}, 
		cutoff,
	).Order("created_at ASC").Limit(50)
	
	if err := query.Find(&messages).Error; err != nil {
		dps.logger.Errorf("Failed to query fallback messages: %v", err)
		return
	}
	
	if len(messages) == 0 {
		return
	}
	
	dps.logger.Infof("Processing %d fallback messages", len(messages))
	
	for _, msg := range messages {
		// 解析节点列表
		var successNodes []string
		var failedNodes []model.NodeError
		
		if msg.SuccessNodes != "" {
			json.Unmarshal([]byte(msg.SuccessNodes), &successNodes)
		}
		if msg.FailedNodes != "" {
			json.Unmarshal([]byte(msg.FailedNodes), &failedNodes)
		}
		
		wsMessage := ProgressMessage{
			TaskID:       msg.TaskID,
			Type:         msg.Type,
			Action:       msg.Action,
			Current:      msg.Current,
			Total:        msg.Total,
			Progress:     msg.Progress,
			CurrentNode:  msg.CurrentNode,
			SuccessNodes: successNodes,
			FailedNodes:  failedNodes,
			Message:      msg.Message,
			Error:        msg.ErrorMsg,
			Timestamp:    msg.CreatedAt,
		}
		
		// 发送并标记
		dps.wsService.sendToUser(msg.UserID, wsMessage)
		dps.db.Model(&msg).Update("processed", true)
		dps.logger.Infof("Sent fallback %s message for task %s", msg.Type, msg.TaskID)
	}
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
	var failedNodes []model.NodeError
	var successNodes []string
	completed := 0 // 已完成的节点数（成功或失败）

	for i, nodeName := range nodeNames {
		wg.Add(1)
		go func(index int, node string) {
			defer func() {
				if r := recover(); r != nil {
					dps.logger.Errorf("Panic while processing node %s: %v", node, r)
					mu.Lock()
					failedNodes = append(failedNodes, model.NodeError{
						NodeName: node,
						Error:    fmt.Sprintf("panic: %v", r),
					})
					completed++ // 即使 panic 也算完成
					currentCompleted := completed
					dps.UpdateNodeLists(taskID, successNodes, failedNodes)
					mu.Unlock()
					// 更新进度
					dps.UpdateProgress(taskID, currentCompleted, node, userID)
				}
				wg.Done()
			}()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 处理节点
			err := processor.ProcessNode(ctx, node, index)
			
			// 处理完成后更新计数和进度
			mu.Lock()
			if err != nil {
				failedNodes = append(failedNodes, model.NodeError{
					NodeName: node,
					Error:    err.Error(),
				})
				dps.logger.Errorf("Failed to process node %s: %v", node, err)
			} else {
				successNodes = append(successNodes, node)
			}
			completed++ // 在处理完成后才递增
			currentCompleted := completed
			dps.UpdateNodeLists(taskID, successNodes, failedNodes)
			mu.Unlock()

			// 发送进度更新（在节点处理完成后）
			dps.UpdateProgress(taskID, currentCompleted, node, userID)
			
			// 每10个节点或最后一个节点记录进度日志
			if currentCompleted%10 == 0 || currentCompleted == total {
				dps.logger.Infof("Progress: %d/%d nodes processed (success=%d, failed=%d)", 
					currentCompleted, total, len(successNodes), len(failedNodes))
			}
		}(i, nodeName)
	}

	// 等待所有任务完成
	wg.Wait()

	// 获取最终统计
	mu.Lock()
	finalSuccess := len(successNodes)
	finalFailed := len(failedNodes)
	finalCompleted := completed
	mu.Unlock()

	dps.logger.Infof("Task %s completed: processed=%d, success=%d, failed=%d", 
		taskID, finalCompleted, finalSuccess, finalFailed)

	// 确保最后一次进度更新显示 100%
	if len(failedNodes) == 0 {
		dps.UpdateProgress(taskID, total, "完成", userID)
	}

	// 处理结果
	if len(failedNodes) > 0 {
		errorMsg := fmt.Sprintf("部分节点处理失败: %d个成功, %d个失败", len(successNodes), len(failedNodes))
		dps.logger.Errorf("Task %s completed with %d failures", taskID, len(failedNodes))
		err := fmt.Errorf("%s", errorMsg)
		dps.ErrorTask(taskID, err, userID)
		return err
	}

	if err := dps.CompleteTask(taskID, userID); err != nil {
		dps.logger.Errorf("Failed to mark task %s as completed: %v", taskID, err)
		return err
	}
	dps.logger.Infof("✅ Task %s completed successfully", taskID)
	return nil
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NotifierInfo 通知器信息
type NotifierInfo struct {
	Type      string // postgres, redis, polling
	IsHealthy bool   // 是否健康
	IsFallback bool  // 是否是降级模式
}

// VerifyNotifier 验证通知器是否正常工作
func (dps *DatabaseProgressService) VerifyNotifier() (*NotifierInfo, error) {
	if dps.notifier == nil {
		return nil, fmt.Errorf("notifier is nil")
	}
	
	notifierType := dps.notifier.Type()
	dps.logger.Infof("Verifying notifier type: %s", notifierType)
	
	info := &NotifierInfo{
		Type:      notifierType,
		IsHealthy: true,
		IsFallback: dps.usePolling && notifierType == "polling",
	}
	
	// 对于PostgreSQL和Redis通知器，尝试发送测试消息
	if notifierType == "postgres" || notifierType == "redis" {
		testMsg := ProgressMessage{
			TaskID:    "test_" + fmt.Sprintf("%d", time.Now().Unix()),
			UserID:    0,
			Type:      "test",
			Action:    "verification",
			Current:   0,
			Total:     1,
			Progress:  0,
			Message:   "Notifier verification test",
			Timestamp: time.Now(),
		}
		
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		if err := dps.notifier.Notify(ctx, testMsg); err != nil {
			dps.logger.Errorf("Notifier verification failed: %v", err)
			info.IsHealthy = false
			return info, fmt.Errorf("failed to send test notification: %w", err)
		}
		
		dps.logger.Infof("✅ Notifier verification successful (type=%s)", notifierType)
	} else if notifierType == "polling" {
		// Polling 模式总是"健康"的，但如果是降级则需要警告
		if info.IsFallback {
			dps.logger.Warningf("⚠️  Using polling mode as fallback - real-time updates may be delayed")
		} else {
			dps.logger.Infof("✅ Polling mode verified (configured mode)")
		}
	}
	
	return info, nil
}
