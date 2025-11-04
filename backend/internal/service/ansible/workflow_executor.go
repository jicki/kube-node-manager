package ansible

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"gorm.io/gorm"
)

// WorkflowExecutor DAG 执行引擎
type WorkflowExecutor struct {
	db            *gorm.DB
	logger        *logger.Logger
	validator     *WorkflowValidator
	taskExecutor  *TaskExecutor
	runningWFs    map[uint]*RunningWorkflow
	mu            sync.Mutex
}

// RunningWorkflow 正在运行的工作流
type RunningWorkflow struct {
	ExecutionID   uint
	WorkflowID    uint
	Context       context.Context
	Cancel        context.CancelFunc
	NodeStatus    map[string]string // nodeID -> status (pending/running/success/failed)
	NodeTaskID    map[string]uint   // nodeID -> taskID
	mu            sync.RWMutex
}

// NewWorkflowExecutor 创建执行引擎实例
func NewWorkflowExecutor(db *gorm.DB, logger *logger.Logger, taskExecutor *TaskExecutor) *WorkflowExecutor {
	return &WorkflowExecutor{
		db:           db,
		logger:       logger,
		validator:    NewWorkflowValidator(logger),
		taskExecutor: taskExecutor,
		runningWFs:   make(map[uint]*RunningWorkflow),
	}
}

// ExecuteWorkflow 执行工作流
// 创建执行记录，按拓扑顺序调度任务
func (e *WorkflowExecutor) ExecuteWorkflow(workflowID uint, userID uint) (*model.AnsibleWorkflowExecution, error) {
	// 获取工作流定义
	var workflow model.AnsibleWorkflow
	if err := e.db.First(&workflow, workflowID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("工作流不存在")
		}
		return nil, fmt.Errorf("获取工作流失败: %w", err)
	}

	// 验证权限
	if workflow.UserID != userID {
		return nil, fmt.Errorf("无权执行该工作流")
	}

	// 验证 DAG
	if err := e.validator.ValidateDAG(workflow.DAG); err != nil {
		return nil, fmt.Errorf("DAG 验证失败: %w", err)
	}

	// 创建执行记录
	execution := &model.AnsibleWorkflowExecution{
		WorkflowID: workflowID,
		Status:     "running",
		StartedAt:  time.Now(),
		UserID:     userID,
	}

	if err := e.db.Create(execution).Error; err != nil {
		e.logger.Errorf("Failed to create workflow execution: %v", err)
		return nil, fmt.Errorf("创建执行记录失败: %w", err)
	}

	// 创建运行上下文
	ctx, cancel := context.WithCancel(context.Background())
	runningWF := &RunningWorkflow{
		ExecutionID: execution.ID,
		WorkflowID:  workflowID,
		Context:     ctx,
		Cancel:      cancel,
		NodeStatus:  make(map[string]string),
		NodeTaskID:  make(map[string]uint),
	}

	// 初始化所有节点状态为 pending
	for _, node := range workflow.DAG.Nodes {
		runningWF.NodeStatus[node.ID] = "pending"
	}

	e.mu.Lock()
	e.runningWFs[execution.ID] = runningWF
	e.mu.Unlock()

	// 异步执行工作流
	go e.executeWorkflowAsync(ctx, &workflow, execution, runningWF)

	e.logger.Infof("Workflow execution started: ExecutionID=%d, WorkflowID=%d", execution.ID, workflowID)
	return execution, nil
}

// executeWorkflowAsync 异步执行工作流
func (e *WorkflowExecutor) executeWorkflowAsync(ctx context.Context, workflow *model.AnsibleWorkflow, execution *model.AnsibleWorkflowExecution, runningWF *RunningWorkflow) {
	defer func() {
		e.mu.Lock()
		delete(e.runningWFs, execution.ID)
		e.mu.Unlock()
	}()

	// 拓扑排序
	sortedNodes, err := e.validator.TopologicalSort(workflow.DAG)
	if err != nil {
		e.failWorkflowExecution(execution, fmt.Sprintf("拓扑排序失败: %v", err))
		return
	}

	// 构建依赖图
	dependencies := e.buildDependencyGraph(workflow.DAG)

	// 按拓扑顺序执行节点
	for _, nodeID := range sortedNodes {
		// 检查上下文是否被取消
		select {
		case <-ctx.Done():
			e.logger.Infof("Workflow execution cancelled: ExecutionID=%d", execution.ID)
			e.failWorkflowExecution(execution, "工作流执行被取消")
			return
		default:
		}

		// 获取节点
		node := e.getNodeByID(workflow.DAG, nodeID)
		if node == nil {
			e.failWorkflowExecution(execution, fmt.Sprintf("节点不存在: %s", nodeID))
			return
		}

		// 跳过开始和结束节点
		if node.Type == "start" || node.Type == "end" {
			runningWF.mu.Lock()
			runningWF.NodeStatus[nodeID] = "success"
			runningWF.mu.Unlock()
			continue
		}

		// 等待依赖节点完成
		if err := e.waitForDependencies(ctx, runningWF, dependencies[nodeID]); err != nil {
			e.failWorkflowExecution(execution, fmt.Sprintf("等待依赖失败: %v", err))
			return
		}

		// 检查依赖节点是否都成功
		if !e.checkDependenciesSuccess(runningWF, dependencies[nodeID]) {
			e.logger.Warningf("Node %s skipped due to failed dependencies", nodeID)
			runningWF.mu.Lock()
			runningWF.NodeStatus[nodeID] = "skipped"
			runningWF.mu.Unlock()
			continue
		}

		// 执行任务节点
		if node.Type == "task" {
			if err := e.executeTaskNode(ctx, execution, node, runningWF); err != nil {
				e.failWorkflowExecution(execution, fmt.Sprintf("执行任务失败: %v", err))
				return
			}
		}
	}

	// 所有节点执行完成
	e.completeWorkflowExecution(execution)
}

