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
	
	// 使用通知器发送实时通知（仅每10次发送一次，避免过多通知）
	if !dps.usePolling && (task.Current%10 == 0 || task.Current == task.Total) {
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
			} else {
				dps.logger.Debugf("Task %s: Unmarshaled %d success nodes", taskID, len(notifySuccessNodes))
			}
		}
		if task.FailedNodes != "" {
			if err := json.Unmarshal([]byte(task.FailedNodes), &notifyFailedNodes); err != nil {
				dps.logger.Errorf("Failed to unmarshal failed nodes for task %s: %v (data: %s)", 
					taskID, err, task.FailedNodes[:min(100, len(task.FailedNodes))])
			} else {
				dps.logger.Debugf("Task %s: Unmarshaled %d failed nodes", taskID, len(notifyFailedNodes))
			}
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

	// 立即尝试推送完成消息，不等待轮询（降级方案）
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
				dps.logger.Debugf("Force pushed completion message for task %s", taskID)
				break
			} else {
				// 减少日志噪音，只记录最后一次尝试
				if i == 4 {
					dps.logger.Warningf("No WebSocket connection after 5 attempts for task %s", taskID)
				}
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
				// 直接通过 WebSocket 推送消息
				dps.wsService.sendToUser(msg.UserID, msg)
				// 只记录重要消息
				if msg.Type == "complete" || msg.Type == "error" {
					dps.logger.Infof("Forwarded %s notification for task %s to user %d", msg.Type, msg.TaskID, msg.UserID)
				}
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
	processed := 0

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
					dps.UpdateNodeLists(taskID, successNodes, failedNodes)
					mu.Unlock()
				}
				wg.Done()
			}()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 先获取当前索引用于日志
			mu.Lock()
			processed++
			currentIndex := processed
			mu.Unlock()

			// 发送开始处理的进度消息
			dps.UpdateProgress(taskID, currentIndex, node, userID)

			// 处理节点
			err := processor.ProcessNode(ctx, node, index)
			if err != nil {
				mu.Lock()
				failedNodes = append(failedNodes, model.NodeError{
					NodeName: node,
					Error:    err.Error(),
				})
				dps.UpdateNodeLists(taskID, successNodes, failedNodes)
				mu.Unlock()
				dps.logger.Errorf("Failed to process node %s: %v", node, err)
			} else {
				mu.Lock()
				successNodes = append(successNodes, node)
				dps.UpdateNodeLists(taskID, successNodes, failedNodes)
				mu.Unlock()
				// 减少日志频率：每10个节点或最后一个节点记录一次
				if currentIndex%10 == 0 || currentIndex == total {
					dps.logger.Infof("Progress: %d/%d nodes processed successfully", currentIndex, total)
				}
			}

			// 处理完成后再次更新进度，确保前端收到最新状态（不记录日志避免轰炸）
			dps.UpdateProgress(taskID, currentIndex, node, userID)
		}(i, nodeName)
	}

	// 等待所有任务完成
	wg.Wait()

	dps.logger.Infof("Task %s completed: processed=%d, success=%d, failed=%d", 
		taskID, processed, len(successNodes), len(failedNodes))

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

// VerifyNotifier 验证通知器是否正常工作
func (dps *DatabaseProgressService) VerifyNotifier() error {
	if dps.notifier == nil {
		return fmt.Errorf("notifier is nil")
	}
	
	notifierType := dps.notifier.Type()
	dps.logger.Infof("Verifying notifier type: %s", notifierType)
	
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
			return fmt.Errorf("failed to send test notification: %w", err)
		}
		
		dps.logger.Infof("✅ Notifier verification successful (type=%s)", notifierType)
	}
	
	return nil
}
