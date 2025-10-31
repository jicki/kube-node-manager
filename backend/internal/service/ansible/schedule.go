package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ScheduleService 定时任务调度服务
type ScheduleService struct {
	db       *gorm.DB
	logger   *logger.Logger
	cron     *cron.Cron
	service  *Service
	jobs     map[uint]cron.EntryID // schedule_id -> cron.EntryID 映射
	mu       sync.RWMutex
	maxTasks int // 最大活跃定时任务数
}

// NewScheduleService 创建定时任务调度服务实例
func NewScheduleService(db *gorm.DB, logger *logger.Logger, ansibleSvc *Service) *ScheduleService {
	// 创建 cron 调度器（使用秒级精度）
	c := cron.New(cron.WithSeconds())

	return &ScheduleService{
		db:       db,
		logger:   logger,
		cron:     c,
		service:  ansibleSvc,
		jobs:     make(map[uint]cron.EntryID),
		maxTasks: 100, // 最多 100 个活跃定时任务
	}
}

// Start 启动定时任务调度器
func (s *ScheduleService) Start() error {
	s.logger.Info("Starting Ansible schedule service...")

	// 从数据库加载所有已启用的定时任务
	var schedules []model.AnsibleSchedule
	if err := s.db.Where("enabled = ? AND deleted_at IS NULL", true).Find(&schedules).Error; err != nil {
		return fmt.Errorf("failed to load schedules: %w", err)
	}

	// 注册所有定时任务
	for _, schedule := range schedules {
		if err := s.AddSchedule(&schedule); err != nil {
			s.logger.Errorf("Failed to add schedule %d (%s): %v", schedule.ID, schedule.Name, err)
		}
	}

	// 启动 cron 调度器
	s.cron.Start()
	s.logger.Infof("Ansible schedule service started with %d active schedules", len(schedules))

	return nil
}

// Stop 停止定时任务调度器
func (s *ScheduleService) Stop() {
	s.logger.Info("Stopping Ansible schedule service...")
	ctx := s.cron.Stop()
	<-ctx.Done()
	s.logger.Info("Ansible schedule service stopped")
}

// AddSchedule 添加定时任务到调度器
func (s *ScheduleService) AddSchedule(schedule *model.AnsibleSchedule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 如果已存在，先删除
	if entryID, exists := s.jobs[schedule.ID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, schedule.ID)
	}

	// 解析 Cron 表达式
	_, err := cron.ParseStandard(schedule.CronExpr)
	if err != nil {
		return fmt.Errorf("invalid cron expression: %w", err)
	}

	// 添加定时任务
	entryID, err := s.cron.AddFunc(schedule.CronExpr, func() {
		s.executeSchedule(schedule.ID)
	})
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	// 保存 entryID
	s.jobs[schedule.ID] = entryID

	// 更新下次执行时间
	entry := s.cron.Entry(entryID)
	nextRun := entry.Next
	schedule.NextRunAt = &nextRun
	if err := s.db.Model(schedule).Update("next_run_at", nextRun).Error; err != nil {
		s.logger.Errorf("Failed to update next_run_at for schedule %d: %v", schedule.ID, err)
	}

	s.logger.Infof("Added schedule %d (%s) with cron expression: %s, next run: %s",
		schedule.ID, schedule.Name, schedule.CronExpr, nextRun.Format(time.RFC3339))

	return nil
}

// RemoveSchedule 从调度器中移除定时任务
func (s *ScheduleService) RemoveSchedule(scheduleID uint) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if entryID, exists := s.jobs[scheduleID]; exists {
		s.cron.Remove(entryID)
		delete(s.jobs, scheduleID)
		s.logger.Infof("Removed schedule %d from cron", scheduleID)
	}
}

