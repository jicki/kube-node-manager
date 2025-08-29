package audit

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 审计日志处理器
type Handler struct {
	auditSvc *audit.Service
	logger   *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的审计处理器实例
func NewHandler(auditSvc *audit.Service, logger *logger.Logger) *Handler {
	return &Handler{
		auditSvc: auditSvc,
		logger:   logger,
	}
}

// List 获取审计日志列表
// @Summary 获取审计日志列表
// @Description 获取审计日志列表，支持分页和筛选
// @Tags audit
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param user_id query int false "用户ID筛选"
// @Param cluster_id query int false "集群ID筛选"
// @Param action query string false "操作类型筛选"
// @Param resource_type query string false "资源类型筛选"
// @Param status query string false "状态筛选"
// @Param start_date query string false "开始日期筛选 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期筛选 (YYYY-MM-DD)"
// @Success 200 {object} Response
// @Router /audit/logs [get]
func (h *Handler) List(c *gin.Context) {
	var req audit.ListRequest

	// 解析查询参数
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil {
			req.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil {
			req.PageSize = pageSize
		}
	}
	if userIDStr := c.Query("user_id"); userIDStr != "" {
		if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
			req.UserID = uint(userID)
		}
	}
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			req.ClusterID = uint(clusterID)
		}
	}

	req.Action = model.AuditAction(c.Query("action"))
	req.ResourceType = model.ResourceType(c.Query("resource_type"))
	req.Status = model.AuditStatus(c.Query("status"))
	req.StartDate = c.Query("start_date")
	req.EndDate = c.Query("end_date")

	// 检查用户权限 - 普通用户只能查看自己的日志
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 检查用户角色
	userRole, roleExists := c.Get("user_role")
	if !roleExists {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "User role not found",
		})
		return
	}

	// 非管理员用户只能查看自己的审计日志
	if userRole.(string) != "admin" {
		req.UserID = currentUserID.(uint)
	}

	result, err := h.auditSvc.List(req)
	if err != nil {
		h.logger.Errorf("Failed to list audit logs: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list audit logs: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    result,
	})
}

// GetByID 获取审计日志详情
// @Summary 获取审计日志详情
// @Description 根据ID获取审计日志详细信息
// @Tags audit
// @Produce json
// @Param id path int true "审计日志ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /audit/logs/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid audit log ID",
		})
		return
	}

	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	userRole, roleExists := c.Get("user_role")
	if !roleExists {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "User role not found",
		})
		return
	}

	auditLog, err := h.auditSvc.GetByID(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get audit log: %v", err)
		if err.Error() == "record not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Audit log not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get audit log: " + err.Error(),
			})
		}
		return
	}

	// 检查权限 - 非管理员只能查看自己的日志
	if userRole.(string) != "admin" && auditLog.UserID != currentUserID.(uint) {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Permission denied",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    auditLog,
	})
}

// GetStats 获取审计统计信息
// @Summary 获取审计统计信息
// @Description 获取审计日志的统计信息，包括操作类型、状态等分布
// @Tags audit
// @Produce json
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} Response
// @Router /audit/stats [get]
func (h *Handler) GetStats(c *gin.Context) {
	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	userRole, roleExists := c.Get("user_role")
	if !roleExists || userRole.(string) != "admin" {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Admin access required",
		})
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	// 获取基础统计信息
	req := audit.ListRequest{
		Page:      1,
		PageSize:  10000, // 获取大量数据用于统计
		StartDate: startDate,
		EndDate:   endDate,
	}

	result, err := h.auditSvc.List(req)
	if err != nil {
		h.logger.Errorf("Failed to get audit stats: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get audit stats: " + err.Error(),
		})
		return
	}

	// 计算统计信息
	stats := h.calculateStats(result.Logs)

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    stats,
	})
}

