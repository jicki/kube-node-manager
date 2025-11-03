package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	ansibleUtil "kube-node-manager/pkg/ansible"
	"kube-node-manager/pkg/crypto"
	"kube-node-manager/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// Service Ansible 服务
type Service struct {
	db              *gorm.DB
	logger          *logger.Logger
	templateSvc     *TemplateService
	inventorySvc    *InventoryService
	sshKeySvc       *SSHKeyService
	scheduleSvc     *ScheduleService
	favoriteSvc     *FavoriteService
	preflightSvc    *PreflightService
	estimationSvc   *EstimationService
	queueSvc        *QueueService
	tagSvc          *TagService
	visualizationSvc *VisualizationService
	executor        *TaskExecutor
}

// NewService 创建 Ansible 服务实例
// encryptionKey 用于加密 SSH 密钥和密码，应该从环境变量或配置中获取
func NewService(db *gorm.DB, logger *logger.Logger, k8sSvc *k8s.Service, wsHub interface{}, encryptionKey string) *Service {
	// 如果未提供加密密钥，使用默认值（生产环境应该从配置读取）
	if encryptionKey == "" {
		encryptionKey = "default-encryption-key-change-in-production"
		logger.Warning("Using default encryption key for SSH credentials. Please set a secure encryption key in production!")
	}

	encryptor := crypto.NewEncryptor(encryptionKey)
	sshKeySvc := NewSSHKeyService(db, logger, encryptor)
	inventorySvc := NewInventoryService(db, logger, k8sSvc)
	templateSvc := NewTemplateService(db, logger)
	favoriteSvc := NewFavoriteService(db, logger)
	preflightSvc := NewPreflightService(db, logger, inventorySvc, sshKeySvc)
	estimationSvc := NewEstimationService(db, logger)
	queueSvc := NewQueueService(db, logger)
	tagSvc := NewTagService(db, logger)
	visualizationSvc := NewVisualizationService(db, logger)
	executor := NewTaskExecutor(db, logger, inventorySvc, sshKeySvc, wsHub)

	service := &Service{
		db:              db,
		logger:          logger,
		templateSvc:     templateSvc,
		inventorySvc:    inventorySvc,
		sshKeySvc:       sshKeySvc,
		favoriteSvc:     favoriteSvc,
		preflightSvc:    preflightSvc,
		estimationSvc:   estimationSvc,
		queueSvc:        queueSvc,
		tagSvc:          tagSvc,
		visualizationSvc: visualizationSvc,
		executor:        executor,
	}

	// 创建定时任务调度服务（需要依赖 service）
	scheduleSvc := NewScheduleService(db, logger, service)
	service.scheduleSvc = scheduleSvc

	return service
}

// GetTemplateService 获取模板服务
func (s *Service) GetTemplateService() *TemplateService {
	return s.templateSvc
}

// GetInventoryService 获取清单服务
func (s *Service) GetInventoryService() *InventoryService {
	return s.inventorySvc
}

// GetSSHKeyService 获取 SSH 密钥服务
func (s *Service) GetSSHKeyService() *SSHKeyService {
	return s.sshKeySvc
}

// GetExecutor 获取执行器
func (s *Service) GetExecutor() *TaskExecutor {
	return s.executor
}

// GetScheduleService 获取定时任务服务
func (s *Service) GetScheduleService() *ScheduleService {
	return s.scheduleSvc
}

// GetFavoriteService 获取收藏服务
func (s *Service) GetFavoriteService() *FavoriteService {
	return s.favoriteSvc
}

// GetPreflightService 获取前置检查服务
func (s *Service) GetPreflightService() *PreflightService {
	return s.preflightSvc
}

// GetEstimationService 获取任务预估服务
func (s *Service) GetEstimationService() *EstimationService {
	return s.estimationSvc
}

// GetQueueService 获取任务队列服务
func (s *Service) GetQueueService() *QueueService {
	return s.queueSvc
}

// GetTagService 获取标签服务
func (s *Service) GetTagService() *TagService {
	return s.tagSvc
}

// GetVisualizationService 获取可视化服务
func (s *Service) GetVisualizationService() *VisualizationService {
	return s.visualizationSvc
}

