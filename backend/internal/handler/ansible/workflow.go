package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// WorkflowHandler 工作流 Handler
type WorkflowHandler struct {
	workflowService  *ansible.WorkflowService
	workflowExecutor *ansible.WorkflowExecutor
	logger           *logger.Logger
}

// NewWorkflowHandler 创建工作流 Handler 实例
func NewWorkflowHandler(workflowService *ansible.WorkflowService, workflowExecutor *ansible.WorkflowExecutor, logger *logger.Logger) *WorkflowHandler {
	return &WorkflowHandler{
		workflowService:  workflowService,
		workflowExecutor: workflowExecutor,
		logger:           logger,
	}
}

// CreateWorkflow 创建工作流
// @Summary 创建工作流
// @Tags Ansible Workflow
// @Accept json
// @Produce json
// @Param workflow body model.WorkflowCreateRequest true "工作流配置"
// @Success 200 {object} model.AnsibleWorkflow
// @Router /api/ansible/workflows [post]
func (h *WorkflowHandler) CreateWorkflow(c *gin.Context) {
	var req model.WorkflowCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	// 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	workflow, err := h.workflowService.CreateWorkflow(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    workflow,
	})
}

// GetWorkflow 获取工作流详情
// @Summary 获取工作流详情
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "工作流 ID"
// @Success 200 {object} model.AnsibleWorkflow
// @Router /api/ansible/workflows/{id} [get]
func (h *WorkflowHandler) GetWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	workflow, err := h.workflowService.GetWorkflow(uint(id), userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    workflow,
	})
}

// UpdateWorkflow 更新工作流
// @Summary 更新工作流
// @Tags Ansible Workflow
// @Accept json
// @Produce json
// @Param id path int true "工作流 ID"
// @Param workflow body model.WorkflowUpdateRequest true "工作流配置"
// @Success 200 {object} model.AnsibleWorkflow
// @Router /api/ansible/workflows/{id} [put]
func (h *WorkflowHandler) UpdateWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	var req model.WorkflowUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	workflow, err := h.workflowService.UpdateWorkflow(uint(id), userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    workflow,
	})
}

// DeleteWorkflow 删除工作流
// @Summary 删除工作流
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "工作流 ID"
// @Success 200 {object} map[string]string
// @Router /api/ansible/workflows/{id} [delete]
func (h *WorkflowHandler) DeleteWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	if err := h.workflowService.DeleteWorkflow(uint(id), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "工作流删除成功",
	})
}

// ListWorkflows 查询工作流列表
// @Summary 查询工作流列表
// @Tags Ansible Workflow
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "搜索关键词"
// @Success 200 {object} map[string]interface{}
// @Router /api/ansible/workflows [get]
func (h *WorkflowHandler) ListWorkflows(c *gin.Context) {
	var req model.WorkflowListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	workflows, total, err := h.workflowService.ListWorkflows(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      200,
		"message":   "Success",
		"workflows": workflows,
		"total":     total,
		"page":      req.Page,
		"page_size": req.PageSize,
	})
}

// ExecuteWorkflow 执行工作流
// @Summary 执行工作流
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "工作流 ID"
// @Success 200 {object} model.AnsibleWorkflowExecution
// @Router /api/ansible/workflows/{id}/execute [post]
func (h *WorkflowHandler) ExecuteWorkflow(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	execution, err := h.workflowExecutor.ExecuteWorkflow(uint(id), userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    execution,
	})
}

// GetWorkflowExecution 获取工作流执行详情
// @Summary 获取工作流执行详情
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "执行 ID"
// @Success 200 {object} model.AnsibleWorkflowExecution
// @Router /api/ansible/workflow-executions/{id} [get]
func (h *WorkflowHandler) GetWorkflowExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	execution, err := h.workflowService.GetWorkflowExecution(uint(id), userID.(uint))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    execution,
	})
}

// ListWorkflowExecutions 查询工作流执行列表
// @Summary 查询工作流执行列表
// @Tags Ansible Workflow
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param workflow_id query int false "工作流 ID"
// @Param status query string false "执行状态"
// @Success 200 {object} map[string]interface{}
// @Router /api/ansible/workflow-executions [get]
func (h *WorkflowHandler) ListWorkflowExecutions(c *gin.Context) {
	var req model.WorkflowExecutionListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的请求参数: " + err.Error()})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	executions, total, err := h.workflowService.ListWorkflowExecutions(userID.(uint), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":       200,
		"message":    "Success",
		"executions": executions,
		"total":      total,
		"page":       req.Page,
		"page_size":  req.PageSize,
	})
}

// CancelWorkflowExecution 取消工作流执行
// @Summary 取消工作流执行
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "执行 ID"
// @Success 200 {object} map[string]string
// @Router /api/ansible/workflow-executions/{id}/cancel [post]
func (h *WorkflowHandler) CancelWorkflowExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	if err := h.workflowExecutor.CancelWorkflowExecution(uint(id), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "工作流执行已取消",
	})
}

// DeleteWorkflowExecution 删除工作流执行记录
// @Summary 删除工作流执行记录
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "执行 ID"
// @Success 200 {object} map[string]string
// @Router /api/ansible/workflow-executions/{id} [delete]
func (h *WorkflowHandler) DeleteWorkflowExecution(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	if err := h.workflowService.DeleteWorkflowExecution(uint(id), userID.(uint)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "执行记录已删除",
	})
}

// GetWorkflowExecutionStatus 获取工作流执行状态
// @Summary 获取工作流执行状态（实时节点状态）
// @Tags Ansible Workflow
// @Produce json
// @Param id path int true "执行 ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/ansible/workflow-executions/{id}/status [get]
func (h *WorkflowHandler) GetWorkflowExecutionStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 ID"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "未认证"})
		return
	}

	// 先尝试从运行中的工作流获取状态
	status := h.workflowExecutor.GetWorkflowExecutionStatus(uint(id))
	
	// 如果不在运行中，则从数据库中获取已完成的状态
	if status == nil {
		status, err = h.workflowService.GetCompletedWorkflowStatus(uint(id), userID.(uint))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":        200,
		"message":     "Success",
		"node_status": status,
	})
}

