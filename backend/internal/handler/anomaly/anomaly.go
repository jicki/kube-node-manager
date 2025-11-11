package anomaly

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/anomaly"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 异常记录处理器
type Handler struct {
	anomalySvc *anomaly.Service
	cleanupSvc *anomaly.CleanupService
	logger     *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的异常记录处理器实例
func NewHandler(anomalySvc *anomaly.Service, cleanupSvc *anomaly.CleanupService, logger *logger.Logger) *Handler {
	return &Handler{
		anomalySvc: anomalySvc,
		cleanupSvc: cleanupSvc,
		logger:     logger,
	}
}

// GetByID 根据ID获取单个异常记录
// @Summary 根据ID获取异常记录详情
// @Description 根据ID获取单个异常记录的详细信息
// @Tags anomalies
// @Produce json
// @Param id path int true "异常记录ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	// 解析ID参数
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid anomaly ID: " + err.Error(),
		})
		return
	}

	// 调用服务获取异常记录
	anomaly, err := h.anomalySvc.GetByID(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get anomaly by ID %d: %v", id, err)
		// 判断是否是记录不存在的错误
		if strings.Contains(err.Error(), "anomaly not found") {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Anomaly not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get anomaly: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    anomaly,
	})
}

// List 获取异常记录列表
// @Summary 获取异常记录列表
// @Description 获取节点异常记录列表，支持多条件过滤和分页
// @Tags anomalies
// @Produce json
// @Param cluster_id query int false "集群ID"
// @Param node_name query string false "节点名称"
// @Param anomaly_type query string false "异常类型"
// @Param status query string false "异常状态"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies [get]
func (h *Handler) List(c *gin.Context) {
	req := anomaly.ListRequest{
		NodeName: c.Query("node_name"),
		Page:     1,
		PageSize: 20,
	}

	// 解析集群ID
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			id := uint(clusterID)
			req.ClusterID = &id
		}
	}

	// 解析异常类型
	if anomalyType := c.Query("anomaly_type"); anomalyType != "" {
		req.AnomalyType = model.AnomalyType(anomalyType)
	}

	// 解析异常状态
	if status := c.Query("status"); status != "" {
		req.Status = model.AnomalyStatus(status)
	}

	// 解析时间范围
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	// 解析分页参数
	if pageStr := c.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		}
	}
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 {
			req.PageSize = pageSize
		}
	}

	result, err := h.anomalySvc.GetAnomalies(req)
	if err != nil {
		h.logger.Errorf("Failed to get anomalies: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get anomalies: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    result,
	})
}

// GetStatistics 获取统计数据
// @Summary 获取异常统计数据
// @Description 获取按时间维度聚合的异常统计数据
// @Tags anomalies
// @Produce json
// @Param cluster_id query int false "集群ID"
// @Param anomaly_type query string false "异常类型"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Param dimension query string false "统计维度 (day/week)" default(day)
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies/statistics [get]
func (h *Handler) GetStatistics(c *gin.Context) {
	req := anomaly.StatisticsRequest{
		Dimension: c.DefaultQuery("dimension", "day"),
	}

	// 解析集群ID
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			id := uint(clusterID)
			req.ClusterID = &id
		}
	}

	// 解析异常类型
	if anomalyType := c.Query("anomaly_type"); anomalyType != "" {
		req.AnomalyType = model.AnomalyType(anomalyType)
	}

	// 解析时间范围
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			req.StartTime = &startTime
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if endTime, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			req.EndTime = &endTime
		}
	}

	statistics, err := h.anomalySvc.GetStatistics(req)
	if err != nil {
		h.logger.Errorf("Failed to get statistics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    statistics,
	})
}

// GetActive 获取活跃异常
// @Summary 获取当前活跃的异常
// @Description 获取所有当前活跃状态的节点异常
// @Tags anomalies
// @Produce json
// @Param cluster_id query int false "集群ID"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies/active [get]
func (h *Handler) GetActive(c *gin.Context) {
	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	summary, err := h.anomalySvc.GetAnomalySummary(clusterID)
	if err != nil {
		h.logger.Errorf("Failed to get anomaly summary: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get anomaly summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    summary,
	})
}

// TriggerCheck 手动触发检测
// @Summary 手动触发异常检测
// @Description 立即执行一次节点异常检测
// @Tags anomalies
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies/check [post]
func (h *Handler) TriggerCheck(c *gin.Context) {
	// 检查用户权限：只有 admin 可以手动触发检测
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin can trigger check",
		})
		return
	}

	if err := h.anomalySvc.TriggerCheck(); err != nil {
		h.logger.Errorf("Failed to trigger anomaly check: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to trigger check: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Anomaly check triggered successfully",
	})
}

// GetTypeStatistics 获取异常类型统计
// @Summary 获取异常类型统计
// @Description 获取按异常类型聚合的统计数据
// @Tags anomalies
// @Produce json
// @Param cluster_id query int false "集群ID"
// @Param start_time query string false "开始时间 (RFC3339格式)"
// @Param end_time query string false "结束时间 (RFC3339格式)"
// @Success 200 {object} Response
// @Failure 500 {object} Response
// @Router /anomalies/type-statistics [get]
func (h *Handler) GetTypeStatistics(c *gin.Context) {
	var clusterID *uint
	var startTime, endTime *time.Time

	// 解析集群ID
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	// 解析时间范围
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if st, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &st
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if et, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &et
		}
	}

	statistics, err := h.anomalySvc.GetTypeStatistics(clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get type statistics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get type statistics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    statistics,
	})
}

// TriggerCleanup 手动触发数据清理
func (h *Handler) TriggerCleanup(c *gin.Context) {
	// 检查用户权限：只有 admin 可以手动触发清理
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin can trigger cleanup",
		})
		return
	}

	if err := h.cleanupSvc.Cleanup(); err != nil {
		h.logger.Errorf("Failed to trigger cleanup: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to trigger cleanup: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cleanup completed successfully",
	})
}

// GetCleanupConfig 获取清理配置
func (h *Handler) GetCleanupConfig(c *gin.Context) {
	config := h.cleanupSvc.GetConfig()
	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    config,
	})
}

// UpdateCleanupConfig 更新清理配置
func (h *Handler) UpdateCleanupConfig(c *gin.Context) {
	// 检查用户权限：只有 admin 可以更新配置
	userRole, exists := c.Get("user_role")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin can update cleanup config",
		})
		return
	}

	var config anomaly.CleanupConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request body: " + err.Error(),
		})
		return
	}

	if err := h.cleanupSvc.UpdateConfig(&config); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid configuration: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Configuration updated successfully",
		Data:    config,
	})
}

// GetCleanupStats 获取清理统计信息
func (h *Handler) GetCleanupStats(c *gin.Context) {
	stats, err := h.cleanupSvc.GetCleanupStats()
	if err != nil {
		h.logger.Errorf("Failed to get cleanup stats: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get cleanup stats: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    stats,
	})
}