// CreateTask 创建并执行任务
func (s *Service) CreateTask(req model.TaskCreateRequest, userID uint) (*model.AnsibleTask, error) {
	// 验证请求
	if err := s.validateTaskCreateRequest(req); err != nil {
		return nil, err
	}

	// 获取 playbook 内容
	playbookContent := req.PlaybookContent

	// 如果指定了模板，使用模板内容
	var template *model.AnsibleTemplate
	if req.TemplateID != nil {
		t, err := s.templateSvc.GetTemplate(*req.TemplateID)
		if err != nil {
			return nil, fmt.Errorf("failed to get template: %w", err)
		}
		template = t

		playbookContent = template.PlaybookContent

		// 如果模板定义了变量，验证提供的变量
		if len(template.Variables) > 0 && len(req.ExtraVars) > 0 {
			if err := s.templateSvc.ValidateTemplateVariables(*req.TemplateID, req.ExtraVars); err != nil {
				return nil, fmt.Errorf("template variable validation failed: %w", err)
			}
		}
		
		// 验证必需变量是否都已提供
		if len(template.RequiredVars) > 0 {
			missingVars := s.validateRequiredVariables(template.RequiredVars, req.ExtraVars)
			if len(missingVars) > 0 {
				s.logger.Warningf("Task creation: missing required variables: %v", missingVars)
				return nil, fmt.Errorf("missing required variables: %v", missingVars)
			}
		}
	}

	// 验证 inventory
	if req.InventoryID == nil {
		return nil, fmt.Errorf("inventory_id is required")
	}

	if _, err := s.inventorySvc.GetInventory(*req.InventoryID); err != nil {
		return nil, fmt.Errorf("invalid inventory: %w", err)
	}

	// 验证 playbook
	if err := s.templateSvc.ValidatePlaybook(playbookContent); err != nil {
		return nil, fmt.Errorf("invalid playbook: %w", err)
	}

	// 设置任务优先级（默认为 medium）
	priority := req.Priority
	if priority == "" {
		priority = string(model.TaskPriorityMedium)
	}
	// 验证优先级有效性
	if priority != string(model.TaskPriorityHigh) && 
	   priority != string(model.TaskPriorityMedium) && 
	   priority != string(model.TaskPriorityLow) {
		priority = string(model.TaskPriorityMedium)
	}
	
	now := time.Now()
	
	// 创建任务
	task := &model.AnsibleTask{
		Name:            req.Name,
		TemplateID:      req.TemplateID,
		ClusterID:       req.ClusterID,
		InventoryID:     req.InventoryID,
		Status:          model.AnsibleTaskStatusPending,
		UserID:          userID,
		PlaybookContent: playbookContent,
		ExtraVars:       req.ExtraVars,
		DryRun:          req.DryRun,
		BatchConfig:     req.BatchConfig,
		TimeoutSeconds:  req.TimeoutSeconds,
		Priority:        priority,
		QueuedAt:        &now,
	}
	
	// 如果启用了分批执行，初始化批次状态
	if task.IsBatchEnabled() {
		task.BatchStatus = "pending"
		task.CurrentBatch = 0
		s.logger.Infof("Task %s: Batch execution enabled - size: %d, percent: %d%%", 
			task.Name, task.BatchConfig.BatchSize, task.BatchConfig.BatchPercent)
	}

	if err := s.db.Create(task).Error; err != nil {
		s.logger.Errorf("Failed to create task: %v", err)
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.Infof("Created task: %s (ID: %d) by user %d", task.Name, task.ID, userID)
	
	// 添加到任务历史（用于快速重新执行）
	go func() {
		if err := s.favoriteSvc.AddOrUpdateTaskHistory(userID, task); err != nil {
			s.logger.Errorf("Failed to add task history: %v", err)
			// 历史记录失败不影响任务执行
		}
	}()

	// 异步执行任务
	go func() {
		if err := s.executor.ExecuteTask(task.ID); err != nil {
			s.logger.Errorf("Failed to execute task %d: %v", task.ID, err)
			
			// 更新任务状态为失败
			task.Status = model.AnsibleTaskStatusFailed
			task.ErrorMsg = err.Error()
			if err := s.db.Save(task).Error; err != nil {
				s.logger.Errorf("Failed to update task status: %v", err)
			}
		}
	}()

	return task, nil
}

// GetTask 获取任务详情
func (s *Service) GetTask(id uint) (*model.AnsibleTask, error) {
	var task model.AnsibleTask

	if err := s.db.Preload("Template").
		Preload("Cluster").
		Preload("Inventory").
		Preload("User").
		First(&task, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found")
		}
		s.logger.Errorf("Failed to get task %d: %v", id, err)
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	return &task, nil
}

// ListTasks 列出任务
func (s *Service) ListTasks(req model.TaskListRequest, userID uint) ([]model.AnsibleTask, int64, error) {
	var tasks []model.AnsibleTask
	var total int64

	query := s.db.Model(&model.AnsibleTask{})

	// 过滤条件
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}

	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	if req.TemplateID > 0 {
		query = query.Where("template_id = ?", req.TemplateID)
	}

	if req.Keyword != "" {
		query = query.Where("name LIKE ?", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorf("Failed to count tasks: %v", err)
		return nil, 0, fmt.Errorf("failed to count tasks: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询数据（包含关联）
	if err := query.Preload("Template").
		Preload("Cluster").
		Preload("Inventory").
		Preload("User").
		Order("created_at DESC").
		Find(&tasks).Error; err != nil {
		s.logger.Errorf("Failed to list tasks: %v", err)
		return nil, 0, fmt.Errorf("failed to list tasks: %w", err)
	}

	return tasks, total, nil
}

// GetTaskLogs 获取任务日志
// GetTaskFullLog 获取任务的完整日志（从 task.FullLog 字段）
func (s *Service) GetTaskFullLog(taskID uint) (string, error) {
	var task model.AnsibleTask
	if err := s.db.Select("full_log, log_size").First(&task, taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return "", fmt.Errorf("task not found")
		}
		s.logger.Errorf("Failed to get task full log: %v", err)
		return "", fmt.Errorf("failed to get task full log: %w", err)
	}

	return task.FullLog, nil
}

// GetTaskLogs 获取任务的重要日志（从 ansible_logs 表）
func (s *Service) GetTaskLogs(taskID uint, logType model.AnsibleLogType, limit int) ([]model.AnsibleLog, error) {
	var logs []model.AnsibleLog

	query := s.db.Where("task_id = ?", taskID)

	if logType != "" {
		query = query.Where("log_type = ?", logType)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Order("line_number ASC").Find(&logs).Error; err != nil {
		s.logger.Errorf("Failed to get task logs: %v", err)
		return nil, fmt.Errorf("failed to get task logs: %w", err)
	}

	return logs, nil
}

// CancelTask 取消任务
func (s *Service) CancelTask(taskID uint, userID uint) error {
	// 获取任务
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status != model.AnsibleTaskStatusRunning {
		return fmt.Errorf("task is not running")
	}

	// 取消执行
	if err := s.executor.CancelTask(taskID); err != nil {
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	s.logger.Infof("Task %d cancelled by user %d", taskID, userID)
	return nil
}

// RetryTask 重试失败的任务
func (s *Service) RetryTask(taskID uint, userID uint) (*model.AnsibleTask, error) {
	// 获取原任务
	originalTask, err := s.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	// 检查任务状态
	if originalTask.Status == model.AnsibleTaskStatusRunning {
		return nil, fmt.Errorf("task is still running")
	}

	// 创建新任务
	newTask := &model.AnsibleTask{
		Name:            originalTask.Name + " (Retry)",
		TemplateID:      originalTask.TemplateID,
		ClusterID:       originalTask.ClusterID,
		InventoryID:     originalTask.InventoryID,
		Status:          model.AnsibleTaskStatusPending,
		UserID:          userID,
		PlaybookContent: originalTask.PlaybookContent,
		ExtraVars:       originalTask.ExtraVars,
	}

	if err := s.db.Create(newTask).Error; err != nil {
		s.logger.Errorf("Failed to create retry task: %v", err)
		return nil, fmt.Errorf("failed to create retry task: %w", err)
	}

	s.logger.Infof("Created retry task: %s (ID: %d) from task %d by user %d", newTask.Name, newTask.ID, taskID, userID)

	// 异步执行任务
	go func() {
		if err := s.executor.ExecuteTask(newTask.ID); err != nil {
			s.logger.Errorf("Failed to execute retry task %d: %v", newTask.ID, err)
			
			// 更新任务状态为失败
			newTask.Status = model.AnsibleTaskStatusFailed
			newTask.ErrorMsg = err.Error()
			if err := s.db.Save(newTask).Error; err != nil {
				s.logger.Errorf("Failed to update retry task status: %v", err)
			}
		}
	}()

	return newTask, nil
}

// PauseBatchExecution 暂停批次执行
func (s *Service) PauseBatchExecution(taskID uint) error {
	// 获取任务
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status != model.AnsibleTaskStatusRunning {
		return fmt.Errorf("task is not running")
	}

	// 检查是否启用了分批执行
	if !task.IsBatchEnabled() {
		return fmt.Errorf("batch execution is not enabled for this task")
	}

	// 检查当前是否已经暂停
	if task.IsBatchPaused() {
		return fmt.Errorf("batch execution is already paused")
	}

	// 更新批次状态为暂停
	task.BatchStatus = "paused"
	if err := s.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to pause batch execution: %w", err)
	}

	s.logger.Infof("Batch execution paused for task %d", taskID)
	return nil
}

// ContinueBatchExecution 继续批次执行
func (s *Service) ContinueBatchExecution(taskID uint) error {
	// 获取任务
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status != model.AnsibleTaskStatusRunning {
		return fmt.Errorf("task is not running")
	}

	// 检查是否启用了分批执行
	if !task.IsBatchEnabled() {
		return fmt.Errorf("batch execution is not enabled for this task")
	}

	// 检查当前是否已经暂停
	if !task.IsBatchPaused() {
		return fmt.Errorf("batch execution is not paused")
	}

	// 更新批次状态为运行中
	task.BatchStatus = "running"
	task.CurrentBatch++ // 移动到下一批次
	if err := s.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to continue batch execution: %w", err)
	}

	s.logger.Infof("Batch execution continued for task %d, moving to batch %d/%d", 
		taskID, task.CurrentBatch, task.TotalBatches)
	
	// 触发执行器继续执行
	go func() {
		if err := s.executor.ContinueBatchExecution(taskID); err != nil {
			s.logger.Errorf("Failed to continue batch execution for task %d: %v", taskID, err)
		}
	}()
	
	return nil
}

// StopBatchExecution 停止批次执行
func (s *Service) StopBatchExecution(taskID uint) error {
	// 获取任务
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}

	// 检查任务状态
	if task.Status != model.AnsibleTaskStatusRunning {
		return fmt.Errorf("task is not running")
	}

	// 检查是否启用了分批执行
	if !task.IsBatchEnabled() {
		return fmt.Errorf("batch execution is not enabled for this task")
	}

	// 更新批次状态为停止
	task.BatchStatus = "stopped"
	task.Status = model.AnsibleTaskStatusCancelled
	finishedAt := time.Now()
	task.FinishedAt = &finishedAt
	duration := int(time.Since(*task.StartedAt).Seconds())
	task.Duration = duration
	
	if err := s.db.Save(task).Error; err != nil {
		return fmt.Errorf("failed to stop batch execution: %w", err)
	}

	s.logger.Infof("Batch execution stopped for task %d at batch %d/%d", 
		taskID, task.CurrentBatch, task.TotalBatches)
	
	// 取消执行
	if err := s.executor.CancelTask(taskID); err != nil {
		s.logger.Errorf("Failed to cancel task executor: %v", err)
	}
	
	return nil
}

// GetTaskStatus 获取任务状态
func (s *Service) GetTaskStatus(taskID uint) (map[string]interface{}, error) {
	task, err := s.GetTask(taskID)
	if err != nil {
		return nil, err
	}

	status := map[string]interface{}{
		"id":             task.ID,
		"name":           task.Name,
		"status":         task.Status,
		"started_at":     task.StartedAt,
		"finished_at":    task.FinishedAt,
		"duration":       task.Duration,
		"hosts_total":    task.HostsTotal,
		"hosts_ok":       task.HostsOk,
		"hosts_failed":   task.HostsFailed,
		"hosts_skipped":  task.HostsSkipped,
		"error_msg":      task.ErrorMsg,
		"is_running":     s.executor.IsTaskRunning(taskID),
	}

	// 如果任务正在运行，添加进度信息
	if task.Status == model.AnsibleTaskStatusRunning {
		progress := 0.0
		if task.HostsTotal > 0 {
			progress = float64(task.HostsOk+task.HostsFailed) / float64(task.HostsTotal) * 100
		}
		status["progress"] = progress
	}

	return status, nil
}

// GetStatistics 获取统计信息
func (s *Service) GetStatistics(userID uint) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 统计总任务数
	var totalTasks int64
	if err := s.db.Model(&model.AnsibleTask{}).Count(&totalTasks).Error; err != nil {
		return nil, err
	}
	stats["total_tasks"] = totalTasks

	// 统计各状态任务数
	statusCounts := make(map[string]int64)
	var results []struct {
		Status model.AnsibleTaskStatus
		Count  int64
	}
	if err := s.db.Model(&model.AnsibleTask{}).
		Select("status, count(*) as count").
		Group("status").
		Find(&results).Error; err != nil {
		return nil, err
	}

	for _, result := range results {
		statusCounts[string(result.Status)] = result.Count
	}
	stats["status_counts"] = statusCounts

	// 统计正在运行的任务数
	stats["running_tasks"] = s.executor.GetRunningTasksCount()

	// 统计模板数
	var totalTemplates int64
	if err := s.db.Model(&model.AnsibleTemplate{}).Count(&totalTemplates).Error; err != nil {
		return nil, err
	}
	stats["total_templates"] = totalTemplates

	// 统计清单数
	var totalInventories int64
	if err := s.db.Model(&model.AnsibleInventory{}).Count(&totalInventories).Error; err != nil {
		return nil, err
	}
	stats["total_inventories"] = totalInventories

	return stats, nil
}