// GetUserActivity 获取用户活动统计
// @Summary 获取用户活动统计
// @Description 获取指定用户的活动统计信息
// @Tags audit
// @Produce json
// @Param user_id query int false "用户ID (管理员可查看所有用户)"
// @Param days query int false "统计天数" default(7)
// @Success 200 {object} Response
// @Router /audit/user-activity [get]
func (h *Handler) GetUserActivity(c *gin.Context) {
	currentUserID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	userRole, roleExists := c.Get("user_role")
	if !roleExists {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "User role not found",
		})
		return
	}

	var targetUserID uint = currentUserID.(uint)

	// 管理员可以查看指定用户的活动
	if userRole.(string) == "admin" {
		if userIDStr := c.Query("user_id"); userIDStr != "" {
			if userID, err := strconv.ParseUint(userIDStr, 10, 32); err == nil {
				targetUserID = uint(userID)
			}
		}
	}

	days := 7
	if daysStr := c.Query("days"); daysStr != "" {
		if d, err := strconv.Atoi(daysStr); err == nil && d > 0 && d <= 90 {
			days = d
		}
	}

	// 获取用户活动数据
	req := audit.ListRequest{
		Page:     1,
		PageSize: 1000,
		UserID:   targetUserID,
	}

	result, err := h.auditSvc.List(req)
	if err != nil {
		h.logger.Errorf("Failed to get user activity: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get user activity: " + err.Error(),
		})
		return
	}

	// 计算用户活动统计
	activity := h.calculateUserActivity(result.Logs, days)

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    activity,
	})
}

// calculateStats 计算审计统计信息
func (h *Handler) calculateStats(logs []model.AuditLog) map[string]interface{} {
	stats := make(map[string]interface{})

	// 基础统计
	stats["total_logs"] = len(logs)

	// 按操作类型统计
	actionStats := make(map[string]int)
	resourceStats := make(map[string]int)
	statusStats := make(map[string]int)
	userStats := make(map[uint]int)

	successCount := 0
	failedCount := 0

	for _, log := range logs {
		// 操作类型统计
		actionStats[string(log.Action)]++

		// 资源类型统计
		resourceStats[string(log.ResourceType)]++

		// 状态统计
		statusStats[string(log.Status)]++

		// 用户统计
		userStats[log.UserID]++

		// 成功失败统计
		if log.Status == model.AuditStatusSuccess {
			successCount++
		} else {
			failedCount++
		}
	}

	stats["actions"] = actionStats
	stats["resources"] = resourceStats
	stats["status"] = statusStats
	stats["success_count"] = successCount
	stats["failed_count"] = failedCount
	stats["active_users"] = len(userStats)

	// 成功率
	if len(logs) > 0 {
		stats["success_rate"] = float64(successCount) / float64(len(logs)) * 100
	} else {
		stats["success_rate"] = 100.0
	}

	return stats
}

// calculateUserActivity 计算用户活动统计
func (h *Handler) calculateUserActivity(logs []model.AuditLog, days int) map[string]interface{} {
	activity := make(map[string]interface{})

	// 基础信息
	activity["total_actions"] = len(logs)
	activity["days"] = days

	// 按日期分组
	dailyActivity := make(map[string]int)
	actionTypes := make(map[string]int)
	resourceTypes := make(map[string]int)

	recentActions := 0
	for _, log := range logs {
		dateKey := log.CreatedAt.Format("2006-01-02")
		dailyActivity[dateKey]++

		actionTypes[string(log.Action)]++
		resourceTypes[string(log.ResourceType)]++

		// 统计最近7天的活动
		if log.CreatedAt.After(log.CreatedAt.AddDate(0, 0, -days)) {
			recentActions++
		}
	}

	activity["daily_activity"] = dailyActivity
	activity["action_types"] = actionTypes
	activity["resource_types"] = resourceTypes
	activity["recent_actions"] = recentActions

	// 平均每日活动
	if days > 0 {
		activity["avg_daily_actions"] = float64(recentActions) / float64(days)
	} else {
		activity["avg_daily_actions"] = 0.0
	}

	return activity
}
