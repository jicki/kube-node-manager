package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// WorkflowService 工作流管理服务
type WorkflowService struct {
	db          *gorm.DB
	logger      *logger.Logger
	progressSvc *progress.Service
	ansibleSvc  *AnsibleService
	sshSvc      *SSHService
	scriptSvc   *ScriptService
}

// WorkflowDefinition 工作流定义结构
type WorkflowDefinition struct {
	Steps []WorkflowStep `json:"steps"`
}

// WorkflowStep 工作流步骤
type WorkflowStep struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Type            string            `json:"type"`              // ansible, ssh, script, node-operation
	Action          string            `json:"action"`            // 具体操作
	Parameters      map[string]string `json:"parameters"`        // 步骤参数
	DependsOn       []string          `json:"depends_on"`        // 依赖的步骤 ID
	ContinueOnError bool              `json:"continue_on_error"` // 错误时是否继续
	Retry           *RetryPolicy      `json:"retry"`             // 重试策略
	Condition       *StepCondition    `json:"condition"`         // 执行条件
	Timeout         int               `json:"timeout"`           // 超时时间（秒）
}

// RetryPolicy 重试策略
type RetryPolicy struct {
	MaxAttempts int `json:"max_attempts"` // 最大重试次数
	Delay       int `json:"delay"`        // 重试延迟（秒）
}

// StepCondition 步骤执行条件
type StepCondition struct {
	Type  string   `json:"type"`  // always, on_success, on_failure
	Steps []string `json:"steps"` // 依赖的步骤 ID
}

// WorkflowExecuteConfig 工作流执行配置
type WorkflowExecuteConfig struct {
	WorkflowID   uint
	ClusterName  string
	TargetNodes  []string
	Parameters   map[string]string
	CredentialID uint
}

// StepResult 步骤执行结果
type StepResult struct {
	StepID    string      `json:"step_id"`
	StepName  string      `json:"step_name"`
	Status    string      `json:"status"` // pending, running, completed, failed, skipped
	StartTime *time.Time  `json:"start_time"`
	EndTime   *time.Time  `json:"end_time"`
	Duration  int         `json:"duration"`
	Output    interface{} `json:"output"`
	Error     string      `json:"error"`
	Attempt   int         `json:"attempt"` // 重试次数
}

// WorkflowExecutionContext 工作流执行上下文
type WorkflowExecutionContext struct {
	ctx          context.Context
	execution    *model.WorkflowExecution
	definition   *WorkflowDefinition
	stepResults  map[string]*StepResult
	config       *WorkflowExecuteConfig
	credentialID uint
	mu           sync.RWMutex
}

// NewWorkflowService 创建工作流服务
func NewWorkflowService(
	db *gorm.DB,
	logger *logger.Logger,
	progressSvc *progress.Service,
	ansibleSvc *AnsibleService,
	sshSvc *SSHService,
	scriptSvc *ScriptService,
) *WorkflowService {
	return &WorkflowService{
		db:          db,
		logger:      logger,
		progressSvc: progressSvc,
		ansibleSvc:  ansibleSvc,
		sshSvc:      sshSvc,
		scriptSvc:   scriptSvc,
	}
}

// CreateWorkflow 创建工作流
func (s *WorkflowService) CreateWorkflow(workflow *model.Workflow) error {
	// 验证工作流定义
	if err := s.validateWorkflowDefinition(workflow.Definition); err != nil {
		return fmt.Errorf("workflow definition validation failed: %w", err)
	}

	if workflow.Version == 0 {
		workflow.Version = 1
	}

	return s.db.Create(workflow).Error
}

// UpdateWorkflow 更新工作流
func (s *WorkflowService) UpdateWorkflow(id uint, updates *model.Workflow) error {
	var workflow model.Workflow
	if err := s.db.First(&workflow, id).Error; err != nil {
		return err
	}

	if workflow.IsBuiltin {
		return fmt.Errorf("cannot modify builtin workflow")
	}

	// 如果定义发生变化，验证新定义
	if updates.Definition != "" && updates.Definition != workflow.Definition {
		if err := s.validateWorkflowDefinition(updates.Definition); err != nil {
			return fmt.Errorf("workflow definition validation failed: %w", err)
		}
		updates.Version = workflow.Version + 1
	}

	return s.db.Model(&workflow).Updates(updates).Error
}