// executeSchedule 执行定时任务
func (s *ScheduleService) executeSchedule(scheduleID uint) {
	s.logger.Infof("Executing schedule %d", scheduleID)

	// 获取定时任务详情
	var schedule model.AnsibleSchedule
	if err := s.db.Preload("Template").Preload("Inventory").First(&schedule, scheduleID).Error; err != nil {
		s.logger.Errorf("Failed to get schedule %d: %v", scheduleID, err)
		return
	}

	// 检查模板和清单是否存在
	if schedule.Template == nil {
		s.logger.Errorf("Schedule %d: template not found", scheduleID)
		return
	}
	if schedule.Inventory == nil {
		s.logger.Errorf("Schedule %d: inventory not found", scheduleID)
		return
	}

	// 创建任务请求
	taskReq := model.TaskCreateRequest{
		Name:            fmt.Sprintf("[定时任务] %s", schedule.Name),
		TemplateID:      &schedule.TemplateID,
		ClusterID:       schedule.ClusterID,
		InventoryID:     &schedule.InventoryID,
		ExtraVars:       schedule.ExtraVars,
	}

	// 创建并执行任务
	task, err := s.service.CreateTask(taskReq, schedule.UserID)
	if err != nil {
		s.logger.Errorf("Failed to create task for schedule %d: %v", scheduleID, err)
		return
	}

	s.logger.Infof("Schedule %d executed successfully, created task %d", scheduleID, task.ID)

	// 更新定时任务统计
	now := time.Now()
	updates := map[string]interface{}{
		"last_run_at": now,
		"run_count":   gorm.Expr("run_count + 1"),
	}

	// 更新下次执行时间
	s.mu.RLock()
	if entryID, exists := s.jobs[scheduleID]; exists {
		entry := s.cron.Entry(entryID)
		updates["next_run_at"] = entry.Next
	}
	s.mu.RUnlock()

	if err := s.db.Model(&model.AnsibleSchedule{}).Where("id = ?", scheduleID).Updates(updates).Error; err != nil {
		s.logger.Errorf("Failed to update schedule %d statistics: %v", scheduleID, err)
	}
}

// CreateSchedule 创建定时任务
func (s *ScheduleService) CreateSchedule(req model.ScheduleCreateRequest, userID uint) (*model.AnsibleSchedule, error) {
	// 检查活跃定时任务数量
	var count int64
	if err := s.db.Model(&model.AnsibleSchedule{}).Where("enabled = ? AND deleted_at IS NULL", true).Count(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to count active schedules: %w", err)
	}

	if count >= int64(s.maxTasks) {
		return nil, fmt.Errorf("maximum number of active schedules (%d) reached", s.maxTasks)
	}

	// 验证 Cron 表达式
	_, err := cron.ParseStandard(req.CronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %w", err)
	}

	// 验证模板是否存在
	var template model.AnsibleTemplate
	if err := s.db.First(&template, req.TemplateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to verify template: %w", err)
	}

	// 验证清单是否存在
	var inventory model.AnsibleInventory
	if err := s.db.First(&inventory, req.InventoryID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("inventory not found")
		}
		return nil, fmt.Errorf("failed to verify inventory: %w", err)
	}

	// 创建定时任务
	schedule := &model.AnsibleSchedule{
		Name:        req.Name,
		Description: req.Description,
		TemplateID:  req.TemplateID,
		InventoryID: req.InventoryID,
		ClusterID:   req.ClusterID,
		CronExpr:    req.CronExpr,
		ExtraVars:   req.ExtraVars,
		Enabled:     req.Enabled,
		UserID:      userID,
	}

	if err := s.db.Create(schedule).Error; err != nil {
		return nil, fmt.Errorf("failed to create schedule: %w", err)
	}

	s.logger.Infof("Created schedule: %s (ID: %d) by user %d", schedule.Name, schedule.ID, userID)

	// 如果启用，添加到调度器
	if schedule.Enabled {
		if err := s.AddSchedule(schedule); err != nil {
			s.logger.Errorf("Failed to add schedule to cron: %v", err)
			// 不返回错误，因为记录已创建
		}
		// 重新查询以获取更新后的 next_run_at
		if err := s.db.First(schedule, schedule.ID).Error; err != nil {
			s.logger.Errorf("Failed to refresh schedule after adding to cron: %v", err)
		}
	}

	return schedule, nil
}

// GetSchedule 获取定时任务详情
func (s *ScheduleService) GetSchedule(id uint) (*model.AnsibleSchedule, error) {
	var schedule model.AnsibleSchedule
	if err := s.db.Preload("Template").Preload("Inventory").Preload("Cluster").Preload("User").First(&schedule, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("schedule not found")
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return &schedule, nil
}

// ListSchedules 列出定时任务
func (s *ScheduleService) ListSchedules(req model.ScheduleListRequest) ([]model.AnsibleSchedule, int64, error) {
	var schedules []model.AnsibleSchedule
	var total int64

	query := s.db.Model(&model.AnsibleSchedule{})

	// 过滤条件
	if req.Enabled != nil {
		query = query.Where("enabled = ?", *req.Enabled)
	}

	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}

	if req.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count schedules: %w", err)
	}

	// 分页
	if req.Page > 0 && req.PageSize > 0 {
		offset := (req.Page - 1) * req.PageSize
		query = query.Offset(offset).Limit(req.PageSize)
	}

	// 查询数据
	if err := query.Preload("Template").Preload("Inventory").Preload("Cluster").Preload("User").
		Order("created_at DESC").Find(&schedules).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list schedules: %w", err)
	}

	return schedules, total, nil
}

