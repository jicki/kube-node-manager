package automation

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/automation"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流 API 处理器
type WorkflowHandler struct {
	workflowSvc *automation.WorkflowService
	logger      *logger.Logger
}

// NewWorkflowHandler 创建工作流处理器
func NewWorkflowHandler(workflowSvc *automation.WorkflowService, logger *logger.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		workflowSvc: workflowSvc,
		logger:      logger,
	}
}

// CreateWorkflow 创建工作流
// @Summary 创建工作流
// @Tags Automation-Workflows
// @Accept json
// @Produce json
// @Param request body model.Workflow true "工作流信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows [post]
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var workflow model.Workflow
	if err := c.ShouldBindJSON(&workflow); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	userID, _ := c.Get("user_id")
	workflow.CreatedBy = userID.(uint)
	workflow.UpdatedBy = userID.(uint)

	if err := h.workflowSvc.CreateWorkflow(&workflow); err != nil {
		h.logger.Errorf("Failed to create workflow: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create workflow: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Workflow created successfully",
		Data:    workflow,
	})
}

// UpdateWorkflow 更新工作流
// @Summary 更新工作流
// @Tags Automation-Workflows
// @Accept json
// @Produce json
// @Param id path int true "工作流 ID"
// @Param request body model.Workflow true "工作流信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows/:id [put]
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid workflow ID",
		})
		return
	}

	var updates model.Workflow
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 获取用户 ID
	userID, _ := c.Get("user_id")
	updates.UpdatedBy = userID.(uint)

	if err := h.workflowSvc.UpdateWorkflow(uint(id), &updates); err != nil {
		h.logger.Errorf("Failed to update workflow: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update workflow: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Workflow updated successfully",
	})
}

// DeleteWorkflow 删除工作流
// @Summary 删除工作流
// @Tags Automation-Workflows
// @Produce json
// @Param id path int true "工作流 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows/:id [delete]
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid workflow ID",
		})
		return
	}

	if err := h.workflowSvc.DeleteWorkflow(uint(id)); err != nil {
		h.logger.Errorf("Failed to delete workflow: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete workflow: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Workflow deleted successfully",
	})
}

// GetWorkflow 获取工作流详情
// @Summary 获取工作流详情
// @Tags Automation-Workflows
// @Produce json
// @Param id path int true "工作流 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows/:id [get]
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid workflow ID",
		})
		return
	}

	workflow, err := h.workflowSvc.GetWorkflow(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get workflow: %v", err)
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "Workflow not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    workflow,
	})
}

// ListWorkflows 列出工作流
// @Summary 列出工作流
// @Tags Automation-Workflows
// @Produce json
// @Param category query string false "工作流分类"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/workflows [get]
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	category := c.Query("category")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	workflows, total, err := h.workflowSvc.ListWorkflows(category, size, offset)
	if err != nil {
		h.logger.Errorf("Failed to list workflows: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list workflows: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    workflows,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// ExecuteWorkflow 执行工作流
// @Summary 执行工作流
// @Tags Automation-Workflows
// @Accept json
// @Produce json
// @Param request body automation.WorkflowExecuteConfig true "执行配置"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows/execute [post]
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	var req struct {
		WorkflowID   uint              `json:"workflow_id" binding:"required"`
		ClusterName  string            `json:"cluster_name" binding:"required"`
		TargetNodes  []string          `json:"target_nodes" binding:"required"`
		Parameters   map[string]string `json:"parameters"`
		CredentialID uint              `json:"credential_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	config := &automation.WorkflowExecuteConfig{
		WorkflowID:   req.WorkflowID,
		ClusterName:  req.ClusterName,
		TargetNodes:  req.TargetNodes,
		Parameters:   req.Parameters,
		CredentialID: req.CredentialID,
	}

	taskID, err := h.workflowSvc.ExecuteWorkflow(c.Request.Context(), config, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to execute workflow: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to execute workflow: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Workflow execution started",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}

// GetExecutionStatus 获取执行状态
// @Summary 获取执行状态
// @Tags Automation-Workflows
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/workflows/status/:task_id [get]
func (h *WorkflowHandler) GetExecutionStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Task ID is required",
		})
		return
	}

	execution, err := h.workflowSvc.GetExecutionStatus(taskID)
	if err != nil {
		h.logger.Errorf("Failed to get execution status: %v", err)
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "Execution not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    execution,
	})
}

// ListExecutions 列出执行历史
// @Summary 列出执行历史
// @Tags Automation-Workflows
// @Produce json
// @Param workflow_id query int false "工作流 ID"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/workflows/history [get]
func (h *WorkflowHandler) ListExecutions(c *gin.Context) {
	workflowID, _ := strconv.ParseUint(c.Query("workflow_id"), 10, 32)
	status := c.Query("status")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	size, _ := strconv.Atoi(c.DefaultQuery("size", "20"))

	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}

	offset := (page - 1) * size

	executions, total, err := h.workflowSvc.ListExecutions(uint(workflowID), status, size, offset)
	if err != nil {
		h.logger.Errorf("Failed to list executions: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list executions: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    executions,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}