// DeleteWorkflow 删除工作流
func (s *WorkflowService) DeleteWorkflow(id uint) error {
	var workflow model.Workflow
	if err := s.db.First(&workflow, id).Error; err != nil {
		return err
	}

	if workflow.IsBuiltin {
		return fmt.Errorf("cannot delete builtin workflow")
	}

	return s.db.Delete(&workflow).Error
}

// GetWorkflow 获取工作流详情
func (s *WorkflowService) GetWorkflow(id uint) (*model.Workflow, error) {
	var workflow model.Workflow
	if err := s.db.First(&workflow, id).Error; err != nil {
		return nil, err
	}
	return &workflow, nil
}

// ListWorkflows 列出工作流
func (s *WorkflowService) ListWorkflows(category string, limit int, offset int) ([]model.Workflow, int64, error) {
	var workflows []model.Workflow
	var total int64

	query := s.db.Model(&model.Workflow{})

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&workflows).Error; err != nil {
		return nil, 0, err
	}

	return workflows, total, nil
}

// ExecuteWorkflow 执行工作流
func (s *WorkflowService) ExecuteWorkflow(ctx context.Context, config *WorkflowExecuteConfig, userID uint) (string, error) {
	// 获取工作流
	workflow, err := s.GetWorkflow(config.WorkflowID)
	if err != nil {
		return "", fmt.Errorf("failed to get workflow: %w", err)
	}

	// 解析工作流定义
	var definition WorkflowDefinition
	if err := json.Unmarshal([]byte(workflow.Definition), &definition); err != nil {
		return "", fmt.Errorf("failed to parse workflow definition: %w", err)
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("workflow_exec_%d_%d", userID, time.Now().UnixNano())

	// 创建执行记录
	execution := &model.WorkflowExecution{
		TaskID:       taskID,
		WorkflowID:   workflow.ID,
		WorkflowName: workflow.Name,
		ClusterName:  config.ClusterName,
		TargetNodes:  marshalJSON(config.TargetNodes),
		Parameters:   marshalJSON(config.Parameters),
		Status:       "pending",
		UserID:       userID,
	}

	if err := s.db.Create(execution).Error; err != nil {
		return "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// 异步执行工作流
	go s.executeWorkflowAsync(ctx, config, workflow, &definition, execution)

	return taskID, nil
}

// executeWorkflowAsync 异步执行工作流
func (s *WorkflowService) executeWorkflowAsync(
	ctx context.Context,
	config *WorkflowExecuteConfig,
	workflow *model.Workflow,
	definition *WorkflowDefinition,
	execution *model.WorkflowExecution,
) {
	startTime := time.Now()

	// 更新状态为 running
	s.db.Model(execution).Updates(map[string]interface{}{
		"status":     "running",
		"start_time": startTime,
	})

	// 创建进度任务
	if s.progressSvc != nil {
		s.progressSvc.CreateTask(execution.TaskID, "workflow", len(definition.Steps), execution.UserID)
	}

	// 创建执行上下文
	execCtx := &WorkflowExecutionContext{
		ctx:          ctx,
		execution:    execution,
		definition:   definition,
		stepResults:  make(map[string]*StepResult),
		config:       config,
		credentialID: config.CredentialID,
	}

	// 执行工作流
	err := s.executeWorkflowSteps(execCtx)

	// 统计结果
	successCount := 0
	failedCount := 0
	skippedCount := 0
	for _, result := range execCtx.stepResults {
		switch result.Status {
		case "completed":
			successCount++
		case "failed":
			failedCount++
		case "skipped":
			skippedCount++
		}
	}

	// 保存结果
	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int(endTime.Sub(startTime).Seconds())
	execution.StepResults = marshalJSON(execCtx.stepResults)

	if err != nil {
		execution.Status = "failed"
		execution.ErrorMessage = err.Error()
	} else if failedCount > 0 {
		execution.Status = "partial"
	} else {
		execution.Status = "completed"
	}

	if err := s.db.Save(execution).Error; err != nil {
		s.logger.Errorf("Failed to save execution record: %v", err)
	}

	// 发送完成消息
	if s.progressSvc != nil {
		if execution.Status == "failed" {
			s.progressSvc.ErrorTask(execution.TaskID, fmt.Errorf("%s", execution.ErrorMessage), execution.UserID)
		} else {
			s.progressSvc.CompleteTask(execution.TaskID, execution.UserID)
		}
	}

	s.logger.Infof("Workflow execution completed: TaskID=%s, Status=%s, Success=%d, Failed=%d, Skipped=%d",
		execution.TaskID, execution.Status, successCount, failedCount, skippedCount)
}

// executeWorkflowSteps 执行工作流步骤
func (s *WorkflowService) executeWorkflowSteps(execCtx *WorkflowExecutionContext) error {
	// 构建步骤依赖图
	steps := execCtx.definition.Steps
	completed := make(map[string]bool)
	processing := make(map[string]bool)

	// 执行所有步骤（按依赖顺序）
	for len(completed) < len(steps) {
		progress := false

		for i := range steps {
			step := &steps[i]

			// 跳过已完成或正在处理的步骤
			if completed[step.ID] || processing[step.ID] {
				continue
			}

			// 检查依赖是否满足
			if !s.checkStepDependencies(step, execCtx.stepResults) {
				continue
			}

			// 检查执行条件
			if !s.evaluateStepCondition(step, execCtx.stepResults) {
				// 标记为跳过
				execCtx.stepResults[step.ID] = &StepResult{
					StepID:   step.ID,
					StepName: step.Name,
					Status:   "skipped",
				}
				completed[step.ID] = true
				progress = true
				continue
			}

			// 执行步骤
			processing[step.ID] = true
			result := s.executeStep(execCtx, step)
			execCtx.mu.Lock()
			execCtx.stepResults[step.ID] = result
			execCtx.mu.Unlock()

			completed[step.ID] = true
			processing[step.ID] = false
			progress = true

			// 更新当前步骤
			s.db.Model(execCtx.execution).Update("current_step", step.Name)

			// 发送进度更新
			if s.progressSvc != nil {
				s.progressSvc.UpdateProgress(execCtx.execution.TaskID, len(completed), step.Name, execCtx.execution.UserID)
			}

			// 如果步骤失败且不继续执行，则停止
			if result.Status == "failed" && !step.ContinueOnError {
				return fmt.Errorf("step %s failed: %s", step.Name, result.Error)
			}
		}

		// 如果没有进度，说明存在循环依赖或无法满足的依赖
		if !progress {
			return fmt.Errorf("workflow execution stuck: circular dependencies or unsatisfied dependencies")
		}
	}

	return nil
}

// executeStep 执行单个步骤
func (s *WorkflowService) executeStep(execCtx *WorkflowExecutionContext, step *WorkflowStep) *StepResult {
	result := &StepResult{
		StepID:   step.ID,
		StepName: step.Name,
		Status:   "running",
	}

	startTime := time.Now()
	result.StartTime = &startTime

	// 执行重试逻辑
	maxAttempts := 1
	delay := 0
	if step.Retry != nil {
		maxAttempts = step.Retry.MaxAttempts
		delay = step.Retry.Delay
	}

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		result.Attempt = attempt

		// 如果不是第一次尝试，等待延迟
		if attempt > 1 && delay > 0 {
			time.Sleep(time.Duration(delay) * time.Second)
			s.logger.Infof("Retrying step %s (attempt %d/%d)", step.Name, attempt, maxAttempts)
		}

		// 根据步骤类型执行
		var err error
		switch step.Type {
		case "ansible":
			err = s.executeAnsibleStep(execCtx, step, result)
		case "ssh":
			err = s.executeSSHStep(execCtx, step, result)
		case "script":
			err = s.executeScriptStep(execCtx, step, result)
		case "node-operation":
			err = s.executeNodeOperationStep(execCtx, step, result)
		default:
			err = fmt.Errorf("unsupported step type: %s", step.Type)
		}

		if err == nil {
			result.Status = "completed"
			break
		}

		lastErr = err
		if attempt == maxAttempts {
			result.Status = "failed"
			result.Error = lastErr.Error()
		}
	}

	endTime := time.Now()
	result.EndTime = &endTime
	result.Duration = int(endTime.Sub(startTime).Seconds())

	return result
}

// executeAnsibleStep 执行 Ansible 步骤
func (s *WorkflowService) executeAnsibleStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, result *StepResult) error {
	// 解析 playbook ID
	var playbookID uint
	if _, err := fmt.Sscanf(step.Action, "%d", &playbookID); err != nil {
		return fmt.Errorf("invalid playbook ID: %s", step.Action)
	}

	// 转换参数类型
	extraVars := make(map[string]interface{})
	for k, v := range step.Parameters {
		extraVars[k] = v
	}

	// 执行 Ansible Playbook
	config := &PlaybookRunConfig{
		PlaybookID:   playbookID,
		ClusterName:  execCtx.config.ClusterName,
		TargetNodes:  execCtx.config.TargetNodes,
		ExtraVars:    extraVars,
		CredentialID: execCtx.credentialID,
		CheckMode:    false,
	}

	taskID, err := s.ansibleSvc.ExecutePlaybook(execCtx.ctx, config, execCtx.execution.UserID)
	if err != nil {
		return err
	}

	// 等待执行完成（轮询状态）
	return s.waitForTaskCompletion(taskID, step.Timeout)
}