// validateTaskCreateRequest 验证任务创建请求
func (s *Service) validateTaskCreateRequest(req model.TaskCreateRequest) error {
	if req.Name == "" {
		return fmt.Errorf("task name is required")
	}

	// 必须提供模板或 playbook 内容
	if req.TemplateID == nil && req.PlaybookContent == "" {
		return fmt.Errorf("either template_id or playbook_content is required")
	}

	// 必须提供 inventory
	if req.InventoryID == nil {
		return fmt.Errorf("inventory_id is required")
	}

	return nil
}

// validateRequiredVariables 验证必需变量是否都已提供
func (s *Service) validateRequiredVariables(requiredVars []string, providedVars model.ExtraVars) []string {
	return ansibleUtil.ValidateVariables(requiredVars, providedVars)
}

// DeleteTask 删除单个任务
// 删除时会级联删除所有关联的日志
func (s *Service) DeleteTask(taskID uint, userID uint, username string) error {
	// 获取任务
	task, err := s.GetTask(taskID)
	if err != nil {
		return err
	}

	// 检查任务状态 - 只能删除非运行中的任务
	if task.Status == model.AnsibleTaskStatusRunning || task.Status == model.AnsibleTaskStatusPending {
		return fmt.Errorf("cannot delete running or pending task")
	}

	// 开启事务处理
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 删除所有关联的日志
		if err := tx.Where("task_id = ?", taskID).Delete(&model.AnsibleLog{}).Error; err != nil {
			s.logger.Errorf("Failed to delete logs for task %d: %v", taskID, err)
			return fmt.Errorf("failed to delete task logs: %w", err)
		}

		// 2. 删除任务
		if err := tx.Delete(task).Error; err != nil {
			s.logger.Errorf("Failed to delete task %d: %v", taskID, err)
			return fmt.Errorf("failed to delete task: %w", err)
		}

		s.logger.Infof("Task %d (%s) and its logs deleted by user %s (ID: %d)", taskID, task.Name, username, userID)
		return nil
	})
}

