package monitoring

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/monitoring"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 监控处理器
type Handler struct {
	monitoringSvc *monitoring.Service
	logger        *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的监控处理器实例
func NewHandler(monitoringSvc *monitoring.Service, logger *logger.Logger) *Handler {
	return &Handler{
		monitoringSvc: monitoringSvc,
		logger:        logger,
	}
}

// GetMonitoringStatus 获取集群监控状态
// @Summary 获取集群监控状态
// @Description 获取指定集群的监控配置和状态信息
// @Tags monitoring
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/status [get]
func (h *Handler) GetMonitoringStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	status, err := h.monitoringSvc.GetMonitoringStatus(uint(id), userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get monitoring status: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get monitoring status: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    status,
	})
}

// GetNodeMetrics 获取节点监控指标
// @Summary 获取节点监控指标
// @Description 获取集群中所有节点的监控指标数据
// @Tags monitoring
// @Produce json
// @Param id path int true "集群ID"
// @Param timeRange query string false "时间范围" default("1h")
// @Param step query string false "数据间隔" default("15s")
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/nodes [get]
func (h *Handler) GetNodeMetrics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 解析查询参数
	timeRange := c.DefaultQuery("timeRange", "1h")
	step := c.DefaultQuery("step", "15s")

	metrics, err := h.monitoringSvc.GetNodeMetrics(uint(id), userID.(uint), timeRange, step)
	if err != nil {
		h.logger.Error("Failed to get node metrics: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get node metrics: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    metrics,
	})
}

// GetNetworkTopology 获取网络拓扑数据
// @Summary 获取网络拓扑数据
// @Description 获取集群网络拓扑结构信息
// @Tags monitoring
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/topology [get]
func (h *Handler) GetNetworkTopology(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	topology, err := h.monitoringSvc.GetNetworkTopology(uint(id), userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get network topology: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get network topology: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    topology,
	})
}

// TestNetworkConnectivity 测试网络连通性
// @Summary 测试网络连通性
// @Description 执行集群节点间的网络连通性测试
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param request body monitoring.ConnectivityTestRequest true "测试参数"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/connectivity [post]
func (h *Handler) TestNetworkConnectivity(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	var req monitoring.ConnectivityTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind connectivity test request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	results, err := h.monitoringSvc.TestNetworkConnectivity(uint(id), userID.(uint), req)
	if err != nil {
		h.logger.Error("Failed to test network connectivity: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to test network connectivity: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Connectivity test completed",
		Data:    results,
	})
}

// GetAlerts 获取告警信息
// @Summary 获取告警信息
// @Description 获取集群的告警信息
// @Tags monitoring
// @Produce json
// @Param id path int true "集群ID"
// @Param severity query string false "告警级别"
// @Param limit query int false "返回数量限制" default(50)
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/alerts [get]
func (h *Handler) GetAlerts(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	severity := c.Query("severity")
	limitStr := c.DefaultQuery("limit", "50")
	limit, _ := strconv.Atoi(limitStr)

	alerts, err := h.monitoringSvc.GetAlerts(uint(id), userID.(uint), severity, limit)
	if err != nil {
		h.logger.Error("Failed to get alerts: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get alerts: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    alerts,
	})
}

// TestMonitoringEndpoint 测试监控端点连接
// @Summary 测试监控端点连接
// @Description 测试 Prometheus 或 VictoriaMetrics 端点是否可访问
// @Tags monitoring
// @Accept json
// @Produce json
// @Param request body monitoring.EndpointTestRequest true "端点测试参数"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /monitoring/test [post]
func (h *Handler) TestMonitoringEndpoint(c *gin.Context) {
	var req monitoring.EndpointTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind endpoint test request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.monitoringSvc.TestMonitoringEndpoint(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to test monitoring endpoint: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to test monitoring endpoint: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Endpoint test completed",
		Data:    result,
	})
}

// PrometheusQuery 执行 Prometheus 查询
// @Summary 执行 Prometheus 查询
// @Description 对指定集群的 Prometheus 执行 PromQL 查询
// @Tags monitoring
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param request body monitoring.PrometheusQueryRequest true "查询参数"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/monitoring/query [post]
func (h *Handler) PrometheusQuery(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	var req monitoring.PrometheusQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind prometheus query request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.monitoringSvc.ExecutePrometheusQuery(uint(id), userID.(uint), req)
	if err != nil {
		h.logger.Error("Failed to execute prometheus query: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to execute query: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Query executed successfully",
		Data:    result,
	})
}