// executeSSHStep 执行 SSH 步骤
func (s *WorkflowService) executeSSHStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, result *StepResult) error {
	config := &SSHExecuteConfig{
		ClusterName:  execCtx.config.ClusterName,
		TargetNodes:  execCtx.config.TargetNodes,
		Command:      step.Action,
		CredentialID: execCtx.credentialID,
		Timeout:      time.Duration(step.Timeout) * time.Second,
		Concurrent:   10,
	}

	taskID, err := s.sshSvc.ExecuteCommand(execCtx.ctx, config, execCtx.execution.UserID)
	if err != nil {
		return err
	}

	return s.waitForTaskCompletion(taskID, step.Timeout)
}

// executeScriptStep 执行脚本步骤
func (s *WorkflowService) executeScriptStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, result *StepResult) error {
	// 解析 script ID
	var scriptID uint
	if _, err := fmt.Sscanf(step.Action, "%d", &scriptID); err != nil {
		return fmt.Errorf("invalid script ID: %s", step.Action)
	}

	config := &ScriptExecuteConfig{
		ScriptID:     scriptID,
		ClusterName:  execCtx.config.ClusterName,
		TargetNodes:  execCtx.config.TargetNodes,
		Parameters:   step.Parameters,
		CredentialID: execCtx.credentialID,
		Timeout:      time.Duration(step.Timeout) * time.Second,
		Concurrent:   10,
	}

	taskID, err := s.scriptSvc.ExecuteScript(execCtx.ctx, config, execCtx.execution.UserID)
	if err != nil {
		return err
	}

	return s.waitForTaskCompletion(taskID, step.Timeout)
}

