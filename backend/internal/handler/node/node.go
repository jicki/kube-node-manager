package node

import (
	"net/http"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/node"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 节点管理处理器
type Handler struct {
	nodeSvc *node.Service
	logger  *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的节点处理器实例
func NewHandler(nodeSvc *node.Service, logger *logger.Logger) *Handler {
	return &Handler{
		nodeSvc: nodeSvc,
		logger:  logger,
	}
}

// List 获取节点列表
// @Summary 获取节点列表
// @Description 获取指定集群的节点列表
// @Tags nodes
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Param status query string false "节点状态筛选"
// @Param role query string false "节点角色筛选"
// @Param label_key query string false "标签键筛选"
// @Param label_value query string false "标签值筛选"
// @Param taint_key query string false "污点键筛选"
// @Param taint_value query string false "污点值筛选"
// @Param taint_effect query string false "污点效果筛选"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes [get]
func (h *Handler) List(c *gin.Context) {
	req := node.ListRequest{
		ClusterName: c.Query("cluster_name"),
		Status:      c.Query("status"),
		Role:        c.Query("role"),
		LabelKey:    c.Query("label_key"),
		LabelValue:  c.Query("label_value"),
		TaintKey:    c.Query("taint_key"),
		TaintValue:  c.Query("taint_value"),
		TaintEffect: c.Query("taint_effect"),
	}

	if req.ClusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
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

	nodes, err := h.nodeSvc.List(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to list nodes: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list nodes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    nodes,
	})
}

// Get 获取节点详情
// @Summary 获取节点详情
// @Description 获取指定节点的详细信息
// @Tags nodes
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Param node_name path string true "节点名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /nodes/{node_name} [get]
func (h *Handler) Get(c *gin.Context) {
	req := node.GetRequest{
		ClusterName: c.Query("cluster_name"),
		NodeName:    c.Param("node_name"),
	}

	if req.ClusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	if req.NodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "node_name is required",
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

	nodeInfo, err := h.nodeSvc.Get(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get node: %v", err)
		if err.Error() == "failed to get node: nodes \""+req.NodeName+"\" not found" ||
			err.Error() == "node not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Node not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get node: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    nodeInfo,
	})
}

// Cordon 禁止调度节点
// @Summary 禁止调度节点
// @Description 标记节点为不可调度（不驱逐现有Pod）
// @Tags nodes
// @Accept json
// @Produce json
// @Param node_name path string true "节点名称"
// @Param request body node.CordonRequest true "禁止调度请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/{node_name}/cordon [post]
func (h *Handler) Cordon(c *gin.Context) {
	nodeName := c.Param("node_name")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Node name is required",
		})
		return
	}

	var req node.CordonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind cordon request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 设置从路径参数获取的节点名称
	req.NodeName = nodeName

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以禁止调度节点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can cordon nodes",
		})
		return
	}

	if err := h.nodeSvc.Cordon(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to cordon node: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to cordon node: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node cordoned successfully",
	})
}

// Uncordon 解除调度节点
// @Summary 解除调度节点
// @Description 标记节点为可调度
// @Tags nodes
// @Accept json
// @Produce json
// @Param node_name path string true "节点名称"
// @Param request body node.CordonRequest true "解除调度请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/{node_name}/uncordon [post]
func (h *Handler) Uncordon(c *gin.Context) {
	nodeName := c.Param("node_name")
	if nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Node name is required",
		})
		return
	}

	var req node.CordonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind uncordon request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 设置从路径参数获取的节点名称
	req.NodeName = nodeName

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以解除调度节点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can uncordon nodes",
		})
		return
	}

	if err := h.nodeSvc.Uncordon(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to uncordon node: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to uncordon node: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node uncordoned successfully",
	})
}

// GetSummary 获取节点摘要统计
// @Summary 获取节点摘要
// @Description 获取集群节点的统计摘要信息
// @Tags nodes
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/summary [get]
func (h *Handler) GetSummary(c *gin.Context) {
	clusterName := c.Query("cluster_name")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
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

	summary, err := h.nodeSvc.GetSummary(clusterName, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get node summary: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get node summary: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    summary,
	})
}

// GetMetrics 获取节点指标
// @Summary 获取节点指标
// @Description 获取节点的资源使用指标
// @Tags nodes
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Param node_name path string true "节点名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/{node_name}/metrics [get]
func (h *Handler) GetMetrics(c *gin.Context) {
	clusterName := c.Query("cluster_name")
	nodeName := c.Param("node_name")

	if clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	if nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "node_name is required",
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

	metrics, err := h.nodeSvc.GetMetrics(clusterName, nodeName, userID.(uint))
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

// BatchCordon 批量禁止调度节点
// @Summary 批量禁止调度节点
// @Description 批量标记节点为不可调度
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body node.BatchNodeRequest true "批量禁止调度请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/batch-cordon [post]
func (h *Handler) BatchCordon(c *gin.Context) {
	var req node.BatchNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch cordon request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	if len(req.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "At least one node is required",
		})
		return
	}

	if req.ClusterName == "" {
		req.ClusterName = c.Query("cluster_name")
		if req.ClusterName == "" {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "cluster_name is required",
			})
			return
		}
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以批量禁止调度节点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can batch cordon nodes",
		})
		return
	}

	results, err := h.nodeSvc.BatchCordon(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to batch cordon nodes: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to batch cordon nodes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Batch cordon completed",
		Data:    results,
	})
}

