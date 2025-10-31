package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ScheduleHandler 定时任务 Handler
type ScheduleHandler struct {
	service *ansible.ScheduleService
	logger  *logger.Logger
}

// NewScheduleHandler 创建 Schedule Handler 实例
func NewScheduleHandler(service *ansible.ScheduleService, logger *logger.Logger) *ScheduleHandler {
	return &ScheduleHandler{
		service: service,
		logger:  logger,
	}
}

// ListSchedules 列出定时任务
// @Summary 列出定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param enabled query bool false "是否启用"
// @Param cluster_id query int false "集群ID"
// @Param keyword query string false "关键字"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules [get]
func (h *ScheduleHandler) ListSchedules(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.ScheduleListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedules, total, err := h.service.ListSchedules(req)
	if err != nil {
		h.logger.Errorf("Failed to list schedules: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    schedules,
		"total":   total,
	})
}

// GetSchedule 获取定时任务详情
// @Summary 获取定时任务详情
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param id path int true "定时任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules/{id} [get]
func (h *ScheduleHandler) GetSchedule(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	schedule, err := h.service.GetSchedule(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    schedule,
	})
}

// CreateSchedule 创建定时任务
// @Summary 创建定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param schedule body model.ScheduleCreateRequest true "定时任务信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules [post]
func (h *ScheduleHandler) CreateSchedule(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.ScheduleCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	schedule, err := h.service.CreateSchedule(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Schedule created successfully",
		"data":    schedule,
	})
}

// UpdateSchedule 更新定时任务
// @Summary 更新定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param id path int true "定时任务ID"
// @Param schedule body model.ScheduleUpdateRequest true "定时任务信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules/{id} [put]
func (h *ScheduleHandler) UpdateSchedule(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	var req model.ScheduleUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	schedule, err := h.service.UpdateSchedule(uint(id), req)
	if err != nil {
		h.logger.Errorf("Failed to update schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Schedule updated successfully",
		"data":    schedule,
	})
}

// DeleteSchedule 删除定时任务
// @Summary 删除定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param id path int true "定时任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules/{id} [delete]
func (h *ScheduleHandler) DeleteSchedule(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	if err := h.service.DeleteSchedule(uint(id)); err != nil {
		h.logger.Errorf("Failed to delete schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Schedule deleted successfully",
	})
}

// ToggleSchedule 启用/禁用定时任务
// @Summary 启用/禁用定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param id path int true "定时任务ID"
// @Param request body map[string]bool true "启用状态 {\"enabled\": true}"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules/{id}/toggle [post]
func (h *ScheduleHandler) ToggleSchedule(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ToggleSchedule(uint(id), req.Enabled); err != nil {
		h.logger.Errorf("Failed to toggle schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	status := "disabled"
	if req.Enabled {
		status = "enabled"
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Schedule " + status + " successfully",
	})
}

// RunNow 立即执行定时任务
// @Summary 立即执行定时任务
// @Tags Ansible Schedules
// @Accept json
// @Produce json
// @Param id path int true "定时任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/schedules/{id}/run-now [post]
func (h *ScheduleHandler) RunNow(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid schedule id"})
		return
	}

	if err := h.service.RunNow(uint(id)); err != nil {
		h.logger.Errorf("Failed to run schedule: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Schedule triggered successfully",
	})
}

