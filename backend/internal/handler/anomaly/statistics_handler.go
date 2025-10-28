package anomaly

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// GetRoleStatistics 获取按角色聚合统计
func (h *Handler) GetRoleStatistics(c *gin.Context) {
	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	statistics, err := h.anomalySvc.GetRoleStatistics(clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get role statistics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取角色统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取角色统计成功",
		Data:    statistics,
	})
}

// GetClusterAggregate 获取按集群聚合统计
func (h *Handler) GetClusterAggregate(c *gin.Context) {
	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	statistics, err := h.anomalySvc.GetClusterAggregateStatistics(startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get cluster aggregate statistics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取集群统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取集群统计成功",
		Data:    statistics,
	})
}

// GetNodeTrend 获取单节点历史趋势
func (h *Handler) GetNodeTrend(c *gin.Context) {
	clusterIDStr := c.Query("cluster_id")
	nodeName := c.Query("node_name")

	if clusterIDStr == "" || nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "缺少必需参数: cluster_id 和 node_name",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的 cluster_id",
		})
		return
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	trend, err := h.anomalySvc.GetNodeHistoryTrend(uint(clusterID), nodeName, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get node trend: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取节点趋势失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取节点趋势成功",
		Data:    trend,
	})
}

// GetMTTR 获取 MTTR 统计
func (h *Handler) GetMTTR(c *gin.Context) {
	entityType := c.DefaultQuery("entity_type", "node")

	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	statistics, err := h.anomalySvc.GetMTTRStatistics(entityType, clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get MTTR statistics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取 MTTR 统计失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取 MTTR 统计成功",
		Data:    statistics,
	})
}

// GetSLA 获取 SLA 可用性指标
// 支持两种模式：
// 1. 提供 entity_name 时，返回单个实体的 SLA
// 2. 不提供 entity_name 时，返回集群/角色聚合的 SLA
func (h *Handler) GetSLA(c *gin.Context) {
	entityType := c.DefaultQuery("entity_type", "cluster")
	entityName := c.Query("entity_name")

	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	// 如果没有提供 entity_name，返回聚合数据
	if entityName == "" {
		metrics, err := h.anomalySvc.GetSLAMetrics(entityType, "", clusterID, startTime, endTime)
		if err != nil {
			h.logger.Errorf("Failed to get SLA metrics: %v", err)
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "获取 SLA 指标失败",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "获取 SLA 指标成功",
			Data:    metrics,
		})
		return
	}

	// 提供了 entity_name，返回单个实体的数据
	metrics, err := h.anomalySvc.GetSLAMetrics(entityType, entityName, clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get SLA metrics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取 SLA 指标失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取 SLA 指标成功",
		Data:    metrics,
	})
}

// GetRecoveryMetrics 获取恢复率和复发率
// 支持两种模式：
// 1. 提供 entity_name 时，返回单个实体的恢复指标
// 2. 不提供 entity_name 时，返回集群/角色聚合的恢复指标
func (h *Handler) GetRecoveryMetrics(c *gin.Context) {
	entityType := c.DefaultQuery("entity_type", "cluster")
	entityName := c.Query("entity_name")

	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	// 如果没有提供 entity_name，返回聚合数据
	if entityName == "" {
		metrics, err := h.anomalySvc.GetRecoveryMetrics(entityType, "", clusterID, startTime, endTime)
		if err != nil {
			h.logger.Errorf("Failed to get recovery metrics: %v", err)
			c.JSON(http.StatusInternalServerError, Response{
				Code:    500,
				Message: "获取恢复指标失败",
			})
			return
		}

		c.JSON(http.StatusOK, Response{
			Code:    200,
			Message: "获取恢复指标成功",
			Data:    metrics,
		})
		return
	}

	// 提供了 entity_name，返回单个实体的数据
	metrics, err := h.anomalySvc.GetRecoveryMetrics(entityType, entityName, clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get recovery metrics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取恢复指标失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取恢复指标成功",
		Data:    metrics,
	})
}

// GetNodeHealth 获取节点健康度评分
func (h *Handler) GetNodeHealth(c *gin.Context) {
	clusterIDStr := c.Query("cluster_id")
	nodeName := c.Query("node_name")

	if clusterIDStr == "" || nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "缺少必需参数: cluster_id 和 node_name",
		})
		return
	}

	clusterID, err := strconv.ParseUint(clusterIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    400,
			Message: "无效的 cluster_id",
		})
		return
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	score, err := h.anomalySvc.GetNodeHealthScore(uint(clusterID), nodeName, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get node health score: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取节点健康度失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取节点健康度成功",
		Data:    score,
	})
}

// GetHeatmap 获取热力图数据
func (h *Handler) GetHeatmap(c *gin.Context) {
	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	data, err := h.anomalySvc.GetHeatmapData(clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get heatmap data: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取热力图数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取热力图数据成功",
		Data:    data,
	})
}

// GetCalendar 获取日历图数据
func (h *Handler) GetCalendar(c *gin.Context) {
	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	data, err := h.anomalySvc.GetCalendarData(clusterID, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get calendar data: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取日历图数据失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取日历图数据成功",
		Data:    data,
	})
}

// GetTopUnhealthyNodes 获取健康度最低的节点列表
func (h *Handler) GetTopUnhealthyNodes(c *gin.Context) {
	var clusterID *uint
	if clusterIDStr := c.Query("cluster_id"); clusterIDStr != "" {
		if id, err := strconv.ParseUint(clusterIDStr, 10, 32); err == nil {
			cid := uint(id)
			clusterID = &cid
		}
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	var startTime, endTime *time.Time
	if startTimeStr := c.Query("start_time"); startTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
			startTime = &t
		}
	}
	if endTimeStr := c.Query("end_time"); endTimeStr != "" {
		if t, err := time.Parse(time.RFC3339, endTimeStr); err == nil {
			endTime = &t
		}
	}

	nodes, err := h.anomalySvc.GetTopUnhealthyNodes(clusterID, limit, startTime, endTime)
	if err != nil {
		h.logger.Errorf("Failed to get top unhealthy nodes: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    500,
			Message: "获取不健康节点列表失败",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    200,
		Message: "获取不健康节点列表成功",
		Data:    nodes,
	})
}
