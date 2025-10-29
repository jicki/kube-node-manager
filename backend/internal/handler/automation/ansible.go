package automation

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/automation"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// AnsibleHandler Ansible API 处理器
type AnsibleHandler struct {
	ansibleSvc *automation.AnsibleService
	logger     *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// PaginatedResponse 分页响应结构
type PaginatedResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

// NewAnsibleHandler 创建 Ansible 处理器
func NewAnsibleHandler(ansibleSvc *automation.AnsibleService, logger *logger.Logger) *AnsibleHandler {
	return &AnsibleHandler{
		ansibleSvc: ansibleSvc,
		logger:     logger,
	}
}

// CreatePlaybook 创建 Playbook
// @Summary 创建 Playbook
// @Tags Automation-Ansible
// @Accept json
// @Produce json
// @Param playbook body model.AnsiblePlaybook true "Playbook 信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/playbooks [post]
func (h *AnsibleHandler) CreatePlaybook(c *gin.Context) {
	var playbook model.AnsiblePlaybook
	if err := c.ShouldBindJSON(&playbook); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	playbook.CreatedBy = userID.(uint)
	playbook.UpdatedBy = userID.(uint)

	if err := h.ansibleSvc.CreatePlaybook(&playbook); err != nil {
		h.logger.Errorf("Failed to create playbook: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create playbook: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Playbook created successfully",
		Data:    playbook,
	})
}

// GetPlaybook 获取 Playbook
// @Summary 获取 Playbook
// @Tags Automation-Ansible
// @Produce json
// @Param id path int true "Playbook ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/playbooks/{id} [get]
func (h *AnsibleHandler) GetPlaybook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid playbook ID",
		})
		return
	}

	playbook, err := h.ansibleSvc.GetPlaybook(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get playbook: %v", err)
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "Playbook not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    playbook,
	})
}

// ListPlaybooks 列出 Playbook
// @Summary 列出 Playbook
// @Tags Automation-Ansible
// @Produce json
// @Param category query string false "分类"
// @Param is_active query bool false "是否启用"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/playbooks [get]
func (h *AnsibleHandler) ListPlaybooks(c *gin.Context) {
	category := c.Query("category")

	var isActive *bool
	if isActiveStr := c.Query("is_active"); isActiveStr != "" {
		active := isActiveStr == "true"
		isActive = &active
	}

	playbooks, err := h.ansibleSvc.ListPlaybooks(category, isActive)
	if err != nil {
		h.logger.Errorf("Failed to list playbooks: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list playbooks: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    playbooks,
	})
}

// UpdatePlaybook 更新 Playbook
// @Summary 更新 Playbook
// @Tags Automation-Ansible
// @Accept json
// @Produce json
// @Param id path int true "Playbook ID"
// @Param playbook body model.AnsiblePlaybook true "Playbook 更新信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/playbooks/{id} [put]
func (h *AnsibleHandler) UpdatePlaybook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid playbook ID",
		})
		return
	}

	var updates model.AnsiblePlaybook
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	updates.UpdatedBy = userID.(uint)

	if err := h.ansibleSvc.UpdatePlaybook(uint(id), &updates); err != nil {
		h.logger.Errorf("Failed to update playbook: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update playbook: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Playbook updated successfully",
	})
}

// DeletePlaybook 删除 Playbook
// @Summary 删除 Playbook
// @Tags Automation-Ansible
// @Produce json
// @Param id path int true "Playbook ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/playbooks/{id} [delete]
func (h *AnsibleHandler) DeletePlaybook(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid playbook ID",
		})
		return
	}

	if err := h.ansibleSvc.DeletePlaybook(uint(id)); err != nil {
		h.logger.Errorf("Failed to delete playbook: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete playbook: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Playbook deleted successfully",
	})
}

// ExecutePlaybook 执行 Playbook
// @Summary 执行 Playbook
// @Tags Automation-Ansible
// @Accept json
// @Produce json
// @Param request body automation.PlaybookRunConfig true "执行配置"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/run [post]
func (h *AnsibleHandler) ExecutePlaybook(c *gin.Context) {
	var config automation.PlaybookRunConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 验证必要参数
	if config.PlaybookID == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Playbook ID is required",
		})
		return
	}

	if len(config.TargetNodes) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Target nodes are required",
		})
		return
	}

	// 获取用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 执行 Playbook
	taskID, err := h.ansibleSvc.ExecutePlaybook(c.Request.Context(), &config, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to execute playbook: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to execute playbook: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Playbook execution started",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}

// GetExecutionStatus 获取执行状态
// @Summary 获取执行状态
// @Tags Automation-Ansible
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/status/{task_id} [get]
func (h *AnsibleHandler) GetExecutionStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Task ID is required",
		})
		return
	}

	execution, err := h.ansibleSvc.GetExecutionStatus(taskID)
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
// @Tags Automation-Ansible
// @Produce json
// @Param cluster_name query string false "集群名称"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/ansible/history [get]
func (h *AnsibleHandler) ListExecutions(c *gin.Context) {
	clusterName := c.Query("cluster_name")
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

	executions, total, err := h.ansibleSvc.ListExecutions(clusterName, status, size, offset)
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

// CancelExecution 取消执行
// @Summary 取消执行
// @Tags Automation-Ansible
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/ansible/cancel/{task_id} [post]
func (h *AnsibleHandler) CancelExecution(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Task ID is required",
		})
		return
	}

	if err := h.ansibleSvc.CancelExecution(taskID); err != nil {
		h.logger.Errorf("Failed to cancel execution: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to cancel execution: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Execution cancelled successfully",
	})
}
