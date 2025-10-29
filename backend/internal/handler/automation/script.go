package automation

import (
	"net/http"
	"strconv"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/automation"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// ScriptHandler 脚本 API 处理器
type ScriptHandler struct {
	scriptSvc *automation.ScriptService
	logger    *logger.Logger
}

// NewScriptHandler 创建脚本处理器
func NewScriptHandler(scriptSvc *automation.ScriptService, logger *logger.Logger) *ScriptHandler {
	return &ScriptHandler{
		scriptSvc: scriptSvc,
		logger:    logger,
	}
}

// CreateScript 创建脚本
// @Summary 创建脚本
// @Tags Automation-Scripts
// @Accept json
// @Produce json
// @Param request body model.Script true "脚本信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts [post]
func (h *ScriptHandler) CreateScript(c *gin.Context) {
	var script model.Script
	if err := c.ShouldBindJSON(&script); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 获取用户 ID
	userID, _ := c.Get("user_id")
	script.CreatedBy = userID.(uint)

	if err := h.scriptSvc.CreateScript(&script); err != nil {
		h.logger.Errorf("Failed to create script: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create script: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Script created successfully",
		Data:    script,
	})
}

// UpdateScript 更新脚本
// @Summary 更新脚本
// @Tags Automation-Scripts
// @Accept json
// @Produce json
// @Param id path int true "脚本 ID"
// @Param request body model.Script true "脚本信息"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts/:id [put]
func (h *ScriptHandler) UpdateScript(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid script ID",
		})
		return
	}

	var updates model.Script
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	if err := h.scriptSvc.UpdateScript(uint(id), &updates); err != nil {
		h.logger.Errorf("Failed to update script: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update script: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Script updated successfully",
	})
}

// DeleteScript 删除脚本
// @Summary 删除脚本
// @Tags Automation-Scripts
// @Produce json
// @Param id path int true "脚本 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts/:id [delete]
func (h *ScriptHandler) DeleteScript(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid script ID",
		})
		return
	}

	if err := h.scriptSvc.DeleteScript(uint(id)); err != nil {
		h.logger.Errorf("Failed to delete script: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to delete script: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Script deleted successfully",
	})
}

// GetScript 获取脚本详情
// @Summary 获取脚本详情
// @Tags Automation-Scripts
// @Produce json
// @Param id path int true "脚本 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts/:id [get]
func (h *ScriptHandler) GetScript(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid script ID",
		})
		return
	}

	script, err := h.scriptSvc.GetScript(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get script: %v", err)
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "Script not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    script,
	})
}

// ListScripts 列出脚本
// @Summary 列出脚本
// @Tags Automation-Scripts
// @Produce json
// @Param type query string false "脚本类型"
// @Param category query string false "脚本分类"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/scripts [get]
func (h *ScriptHandler) ListScripts(c *gin.Context) {
	scriptType := c.Query("type")
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

	scripts, total, err := h.scriptSvc.ListScripts(scriptType, category, size, offset)
	if err != nil {
		h.logger.Errorf("Failed to list scripts: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list scripts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, PaginatedResponse{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    scripts,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}

// ExecuteScript 执行脚本
// @Summary 执行脚本
// @Tags Automation-Scripts
// @Accept json
// @Produce json
// @Param request body automation.ScriptExecuteConfig true "执行配置"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts/execute [post]
func (h *ScriptHandler) ExecuteScript(c *gin.Context) {
	var req struct {
		ScriptID     uint              `json:"script_id" binding:"required"`
		ClusterName  string            `json:"cluster_name" binding:"required"`
		TargetNodes  []string          `json:"target_nodes" binding:"required"`
		Parameters   map[string]string `json:"parameters"`
		CredentialID uint              `json:"credential_id" binding:"required"`
		Timeout      int               `json:"timeout"`    // 秒，默认 300
		Concurrent   int               `json:"concurrent"` // 并发数，默认 10
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 设置默认值
	if req.Timeout == 0 {
		req.Timeout = 300 // 5分钟
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
	config := &automation.ScriptExecuteConfig{
		ScriptID:     req.ScriptID,
		ClusterName:  req.ClusterName,
		TargetNodes:  req.TargetNodes,
		Parameters:   req.Parameters,
		CredentialID: req.CredentialID,
		Timeout:      time.Duration(req.Timeout) * time.Second,
		Concurrent:   req.Concurrent,
	}

	// 执行脚本
	taskID, err := h.scriptSvc.ExecuteScript(c.Request.Context(), config, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to execute script: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to execute script: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Script execution started",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}

// GetExecutionStatus 获取执行状态
// @Summary 获取执行状态
// @Tags Automation-Scripts
// @Produce json
// @Param task_id path string true "任务 ID"
// @Success 200 {object} Response
// @Router /api/v1/automation/scripts/status/:task_id [get]
func (h *ScriptHandler) GetExecutionStatus(c *gin.Context) {
	taskID := c.Param("task_id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Task ID is required",
		})
		return
	}

	execution, err := h.scriptSvc.GetExecutionStatus(taskID)
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
// @Tags Automation-Scripts
// @Produce json
// @Param script_id query int false "脚本 ID"
// @Param status query string false "状态"
// @Param page query int false "页码" default(1)
// @Param size query int false "每页数量" default(20)
// @Success 200 {object} PaginatedResponse
// @Router /api/v1/automation/scripts/history [get]
func (h *ScriptHandler) ListExecutions(c *gin.Context) {
	scriptID, _ := strconv.ParseUint(c.Query("script_id"), 10, 32)
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

	executions, total, err := h.scriptSvc.ListExecutions(uint(scriptID), status, size, offset)
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
