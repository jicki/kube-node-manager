package ansible

import (
	"fmt"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"gorm.io/gorm"
)

// WorkflowService 工作流服务
type WorkflowService struct {
	db        *gorm.DB
	logger    *logger.Logger
	validator *WorkflowValidator
}

// NewWorkflowService 创建工作流服务实例
func NewWorkflowService(db *gorm.DB, logger *logger.Logger) *WorkflowService {
	return &WorkflowService{
		db:        db,
		logger:    logger,
		validator: NewWorkflowValidator(logger),
	}
}

// CreateWorkflow 创建工作流
// 验证 DAG 有效性并保存到数据库
func (s *WorkflowService) CreateWorkflow(userID uint, req *model.WorkflowCreateRequest) (*model.AnsibleWorkflow, error) {
	// 验证 DAG 结构
	if err := s.validator.ValidateDAG(req.DAG); err != nil {
		s.logger.Errorf("DAG validation failed: %v", err)
		return nil, fmt.Errorf("DAG 验证失败: %w", err)
	}

	// 创建工作流
	workflow := &model.AnsibleWorkflow{
		Name:        req.Name,
		Description: req.Description,
		DAG:         req.DAG,
		UserID:      userID,
	}

	if err := s.db.Create(workflow).Error; err != nil {
		s.logger.Errorf("Failed to create workflow: %v", err)
		return nil, fmt.Errorf("创建工作流失败: %w", err)
	}

	s.logger.Infof("Workflow created successfully: ID=%d, Name=%s, UserID=%d", workflow.ID, workflow.Name, userID)
	return workflow, nil
}

// GetWorkflow 获取工作流详情
func (s *WorkflowService) GetWorkflow(id uint, userID uint) (*model.AnsibleWorkflow, error) {
	var workflow model.AnsibleWorkflow
	
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).
		First(&workflow).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工作流不存在或无权访问")
		}
		s.logger.Errorf("Failed to get workflow: %v", err)
		return nil, fmt.Errorf("获取工作流失败: %w", err)
	}

	return &workflow, nil
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(id uint, userID uint, req *model.WorkflowUpdateRequest) (*model.AnsibleWorkflow, error) {
	// 获取现有工作流
	workflow, err := s.GetWorkflow(id, userID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != "" {
		workflow.Name = req.Name
	}
	if req.Description != "" {
		workflow.Description = req.Description
	}
	if req.DAG != nil {
		// 验证新的 DAG
		if err := s.validator.ValidateDAG(req.DAG); err != nil {
			s.logger.Errorf("DAG validation failed: %v", err)
			return nil, fmt.Errorf("DAG 验证失败: %w", err)
		}
		workflow.DAG = req.DAG
	}

	if err := s.db.Save(workflow).Error; err != nil {
		s.logger.Errorf("Failed to update workflow: %v", err)
		return nil, fmt.Errorf("更新工作流失败: %w", err)
	}

	s.logger.Infof("Workflow updated successfully: ID=%d", id)
	return workflow, nil
}

// DeleteWorkflow 删除工作流（软删除）
func (s *WorkflowService) DeleteWorkflow(id uint, userID uint) error {
	// 检查工作流是否存在
	workflow, err := s.GetWorkflow(id, userID)
	if err != nil {
		return err
	}

	// 检查是否有正在运行的执行
	var runningCount int64
	if err := s.db.Model(&model.AnsibleWorkflowExecution{}).
		Where("workflow_id = ? AND status = ?", id, "running").
		Count(&runningCount).Error; err != nil {
		s.logger.Errorf("Failed to check running executions: %v", err)
		return fmt.Errorf("检查执行状态失败: %w", err)
	}

	if runningCount > 0 {
		return fmt.Errorf("工作流有正在运行的执行，无法删除")
	}

	// 软删除
	if err := s.db.Delete(workflow).Error; err != nil {
		s.logger.Errorf("Failed to delete workflow: %v", err)
		return fmt.Errorf("删除工作流失败: %w", err)
	}

	s.logger.Infof("Workflow deleted successfully: ID=%d", id)
	return nil
}

// ListWorkflows 查询工作流列表
func (s *WorkflowService) ListWorkflows(userID uint, req *model.WorkflowListRequest) ([]model.AnsibleWorkflow, int64, error) {
	// 设置默认分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&model.AnsibleWorkflow{}).Where("user_id = ?", userID)

	// 关键词搜索
	if req.Keyword != "" {
		keyword := "%" + req.Keyword + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", keyword, keyword)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorf("Failed to count workflows: %v", err)
		return nil, 0, fmt.Errorf("查询工作流总数失败: %w", err)
	}

	// 查询列表
	var workflows []model.AnsibleWorkflow
	offset := (page - 1) * pageSize
	if err := query.Order("updated_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&workflows).Error; err != nil {
		s.logger.Errorf("Failed to list workflows: %v", err)
		return nil, 0, fmt.Errorf("查询工作流列表失败: %w", err)
	}

	return workflows, total, nil
}

// GetWorkflowExecution 获取工作流执行详情
func (s *WorkflowService) GetWorkflowExecution(id uint, userID uint) (*model.AnsibleWorkflowExecution, error) {
	var execution model.AnsibleWorkflowExecution
	
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).
		Preload("Workflow").
		Preload("Tasks").
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工作流执行不存在或无权访问")
		}
		s.logger.Errorf("Failed to get workflow execution: %v", err)
		return nil, fmt.Errorf("获取工作流执行失败: %w", err)
	}

	return &execution, nil
}

// ListWorkflowExecutions 查询工作流执行列表
func (s *WorkflowService) ListWorkflowExecutions(userID uint, req *model.WorkflowExecutionListRequest) ([]model.AnsibleWorkflowExecution, int64, error) {
	// 设置默认分页参数
	page := req.Page
	if page < 1 {
		page = 1
	}
	pageSize := req.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	query := s.db.Model(&model.AnsibleWorkflowExecution{}).Where("user_id = ?", userID)

	// 按工作流 ID 过滤
	if req.WorkflowID > 0 {
		query = query.Where("workflow_id = ?", req.WorkflowID)
	}

	// 按状态过滤
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 查询总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.logger.Errorf("Failed to count workflow executions: %v", err)
		return nil, 0, fmt.Errorf("查询执行记录总数失败: %w", err)
	}

	// 查询列表
	var executions []model.AnsibleWorkflowExecution
	offset := (page - 1) * pageSize
	if err := query.Preload("Workflow").
		Order("started_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&executions).Error; err != nil {
		s.logger.Errorf("Failed to list workflow executions: %v", err)
		return nil, 0, fmt.Errorf("查询执行记录列表失败: %w", err)
	}

	return executions, total, nil
}