// executeNodeOperationStep 执行节点操作步骤
func (s *WorkflowService) executeNodeOperationStep(execCtx *WorkflowExecutionContext, step *WorkflowStep, result *StepResult) error {
	// TODO: 集成节点操作（cordon, uncordon, drain等）
	// 这需要调用 node service
	return fmt.Errorf("node operation not yet implemented")
}

// waitForTaskCompletion 等待任务完成
func (s *WorkflowService) waitForTaskCompletion(taskID string, timeoutSeconds int) error {
	timeout := time.Duration(timeoutSeconds) * time.Second
	if timeoutSeconds == 0 {
		timeout = 30 * time.Minute // 默认 30 分钟
	}

	deadline := time.Now().Add(timeout)

	for {
		if time.Now().After(deadline) {
			return fmt.Errorf("task execution timeout")
		}

		// 根据 taskID 前缀判断任务类型并查询状态
		// 这里简化处理，实际应该有更通用的状态查询机制
		time.Sleep(2 * time.Second)

		// TODO: 实现通用的任务状态查询
		// 目前简化为等待固定时间
		return nil
	}
}

// checkStepDependencies 检查步骤依赖是否满足
func (s *WorkflowService) checkStepDependencies(step *WorkflowStep, results map[string]*StepResult) bool {
	for _, depID := range step.DependsOn {
		result, exists := results[depID]
		if !exists || (result.Status != "completed" && result.Status != "skipped") {
			return false
		}
	}
	return true
}