// buildDependencyGraph 构建依赖图
func (e *WorkflowExecutor) buildDependencyGraph(dag *model.WorkflowDAG) map[string][]string {
	dependencies := make(map[string][]string)

	for _, edge := range dag.Edges {
		dependencies[edge.Target] = append(dependencies[edge.Target], edge.Source)
	}

	return dependencies
}

// getNodeByID 根据 ID 获取节点
func (e *WorkflowExecutor) getNodeByID(dag *model.WorkflowDAG, nodeID string) *model.WorkflowNode {
	for i := range dag.Nodes {
		if dag.Nodes[i].ID == nodeID {
			return &dag.Nodes[i]
		}
	}
	return nil
}

// waitForDependencies 等待依赖节点完成
func (e *WorkflowExecutor) waitForDependencies(ctx context.Context, runningWF *RunningWorkflow, deps []string) error {
	if len(deps) == 0 {
		return nil
	}

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(30 * time.Minute) // 30 分钟超时
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上下文已取消")
		case <-timeout.C:
			return fmt.Errorf("等待依赖超时")
		case <-ticker.C:
			allCompleted := true
			runningWF.mu.RLock()
			for _, depID := range deps {
				status := runningWF.NodeStatus[depID]
				if status != "success" && status != "failed" && status != "skipped" {
					allCompleted = false
					break
				}
			}
			runningWF.mu.RUnlock()

			if allCompleted {
				return nil
			}
		}
	}
}

// checkDependenciesSuccess 检查依赖节点是否都成功
func (e *WorkflowExecutor) checkDependenciesSuccess(runningWF *RunningWorkflow, deps []string) bool {
	runningWF.mu.RLock()
	defer runningWF.mu.RUnlock()

	for _, depID := range deps {
		if runningWF.NodeStatus[depID] != "success" {
			return false
		}
	}
	return true
}

// executeTaskNode 执行任务节点
func (e *WorkflowExecutor) executeTaskNode(ctx context.Context, execution *model.AnsibleWorkflowExecution, node *model.WorkflowNode, runningWF *RunningWorkflow) error {
	e.logger.Infof("Executing task node: %s (%s)", node.ID, node.Label)

	// 标记节点为运行中
	runningWF.mu.Lock()
	runningWF.NodeStatus[node.ID] = "running"
	runningWF.mu.Unlock()

	// 构建依赖列表
	var dependsOn model.StringArray
	deps := e.buildDependencyGraph(&model.WorkflowDAG{
		Nodes: []model.WorkflowNode{*node},
		Edges: []model.WorkflowEdge{},
	})
	if len(deps[node.ID]) > 0 {
		dependsOn = deps[node.ID]
	}

	// 创建任务
	task := &model.AnsibleTask{
		Name:                node.TaskConfig.Name,
		TemplateID:          node.TaskConfig.TemplateID,
		ClusterID:           node.TaskConfig.ClusterID,
		InventoryID:         node.TaskConfig.InventoryID,
		Status:              model.AnsibleTaskStatusPending,
		UserID:              execution.UserID,
		PlaybookContent:     node.TaskConfig.PlaybookContent,
		ExtraVars:           node.TaskConfig.ExtraVars,
		DryRun:              node.TaskConfig.DryRun,
		TimeoutSeconds:      node.TaskConfig.TimeoutSeconds,
		Priority:            node.TaskConfig.Priority,
		WorkflowExecutionID: &execution.ID,
		NodeID:              node.ID,
		DependsOn:           dependsOn,
	}

	// 保存任务
	if err := e.db.Create(task).Error; err != nil {
		e.logger.Errorf("Failed to create task for node %s: %v", node.ID, err)
		runningWF.mu.Lock()
		runningWF.NodeStatus[node.ID] = "failed"
		runningWF.mu.Unlock()
		return fmt.Errorf("创建任务失败: %w", err)
	}

	// 记录任务 ID
	runningWF.mu.Lock()
	runningWF.NodeTaskID[node.ID] = task.ID
	runningWF.mu.Unlock()

	// 执行任务
	if err := e.taskExecutor.ExecuteTask(task.ID); err != nil {
		e.logger.Errorf("Failed to execute task %d for node %s: %v", task.ID, node.ID, err)
		runningWF.mu.Lock()
		runningWF.NodeStatus[node.ID] = "failed"
		runningWF.mu.Unlock()
		return fmt.Errorf("执行任务失败: %w", err)
	}

	// 等待任务完成
	if err := e.waitForTaskCompletion(ctx, task.ID, runningWF, node.ID); err != nil {
		return err
	}

	return nil
}

