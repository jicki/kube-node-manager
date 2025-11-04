package ansible

import (
	"fmt"
	"time"

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
	
	// 加载执行记录，包括工作流和任务
	if err := s.db.Where("id = ? AND user_id = ?", id, userID).
		Preload("Workflow"). // 预加载工作流（包括 DAG）
		Preload("Tasks", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC") // 按创建时间排序任务
		}).
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工作流执行不存在或无权访问")
		}
		s.logger.Errorf("Failed to get workflow execution: %v", err)
		return nil, fmt.Errorf("获取工作流执行失败: %w", err)
	}

	// 确保 Workflow.DAG 被正确加载
	if execution.Workflow != nil {
		s.logger.Infof("Loaded workflow execution %d with workflow %d (DAG nodes: %d)", 
			execution.ID, 
			execution.Workflow.ID, 
			len(execution.Workflow.DAG.Nodes))
	}

	return &execution, nil
}

// ListWorkflowExecutions 查询工作流执行列表
func (s *WorkflowService) ListWorkflowExecutions(userID uint, req *model.WorkflowExecutionListRequest) ([]map[string]interface{}, int64, error) {
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

	// 为每个执行计算任务统计
	result := make([]map[string]interface{}, 0, len(executions))
	for _, execution := range executions {
		// 查询该执行的所有任务
		var tasks []model.AnsibleTask
		s.db.Where("workflow_execution_id = ?", execution.ID).Find(&tasks)

		// 统计任务状态
		totalTasks := len(tasks)
		completedTasks := 0
		failedTasks := 0
		for _, task := range tasks {
			if task.Status == model.AnsibleTaskStatusSuccess {
				completedTasks++
			} else if task.Status == model.AnsibleTaskStatusFailed {
				failedTasks++
			}
		}

		// 计算耗时（秒）
		var duration int64
		if execution.FinishedAt != nil {
			duration = int64(execution.FinishedAt.Sub(execution.StartedAt).Seconds())
		} else if execution.Status == "running" {
			duration = int64(time.Since(execution.StartedAt).Seconds())
		}

		// 构建返回数据
		execData := map[string]interface{}{
			"id":              execution.ID,
			"workflow_id":     execution.WorkflowID,
			"status":          execution.Status,
			"started_at":      execution.StartedAt,
			"finished_at":     execution.FinishedAt,
			"error_message":   execution.ErrorMessage,
			"user_id":         execution.UserID,
			"created_at":      execution.CreatedAt,
			"updated_at":      execution.UpdatedAt,
			"workflow":        execution.Workflow,
			"total_tasks":     totalTasks,
			"completed_tasks": completedTasks,
			"failed_tasks":    failedTasks,
			"duration":        duration,
		}
		result = append(result, execData)
	}

	return result, total, nil
}

// GetCompletedWorkflowStatus 获取已完成工作流的节点状态
func (s *WorkflowService) GetCompletedWorkflowStatus(executionID uint, userID uint) (map[string]string, error) {
	// 验证执行记录存在且有权访问
	var execution model.AnsibleWorkflowExecution
	if err := s.db.Where("id = ? AND user_id = ?", executionID, userID).
		Preload("Workflow").
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("执行记录不存在或无权访问")
		}
		return nil, fmt.Errorf("查询执行记录失败: %w", err)
	}

	// 如果还在运行中，返回空（应该从运行时获取）
	if execution.Status == "running" || execution.Status == "pending" {
		return nil, nil
	}

	// 获取所有关联的任务
	var tasks []model.AnsibleTask
	if err := s.db.Where("workflow_execution_id = ?", executionID).
		Find(&tasks).Error; err != nil {
		s.logger.Errorf("Failed to get workflow tasks: %v", err)
		return nil, fmt.Errorf("查询工作流任务失败: %w", err)
	}

	// 构建节点状态映射
	nodeStatus := make(map[string]string)
	for _, task := range tasks {
		// 将任务状态映射到节点状态
		switch task.Status {
		case model.AnsibleTaskStatusSuccess:
			nodeStatus[task.NodeID] = "success"
		case model.AnsibleTaskStatusFailed:
			nodeStatus[task.NodeID] = "failed"
		case model.AnsibleTaskStatusRunning:
			nodeStatus[task.NodeID] = "running"
		case model.AnsibleTaskStatusPending:
			nodeStatus[task.NodeID] = "pending"
		default:
			nodeStatus[task.NodeID] = "pending"
		}
	}

	// 如果有 DAG 信息，补充开始和结束节点的状态
	if execution.Workflow != nil && execution.Workflow.DAG != nil {
		for _, node := range execution.Workflow.DAG.Nodes {
			if node.Type == "start" || node.Type == "end" {
				// 如果工作流已完成，开始和结束节点都标记为成功
				if execution.Status == "success" {
					nodeStatus[node.ID] = "success"
				} else if execution.Status == "failed" {
					// 开始节点成功，结束节点失败
					if node.Type == "start" {
						nodeStatus[node.ID] = "success"
					} else {
						nodeStatus[node.ID] = "failed"
					}
				} else if execution.Status == "cancelled" {
					nodeStatus[node.ID] = "cancelled"
				}
			}
		}
	}

	return nodeStatus, nil
}