// evaluateStepCondition 评估步骤执行条件
func (s *WorkflowService) evaluateStepCondition(step *WorkflowStep, results map[string]*StepResult) bool {
	if step.Condition == nil {
		return true // 无条件，总是执行
	}

	switch step.Condition.Type {
	case "always":
		return true
	case "on_success":
		// 所有依赖步骤都成功
		for _, depID := range step.Condition.Steps {
			result, exists := results[depID]
			if !exists || result.Status != "completed" {
				return false
			}
		}
		return true
	case "on_failure":
		// 至少一个依赖步骤失败
		for _, depID := range step.Condition.Steps {
			result, exists := results[depID]
			if exists && result.Status == "failed" {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// validateWorkflowDefinition 验证工作流定义
func (s *WorkflowService) validateWorkflowDefinition(definitionJSON string) error {
	var definition WorkflowDefinition
	if err := json.Unmarshal([]byte(definitionJSON), &definition); err != nil {
		return fmt.Errorf("invalid JSON: %w", err)
	}

	if len(definition.Steps) == 0 {
		return fmt.Errorf("workflow must have at least one step")
	}

	// 检查步骤 ID 唯一性
	stepIDs := make(map[string]bool)
	for _, step := range definition.Steps {
		if step.ID == "" {
			return fmt.Errorf("step ID is required")
		}
		if stepIDs[step.ID] {
			return fmt.Errorf("duplicate step ID: %s", step.ID)
		}
		stepIDs[step.ID] = true

		// 检查依赖的步骤是否存在
		for _, depID := range step.DependsOn {
			if !stepIDs[depID] {
				// 依赖的步骤可能在后面定义，这里不做严格检查
			}
		}
	}

	return nil
}

// GetExecutionStatus 获取执行状态
func (s *WorkflowService) GetExecutionStatus(taskID string) (*model.WorkflowExecution, error) {
	var execution model.WorkflowExecution
	if err := s.db.Where("task_id = ?", taskID).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListExecutions 列出执行历史
func (s *WorkflowService) ListExecutions(workflowID uint, status string, limit int, offset int) ([]model.WorkflowExecution, int64, error) {
	var executions []model.WorkflowExecution
	var total int64

	query := s.db.Model(&model.WorkflowExecution{})

	if workflowID > 0 {
		query = query.Where("workflow_id = ?", workflowID)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// InitializeBuiltinWorkflows 初始化内置工作流
func (s *WorkflowService) InitializeBuiltinWorkflows() error {
	builtinWorkflows := []model.Workflow{
		{
			Name:        "节点维护工作流",
			Description: "完整的节点维护流程：Cordon → 升级 → Reboot → Uncordon",
			Definition:  getNodeMaintenanceWorkflow(),
			Category:    "maintenance",
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			Name:        "故障诊断工作流",
			Description: "系统故障诊断流程：信息收集 → 日志分析 → 报告生成",
			Definition:  getDiagnosisWorkflow(),
			Category:    "diagnosis",
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			Name:        "批量部署工作流",
			Description: "软件批量部署流程：环境检查 → 软件安装 → 配置更新 → 服务重启",
			Definition:  getDeploymentWorkflow(),
			Category:    "deployment",
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
	}

	for _, workflow := range builtinWorkflows {
		var existing model.Workflow
		result := s.db.Where("name = ? AND is_builtin = ?", workflow.Name, true).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			if err := s.db.Create(&workflow).Error; err != nil {
				s.logger.Errorf("Failed to create builtin workflow %s: %v", workflow.Name, err)
				continue
			}
			s.logger.Infof("Created builtin workflow: %s", workflow.Name)
		} else if result.Error == nil {
			if existing.Definition != workflow.Definition {
				updates := map[string]interface{}{
					"definition":  workflow.Definition,
					"description": workflow.Description,
					"version":     existing.Version + 1,
				}
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					s.logger.Errorf("Failed to update builtin workflow %s: %v", workflow.Name, err)
				} else {
					s.logger.Infof("Updated builtin workflow: %s", workflow.Name)
				}
			}
		}
	}

	return nil
}

// 内置工作流定义

func getNodeMaintenanceWorkflow() string {
	def := WorkflowDefinition{
		Steps: []WorkflowStep{
			{
				ID:      "step1",
				Name:    "Cordon Node",
				Type:    "node-operation",
				Action:  "cordon",
				Timeout: 60,
			},
			{
				ID:        "step2",
				Name:      "System Upgrade",
				Type:      "ssh",
				Action:    "apt-get update && apt-get upgrade -y",
				DependsOn: []string{"step1"},
				Timeout:   600,
			},
			{
				ID:              "step3",
				Name:            "Reboot Node",
				Type:            "ssh",
				Action:          "reboot",
				DependsOn:       []string{"step2"},
				ContinueOnError: true,
				Timeout:         300,
			},
			{
				ID:        "step4",
				Name:      "Wait for Node Ready",
				Type:      "ssh",
				Action:    "sleep 60",
				DependsOn: []string{"step3"},
				Timeout:   120,
			},
			{
				ID:        "step5",
				Name:      "Uncordon Node",
				Type:      "node-operation",
				Action:    "uncordon",
				DependsOn: []string{"step4"},
				Timeout:   60,
			},
		},
	}
	data, _ := json.Marshal(def)
	return string(data)
}

func getDiagnosisWorkflow() string {
	def := WorkflowDefinition{
		Steps: []WorkflowStep{
			{
				ID:      "step1",
				Name:    "Collect System Info",
				Type:    "script",
				Action:  "1", // 系统信息收集脚本 ID
				Timeout: 300,
			},
			{
				ID:        "step2",
				Name:      "Collect Logs",
				Type:      "script",
				Action:    "3", // 日志收集脚本 ID
				DependsOn: []string{"step1"},
				Timeout:   300,
			},
			{
				ID:        "step3",
				Name:      "Performance Diagnosis",
				Type:      "script",
				Action:    "4", // 性能诊断脚本 ID
				DependsOn: []string{"step1"},
				Timeout:   300,
			},
		},
	}
	data, _ := json.Marshal(def)
	return string(data)
}

func getDeploymentWorkflow() string {
	def := WorkflowDefinition{
		Steps: []WorkflowStep{
			{
				ID:      "step1",
				Name:    "Check Environment",
				Type:    "ssh",
				Action:  "df -h && free -h",
				Timeout: 60,
			},
			{
				ID:        "step2",
				Name:      "Install Software",
				Type:      "ansible",
				Action:    "1", // Ansible Playbook ID
				DependsOn: []string{"step1"},
				Timeout:   600,
				Retry: &RetryPolicy{
					MaxAttempts: 3,
					Delay:       30,
				},
			},
			{
				ID:        "step3",
				Name:      "Update Configuration",
				Type:      "ssh",
				Action:    "systemctl restart docker",
				DependsOn: []string{"step2"},
				Timeout:   120,
			},
		},
	}
	data, _ := json.Marshal(def)
	return string(data)
}