// UpdateSchedule 更新定时任务
func (s *ScheduleService) UpdateSchedule(id uint, req model.ScheduleUpdateRequest) (*model.AnsibleSchedule, error) {
	schedule, err := s.GetSchedule(id)
	if err != nil {
		return nil, err
	}

	// 更新字段
	updates := make(map[string]interface{})

	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.TemplateID > 0 {
		// 验证模板
		var template model.AnsibleTemplate
		if err := s.db.First(&template, req.TemplateID).Error; err != nil {
			return nil, fmt.Errorf("template not found")
		}
		updates["template_id"] = req.TemplateID
	}
	if req.InventoryID > 0 {
		// 验证清单
		var inventory model.AnsibleInventory
		if err := s.db.First(&inventory, req.InventoryID).Error; err != nil {
			return nil, fmt.Errorf("inventory not found")
		}
		updates["inventory_id"] = req.InventoryID
	}
	if req.ClusterID != nil {
		updates["cluster_id"] = req.ClusterID
	}
	if req.CronExpr != "" {
		// 验证 Cron 表达式
		if _, err := cron.ParseStandard(req.CronExpr); err != nil {
			return nil, fmt.Errorf("invalid cron expression: %w", err)
		}
		updates["cron_expr"] = req.CronExpr
	}
	if req.ExtraVars != nil {
		updates["extra_vars"] = req.ExtraVars
	}
	if req.Enabled != nil {
		updates["enabled"] = *req.Enabled
	}

	// 更新数据库
	if len(updates) > 0 {
		if err := s.db.Model(schedule).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("failed to update schedule: %w", err)
		}
	}

	// 重新加载数据
	schedule, err = s.GetSchedule(id)
	if err != nil {
		return nil, err
	}

	// 更新调度器
	if schedule.Enabled {
		if err := s.AddSchedule(schedule); err != nil {
			s.logger.Errorf("Failed to update schedule in cron: %v", err)
		}
		// 重新查询以获取更新后的 next_run_at
		schedule, err = s.GetSchedule(id)
		if err != nil {
			s.logger.Errorf("Failed to refresh schedule after update: %v", err)
		}
	} else {
		s.RemoveSchedule(id)
		// 清除 next_run_at
		if err := s.db.Model(schedule).Update("next_run_at", nil).Error; err != nil {
			s.logger.Errorf("Failed to clear next_run_at: %v", err)
		}
		schedule.NextRunAt = nil
	}

	s.logger.Infof("Updated schedule %d", id)
	return schedule, nil
}

// DeleteSchedule 删除定时任务
func (s *ScheduleService) DeleteSchedule(id uint) error {
	schedule, err := s.GetSchedule(id)
	if err != nil {
		return err
	}

	// 从调度器移除
	s.RemoveSchedule(id)

	// 软删除
	if err := s.db.Delete(schedule).Error; err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	s.logger.Infof("Deleted schedule %d (%s)", id, schedule.Name)
	return nil
}

// ToggleSchedule 启用/禁用定时任务
func (s *ScheduleService) ToggleSchedule(id uint, enabled bool) (*model.AnsibleSchedule, error) {
	schedule, err := s.GetSchedule(id)
	if err != nil {
		return nil, err
	}

	// 更新状态
	if err := s.db.Model(schedule).Update("enabled", enabled).Error; err != nil {
		return nil, fmt.Errorf("failed to toggle schedule: %w", err)
	}

	// 更新调度器
	if enabled {
		schedule.Enabled = true
		if err := s.AddSchedule(schedule); err != nil {
			return nil, fmt.Errorf("failed to enable schedule in cron: %w", err)
		}
		// 重新查询以获取更新后的 next_run_at
		if err := s.db.Preload("Template").Preload("Inventory").Preload("Cluster").Preload("User").First(schedule, id).Error; err != nil {
			s.logger.Errorf("Failed to refresh schedule after enabling: %v", err)
		}
		s.logger.Infof("Enabled schedule %d", id)
	} else {
		s.RemoveSchedule(id)
		// 清除 next_run_at
		if err := s.db.Model(schedule).Update("next_run_at", nil).Error; err != nil {
			s.logger.Errorf("Failed to clear next_run_at: %v", err)
		}
		schedule.NextRunAt = nil
		s.logger.Infof("Disabled schedule %d", id)
	}

	return schedule, nil
}

// RunNow 立即执行定时任务
func (s *ScheduleService) RunNow(id uint) error {
	// 验证定时任务存在
	if _, err := s.GetSchedule(id); err != nil {
		return err
	}

	// 异步执行
	go s.executeSchedule(id)

	s.logger.Infof("Manually triggered schedule %d", id)
	return nil
}

// GetActiveSchedulesCount 获取活跃定时任务数量
func (s *ScheduleService) GetActiveSchedulesCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.jobs)
}

