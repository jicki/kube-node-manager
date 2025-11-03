package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// Handler Ansible 任务 Handler
type Handler struct {
	service *ansible.Service
	logger  *logger.Logger
}

// NewHandler 创建 Handler 实例
func NewHandler(service *ansible.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// checkAdminPermission 检查管理员权限
func checkAdminPermission(c *gin.Context) bool {
	userRole, exists := c.Get("user_role")
	if !exists || userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can access Ansible module"})
		return false
	}
	return true
}

// ListTasks 列出任务
// @Summary 列出任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param status query string false "任务状态"
// @Param cluster_id query int false "集群ID"
// @Param template_id query int false "模板ID"
// @Param keyword query string false "关键字"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks [get]
func (h *Handler) ListTasks(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.TaskListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	tasks, total, err := h.service.ListTasks(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to list tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    tasks,
		"total":   total,
	})
}

// GetTask 获取任务详情
// @Summary 获取任务详情
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id} [get]
func (h *Handler) GetTask(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	task, err := h.service.GetTask(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    task,
	})
}

// CreateTask 创建并执行任务
// @Summary 创建并执行任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param task body model.TaskCreateRequest true "任务信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks [post]
func (h *Handler) CreateTask(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.TaskCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	task, err := h.service.CreateTask(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Task created and started successfully",
		"data":    task,
	})
}

// CancelTask 取消任务
// @Summary 取消任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id}/cancel [post]
func (h *Handler) CancelTask(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	if err := h.service.CancelTask(uint(id), userID.(uint)); err != nil {
		h.logger.Errorf("Failed to cancel task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Task cancelled successfully",
	})
}

// RetryTask 重试失败的任务
// @Summary 重试失败的任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id}/retry [post]
func (h *Handler) RetryTask(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	task, err := h.service.RetryTask(uint(id), userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to retry task: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Task retried successfully",
		"data":    task,
	})
}

// PauseBatch 暂停批次执行
// @Summary 暂停批次执行
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200
// @Router /api/v1/ansible/tasks/{id}/pause-batch [post]
func (h *Handler) PauseBatch(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.PauseBatchExecution(uint(id)); err != nil {
		h.logger.Errorf("Failed to pause batch: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Batch execution paused",
	})
}

// ContinueBatch 继续批次执行
// @Summary 继续批次执行
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200
// @Router /api/v1/ansible/tasks/{id}/continue-batch [post]
func (h *Handler) ContinueBatch(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.ContinueBatchExecution(uint(id)); err != nil {
		h.logger.Errorf("Failed to continue batch: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Batch execution continued",
	})
}

// StopBatch 停止批次执行
// @Summary 停止批次执行（停止所有剩余批次）
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200
// @Router /api/v1/ansible/tasks/{id}/stop-batch [post]
func (h *Handler) StopBatch(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	if err := h.service.StopBatchExecution(uint(id)); err != nil {
		h.logger.Errorf("Failed to stop batch: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Batch execution stopped",
	})
}

// RunPreflightChecks 执行前置检查
// @Summary 执行前置检查
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200 {object} model.PreflightCheckResult
// @Router /api/v1/ansible/tasks/{id}/preflight-checks [post]
func (h *Handler) RunPreflightChecks(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	preflightSvc := h.service.GetPreflightService()
	result, err := preflightSvc.RunPreflightChecks(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to run preflight checks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": result,
	})
}

// GetPreflightChecks 获取前置检查结果
// @Summary 获取前置检查结果
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200 {object} model.PreflightCheckResult
// @Router /api/v1/ansible/tasks/{id}/preflight-checks [get]
func (h *Handler) GetPreflightChecks(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	preflightSvc := h.service.GetPreflightService()
	result, err := preflightSvc.GetPreflightChecks(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "前置检查结果不存在"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": result,
	})
}

// GetTaskLogs 获取任务日志
// @Summary 获取任务日志
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Param log_type query string false "日志类型"
// @Param limit query int false "限制数量"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id}/logs [get]
func (h *Handler) GetTaskLogs(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 默认获取完整日志
	// 如果指定了 full=false，则只获取重要日志（保留向后兼容）
	useFull := c.DefaultQuery("full", "true") == "true"

	if useFull {
		// 获取完整日志
		fullLog, err := h.service.GetTaskFullLog(uint(id))
		if err != nil {
			h.logger.Errorf("Failed to get task full log: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "Success",
			"data":    fullLog,
		})
	} else {
		// 获取重要日志（旧方式）
		logType := model.AnsibleLogType(c.Query("log_type"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))

		logs, err := h.service.GetTaskLogs(uint(id), logType, limit)
		if err != nil {
			h.logger.Errorf("Failed to get task logs: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "Success",
			"data":    logs,
		})
	}
}

// RefreshTaskStatus 刷新任务状态
// @Summary 刷新任务状态
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id}/refresh [post]
func (h *Handler) RefreshTaskStatus(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	status, err := h.service.GetTaskStatus(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to refresh task status: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    status,
	})
}

// GetStatistics 获取统计信息
// @Summary 获取统计信息
// @Tags Ansible
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/statistics [get]
func (h *Handler) GetStatistics(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	stats, err := h.service.GetStatistics(userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to get statistics: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    stats,
	})
}

// DeleteTask 删除任务
// @Summary 删除任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id} [delete]
func (h *Handler) DeleteTask(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	if err := h.service.DeleteTask(uint(id), userID.(uint), username.(string)); err != nil {
		h.logger.Errorf("Failed to delete task %d: %v", id, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录审计日志
	h.logger.InfoWithFields("Deleted Ansible task", map[string]interface{}{
		"task_id":  id,
		"user_id":  userID,
		"username": username,
		"action":   "delete_ansible_task",
	})

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Task deleted successfully",
	})
}

// DeleteTasks 批量删除任务
// @Summary 批量删除任务
// @Tags Ansible
// @Accept json
// @Produce json
// @Param ids body []uint true "任务ID列表"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/batch-delete [post]
func (h *Handler) DeleteTasks(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req struct {
		IDs []uint `json:"ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if len(req.IDs) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "no task ids provided"})
		return
	}

	// 获取用户信息
	userID, _ := c.Get("user_id")
	username, _ := c.Get("username")

	successCount, errors, err := h.service.DeleteTasks(req.IDs, userID.(uint), username.(string))
	if err != nil {
		h.logger.Errorf("Failed to batch delete tasks: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 记录审计日志
	h.logger.InfoWithFields("Batch deleted Ansible tasks", map[string]interface{}{
		"task_ids":      req.IDs,
		"success_count": successCount,
		"failed_count":  len(errors),
		"user_id":       userID,
		"username":      username,
		"action":        "batch_delete_ansible_tasks",
	})

	response := gin.H{
		"code":          200,
		"message":       fmt.Sprintf("Successfully deleted %d tasks", successCount),
		"success_count": successCount,
	}

	if len(errors) > 0 {
		response["errors"] = errors
		response["failed_count"] = len(errors)
	}

	c.JSON(http.StatusOK, response)
}

