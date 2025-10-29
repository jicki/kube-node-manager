package automation

import (
	"net/http"
	"strconv"
	"time"

	"kube-node-manager/internal/service/automation"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// SSHHandler SSH API 处理器
type SSHHandler struct {
	sshSvc *automation.SSHService
	logger *logger.Logger
}

// NewSSHHandler 创建 SSH 处理器
func NewSSHHandler(sshSvc *automation.SSHService, logger *logger.Logger) *SSHHandler {
	return &SSHHandler{
		sshSvc: sshSvc,
		logger: logger,
	}
}

// ExecuteCommand 执行 SSH 命令
// @Summary 执行 SSH 命令
// @Tags Automation-SSH
// @Accept json
// @Produce json
// @Param request body automation.SSHExecuteConfig true "执行配置"
// @Success 200 {object} Response
// @Router /api/v1/automation/ssh/execute [post]
func (h *SSHHandler) ExecuteCommand(c *gin.Context) {
	var req struct {
		ClusterName  string   `json:"cluster_name" binding:"required"`
		TargetNodes  []string `json:"target_nodes" binding:"required"`
		Command      string   `json:"command" binding:"required"`
		CredentialID uint     `json:"credential_id" binding:"required"`
		Timeout      int      `json:"timeout"`    // 秒，默认 30
		Concurrent   int      `json:"concurrent"` // 并发数，默认 10
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 参数校验
	if len(req.TargetNodes) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Target nodes are required",
		})
		return
	}

	if req.Command == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Command is required",
		})
		return
	}

	// 设置默认值
	if req.Timeout == 0 {
		req.Timeout = 30
	}
	if req.Concurrent == 0 {
		req.Concurrent = 10
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

	// 构建执行配置
	config := &automation.SSHExecuteConfig{
		ClusterName:  req.ClusterName,
		TargetNodes:  req.TargetNodes,
		Command:      req.Command,
		CredentialID: req.CredentialID,
		Timeout:      time.Duration(req.Timeout) * time.Second,
		Concurrent:   req.Concurrent,
	}

	// 执行命令
	taskID, err := h.sshSvc.ExecuteCommand(c.Request.Context(), config, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to execute SSH command: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to execute command: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Command execution started",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}

// GetExecutionStatus 获取执行状态
// @Summary 获取执行状态
// @Tags Automation-SSH
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/ssh/status/{task_id} [get]
func (h *SSHHandler) GetExecutionStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Task ID is required",
		})
		return
	}

	record, err := h.sshSvc.GetExecutionStatus(taskID)
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
		Data:    record,
	})
}

// ListExecutions 列出执行历史
// @Summary 列出执行历史
// @Tags Automation-SSH
// @Produce json
// @Param cluster_name query string false "集群名称"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/ssh/history [get]
func (h *SSHHandler) ListExecutions(c *gin.Context) {
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

	records, total, err := h.sshSvc.ListExecutions(clusterName, status, size, offset)
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
		Data:    records,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}
