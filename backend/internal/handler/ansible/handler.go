package ansible

import (
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
	// 检查用户权限 (只有管理员能访问 Ansible 模块)
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can access Ansible module"})
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
		"code":  0,
		"data":  tasks,
		"total": total,
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can access Ansible module"})
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
		"code": 0,
		"data": task,
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can create Ansible tasks"})
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
		"code":    0,
		"data":    task,
		"message": "Task created and started successfully",
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can cancel Ansible tasks"})
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
		"code":    0,
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can retry Ansible tasks"})
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
		"code":    0,
		"data":    task,
		"message": "Task retried successfully",
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can view Ansible task logs"})
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	logType := model.AnsibleLogType(c.Query("log_type"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "1000"))

	logs, err := h.service.GetTaskLogs(uint(id), logType, limit)
	if err != nil {
		h.logger.Errorf("Failed to get task logs: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": logs,
	})
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can refresh Ansible task status"})
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
		"code": 0,
		"data": status,
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
	// 检查用户权限
	userRole, exists := c.Get("user_role")
	if !exists || userRole.(string) != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Only administrators can view Ansible statistics"})
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
		"code": 0,
		"data": stats,
	})
}

