package cluster

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 集群管理处理器
type Handler struct {
	clusterSvc *cluster.Service
	logger     *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的集群处理器实例
func NewHandler(clusterSvc *cluster.Service, logger *logger.Logger) *Handler {
	return &Handler{
		clusterSvc: clusterSvc,
		logger:     logger,
	}
}

// Create 创建集群
// @Summary 创建集群
// @Description 创建新的Kubernetes集群连接
// @Tags clusters
// @Accept json
// @Produce json
// @Param cluster body cluster.CreateRequest true "集群信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /clusters [post]
func (h *Handler) Create(c *gin.Context) {
	var req cluster.CreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create cluster request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	// 添加调试日志
	h.logger.Infof("接收到集群创建请求: Name=%s, Description=%s, KubeConfig长度=%d",
		req.Name, req.Description, len(req.KubeConfig))

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	clusterInfo, err := h.clusterSvc.Create(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to create cluster: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create cluster: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cluster created successfully",
		Data:    clusterInfo,
	})
}

// GetByID 根据ID获取集群详情
// @Summary 获取集群详情
// @Description 根据集群ID获取集群详细信息
// @Tags clusters
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
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

	clusterInfo, err := h.clusterSvc.GetByID(uint(id), userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get cluster: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get cluster: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    clusterInfo,
	})
}

// Update 更新集群信息
// @Summary 更新集群
// @Description 更新集群信息
// @Tags clusters
// @Accept json
// @Produce json
// @Param id path int true "集群ID"
// @Param cluster body cluster.UpdateRequest true "更新信息"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid cluster ID",
		})
		return
	}

	var req cluster.UpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update cluster request: %v", err)
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

	clusterInfo, err := h.clusterSvc.Update(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to update cluster: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update cluster: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cluster updated successfully",
		Data:    clusterInfo,
	})
}

// Delete 删除集群
// @Summary 删除集群
// @Description 删除集群连接
// @Tags clusters
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
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

	if err := h.clusterSvc.Delete(uint(id), userID.(uint)); err != nil {
		h.logger.Error("Failed to delete cluster: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to delete cluster: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cluster deleted successfully",
	})
}

// List 获取集群列表
// @Summary 获取集群列表
// @Description 获取集群列表，支持分页和筛选
// @Tags clusters
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param name query string false "集群名称筛选"
// @Param status query string false "状态筛选"
// @Success 200 {object} Response
// @Router /clusters [get]
func (h *Handler) List(c *gin.Context) {
	var req cluster.ListRequest

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
	req.Name = c.Query("name")
	req.Status = model.ClusterStatus(c.Query("status"))

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.clusterSvc.List(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to list clusters: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list clusters: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    result,
	})
}

// Sync 同步集群信息
// @Summary 同步集群信息
// @Description 从Kubernetes API同步集群信息
// @Tags clusters
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/sync [post]
func (h *Handler) Sync(c *gin.Context) {
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

	if err := h.clusterSvc.Sync(uint(id), userID.(uint)); err != nil {
		h.logger.Error("Failed to sync cluster: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to sync cluster: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cluster synced successfully",
	})
}

// GetNodes 获取集群节点
// @Summary 获取集群节点
// @Description 获取集群中的所有节点信息
// @Tags clusters
// @Produce json
// @Param id path int true "集群ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /clusters/{id}/nodes [get]
func (h *Handler) GetNodes(c *gin.Context) {
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

	nodes, err := h.clusterSvc.GetNodes(uint(id), userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get cluster nodes: %v", err)
		if err.Error() == "cluster not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Cluster not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to get cluster nodes: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    nodes,
	})
}

// TestConnection 测试集群连接
// @Summary 测试集群连接
// @Description 测试Kubernetes集群连接是否正常
// @Tags clusters
// @Accept json
// @Produce json
// @Param config body map[string]string true "Kubeconfig"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Router /clusters/test [post]
func (h *Handler) TestConnection(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	kubeconfig, exists := req["kube_config"]
	if !exists || kubeconfig == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "kube_config is required",
		})
		return
	}

	if err := h.clusterSvc.TestConnection(kubeconfig); err != nil {
		h.logger.Error("Failed to test cluster connection: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Connection test failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Connection test successful",
	})
}

// CheckStatus 检查集群状态
// @Summary 检查集群状态
// @Description 检查指定集群的连接状态
// @Tags clusters
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /clusters/status [get]
func (h *Handler) CheckStatus(c *gin.Context) {
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

	if err := h.clusterSvc.CheckClusterStatus(clusterName, userID.(uint)); err != nil {
		h.logger.Error("Cluster status check failed for %s: %v", clusterName, err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Cluster status check failed: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Cluster is healthy and accessible",
		Data: map[string]interface{}{
			"cluster_name": clusterName,
			"status":       "healthy",
		},
	})
}