// waitForTaskCompletion 等待任务完成
func (e *WorkflowExecutor) waitForTaskCompletion(ctx context.Context, taskID uint, runningWF *RunningWorkflow, nodeID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(2 * time.Hour) // 2 小时超时
	defer timeout.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("上下文已取消")
		case <-timeout.C:
			runningWF.mu.Lock()
			runningWF.NodeStatus[nodeID] = "failed"
			runningWF.mu.Unlock()
			return fmt.Errorf("任务执行超时")
		case <-ticker.C:
			var task model.AnsibleTask
			if err := e.db.First(&task, taskID).Error; err != nil {
				e.logger.Errorf("Failed to get task status: %v", err)
				continue
			}

			if task.IsCompleted() {
				runningWF.mu.Lock()
				if task.Status == model.AnsibleTaskStatusSuccess {
					runningWF.NodeStatus[nodeID] = "success"
				} else {
					runningWF.NodeStatus[nodeID] = "failed"
				}
				runningWF.mu.Unlock()

				e.logger.Infof("Task %d completed with status: %s", taskID, task.Status)
				return nil
			}
		}
	}
}

// failWorkflowExecution 标记工作流执行失败
func (e *WorkflowExecutor) failWorkflowExecution(execution *model.AnsibleWorkflowExecution, errorMsg string) {
	now := time.Now()
	execution.Status = "failed"
	execution.FinishedAt = &now
	execution.ErrorMessage = errorMsg

	if err := e.db.Save(execution).Error; err != nil {
		e.logger.Errorf("Failed to update workflow execution status: %v", err)
	}

	e.logger.Errorf("Workflow execution failed: ExecutionID=%d, Error=%s", execution.ID, errorMsg)
}

// completeWorkflowExecution 标记工作流执行完成
func (e *WorkflowExecutor) completeWorkflowExecution(execution *model.AnsibleWorkflowExecution) {
	now := time.Now()
	execution.Status = "success"
	execution.FinishedAt = &now

	if err := e.db.Save(execution).Error; err != nil {
		e.logger.Errorf("Failed to update workflow execution status: %v", err)
	}

	e.logger.Infof("Workflow execution completed successfully: ExecutionID=%d", execution.ID)
}

// CancelWorkflowExecution 取消工作流执行
func (e *WorkflowExecutor) CancelWorkflowExecution(executionID uint, userID uint) error {
	// 检查执行是否存在
	var execution model.AnsibleWorkflowExecution
	if err := e.db.Where("id = ? AND user_id = ?", executionID, userID).
		First(&execution).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("执行记录不存在或无权访问")
		}
		return fmt.Errorf("获取执行记录失败: %w", err)
	}

	// 检查是否正在运行
	if execution.Status != "running" {
		return fmt.Errorf("只能取消正在运行的工作流")
	}

	// 取消执行
	e.mu.Lock()
	runningWF, exists := e.runningWFs[executionID]
	e.mu.Unlock()

	if exists && runningWF.Cancel != nil {
		runningWF.Cancel()
	}

	// 更新状态
	now := time.Now()
	execution.Status = "cancelled"
	execution.FinishedAt = &now
	execution.ErrorMessage = "用户取消执行"

	if err := e.db.Save(&execution).Error; err != nil {
		e.logger.Errorf("Failed to update execution status: %v", err)
		return fmt.Errorf("更新执行状态失败: %w", err)
	}

	// 取消所有关联的任务
	var tasks []model.AnsibleTask
	if err := e.db.Where("workflow_execution_id = ? AND status = ?", executionID, model.AnsibleTaskStatusRunning).
		Find(&tasks).Error; err != nil {
		e.logger.Errorf("Failed to get running tasks: %v", err)
	} else {
		for _, task := range tasks {
			if err := e.taskExecutor.CancelTask(task.ID); err != nil {
				e.logger.Errorf("Failed to cancel task %d: %v", task.ID, err)
			}
		}
	}

	e.logger.Infof("Workflow execution cancelled: ExecutionID=%d", executionID)
	return nil
}

// GetWorkflowExecutionStatus 获取工作流执行状态
func (e *WorkflowExecutor) GetWorkflowExecutionStatus(executionID uint) map[string]string {
	e.mu.Lock()
	runningWF, exists := e.runningWFs[executionID]
	e.mu.Unlock()

	if !exists {
		return nil
	}

	runningWF.mu.RLock()
	defer runningWF.mu.RUnlock()

	// 复制状态映射
	status := make(map[string]string)
	for nodeID, nodeStatus := range runningWF.NodeStatus {
		status[nodeID] = nodeStatus
	}

	return status
}