// BatchUncordon 批量解除调度节点
// @Summary 批量解除调度节点
// @Description 批量标记节点为可调度
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body node.BatchNodeRequest true "批量解除调度请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/batch-uncordon [post]
func (h *Handler) BatchUncordon(c *gin.Context) {
	var req node.BatchNodeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch uncordon request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	if len(req.Nodes) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "At least one node is required",
		})
		return
	}

	if req.ClusterName == "" {
		req.ClusterName = c.Query("cluster_name")
		if req.ClusterName == "" {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "cluster_name is required",
			})
			return
		}
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以批量解除调度节点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can batch uncordon nodes",
		})
		return
	}

	results, err := h.nodeSvc.BatchUncordon(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to batch uncordon nodes: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to batch uncordon nodes: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Batch uncordon completed",
		Data:    results,
	})
}

// GetCordonHistory 获取节点禁止调度历史
// @Summary 获取节点禁止调度历史
// @Description 获取指定节点的禁止调度历史记录
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "单节点查询请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /nodes/cordon-history [post]
func (h *Handler) GetCordonHistory(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind cordon history request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	clusterName, ok := req["cluster_name"].(string)
	if !ok || clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	nodeName, ok := req["node_name"].(string)
	if !ok || nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "node_name is required",
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

	history, err := h.nodeSvc.GetCordonHistory(nodeName, clusterName, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get cordon history: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get cordon history: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    history,
	})
}

// GetNodeCordonInfo 获取节点禁止调度信息
// @Summary 获取节点禁止调度信息
// @Description 获取节点的禁止调度信息，包括原因和ISO 8601格式的时间戳 (如: 2025-03-12T07:02:10Z)
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body node.CordonInfoRequest true "获取禁止调度信息请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/cordon-info [post]
func (h *Handler) GetNodeCordonInfo(c *gin.Context) {
	var req node.CordonInfoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind cordon info request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	if req.NodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Node name is required",
		})
		return
	}

	if req.ClusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	// Note: 审计日志由service层处理

	info, err := h.nodeSvc.GetNodeCordonInfo(req.ClusterName, req.NodeName)
	if err != nil {
		h.logger.Error("Failed to get node cordon info: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get node cordon info: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    info,
	})
}

// GetBatchCordonHistory 批量获取节点禁止调度历史
// @Summary 批量获取节点禁止调度历史
// @Description 批量获取指定节点列表的禁止调度历史记录
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "批量查询请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/batch-cordon-history [post]
func (h *Handler) GetBatchCordonHistory(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch cordon history request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	clusterName, ok := req["cluster_name"].(string)
	if !ok || clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	nodeNamesInterface, ok := req["node_names"].([]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "node_names is required and must be an array",
		})
		return
	}

	var nodeNames []string
	for _, name := range nodeNamesInterface {
		if str, ok := name.(string); ok {
			nodeNames = append(nodeNames, str)
		}
	}

	if len(nodeNames) == 0 {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "At least one node name is required",
		})
		return
	}

	_, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	histories, err := h.nodeSvc.GetBatchCordonHistory(nodeNames, clusterName)
	if err != nil {
		h.logger.Error("Failed to get batch cordon history: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get batch cordon history: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    histories,
	})
}

// GetByLabels 根据标签获取节点
// @Summary 根据标签获取节点
// @Description 根据标签选择器获取节点列表
// @Tags nodes
// @Accept json
// @Produce json
// @Param request body map[string]interface{} true "标签查询"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /nodes/by-labels [post]
func (h *Handler) GetByLabels(c *gin.Context) {
	var req map[string]interface{}
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind labels request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	clusterName, ok := req["cluster_name"].(string)
	if !ok || clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	labels, ok := req["labels"].(map[string]interface{})
	if !ok {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "labels is required and must be an object",
		})
		return
	}

	// 转换标签格式
	labelMap := make(map[string]string)
	for k, v := range labels {
		if strVal, ok := v.(string); ok {
			labelMap[k] = strVal
		}
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	nodes, err := h.nodeSvc.GetNodesByLabels(clusterName, labelMap, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get nodes by labels: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get nodes by labels: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    nodes,
	})
}