// DeleteTasks 批量删除任务
// 删除时会级联删除所有关联的日志
func (s *Service) DeleteTasks(taskIDs []uint, userID uint, username string) (int, []string, error) {
	var successCount int
	var errors []string

	for _, taskID := range taskIDs {
		// 获取任务
		task, err := s.GetTask(taskID)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d: %v", taskID, err))
			continue
		}

		// 检查任务状态 - 只能删除非运行中的任务
		if task.Status == model.AnsibleTaskStatusRunning || task.Status == model.AnsibleTaskStatusPending {
			errors = append(errors, fmt.Sprintf("Task %d: cannot delete running or pending task", taskID))
			continue
		}

		// 开启事务处理
		err = s.db.Transaction(func(tx *gorm.DB) error {
			// 1. 删除所有关联的日志
			if err := tx.Where("task_id = ?", taskID).Delete(&model.AnsibleLog{}).Error; err != nil {
				s.logger.Errorf("Failed to delete logs for task %d: %v", taskID, err)
				return fmt.Errorf("failed to delete task logs: %w", err)
			}

			// 2. 删除任务
			if err := tx.Delete(task).Error; err != nil {
				s.logger.Errorf("Failed to delete task %d: %v", taskID, err)
				return fmt.Errorf("failed to delete task: %w", err)
			}

			return nil
		})

		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d: %v", taskID, err))
			continue
		}

		successCount++
	}

	if successCount > 0 {
		s.logger.Infof("Deleted %d tasks and their logs by user %s (ID: %d)", successCount, username, userID)
	}

	return successCount, errors, nil
}

// CleanupOldTasks 清理旧任务（可定期执行）
func (s *Service) CleanupOldTasks(daysToKeep int) error {
	// 删除指定天数前的已完成任务
	cutoffDate := time.Now().AddDate(0, 0, -daysToKeep)

	result := s.db.Where("status IN (?, ?, ?) AND finished_at < ?",
		model.AnsibleTaskStatusSuccess,
		model.AnsibleTaskStatusFailed,
		model.AnsibleTaskStatusCancelled,
		cutoffDate).
		Delete(&model.AnsibleTask{})

	if result.Error != nil {
		s.logger.Errorf("Failed to cleanup old tasks: %v", result.Error)
		return result.Error
	}

	if result.RowsAffected > 0 {
		s.logger.Infof("Cleaned up %d old tasks", result.RowsAffected)
	}

	// 清理执行器的临时文件
	s.executor.Cleanup()

	return nil
}

