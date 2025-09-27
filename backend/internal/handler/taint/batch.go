package taint

import (
	"fmt"
	"net/http"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/taint"

	"github.com/gin-gonic/gin"
)

// BatchTaintRequest 批量污点请求
type BatchTaintRequest struct {
	Nodes   []string                 `json:"nodes" binding:"required"`
	Taints  []map[string]interface{} `json:"taints" binding:"required"`
	Cluster string                   `json:"cluster"`
}

// BatchAddTaints 批量添加污点
func (h *Handler) BatchAddTaints(c *gin.Context) {
	var req BatchTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch add taints request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以批量添加污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量添加污点",
		})
		return
	}

	// 从请求参数中获取集群名称，如果没有提供则从查询参数获取
	clusterName := req.Cluster
	if clusterName == "" {
		clusterName = c.Query("cluster_name")
		if clusterName == "" {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "缺少集群名称参数",
			})
			return
		}
	}

	// 为每个节点批量添加污点
	for _, nodeName := range req.Nodes {
		for _, taintData := range req.Taints {
			// 构建污点数据
			var taintReq taint.UpdateTaintsRequest
			taintReq.ClusterName = clusterName
			taintReq.NodeName = nodeName

			// 解析污点数据
			if key, ok := taintData["key"].(string); ok {
				if value, ok := taintData["value"].(string); ok {
					if effect, ok := taintData["effect"].(string); ok {
						taintReq.Taints = []k8s.TaintInfo{{
							Key:    key,
							Value:  value,
							Effect: effect,
						}}
						taintReq.Operation = "add"

						if err := h.taintSvc.UpdateNodeTaints(taintReq, userID.(uint)); err != nil {
							h.logger.Error("Failed to batch add taints for node %s: %v", nodeName, err)
							c.JSON(http.StatusInternalServerError, Response{
								Code:    http.StatusInternalServerError,
								Message: fmt.Sprintf("为节点 %s 添加污点失败: %s", nodeName, err.Error()),
							})
							return
						}
					}
				}
			}
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量添加污点成功",
	})
}

// BatchDeleteTaints 批量删除污点
func (h *Handler) BatchDeleteTaints(c *gin.Context) {
	var req struct {
		Nodes []string `json:"nodes" binding:"required"`
		Keys  []string `json:"keys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch delete taints request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以批量删除污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量删除污点",
		})
		return
	}

	// 从查询参数获取集群名称
	clusterName := c.Query("cluster_name")
	if clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "缺少集群名称参数",
		})
		return
	}

	// 为每个节点批量删除污点
	for _, nodeName := range req.Nodes {
		// 构建删除污点的请求
		var taintReq taint.UpdateTaintsRequest
		taintReq.ClusterName = clusterName
		taintReq.NodeName = nodeName
		taintReq.Operation = "remove"

		// 构建要删除的污点键
		for _, key := range req.Keys {
			taintReq.Taints = append(taintReq.Taints, k8s.TaintInfo{
				Key: key,
			})
		}

		if err := h.taintSvc.UpdateNodeTaints(taintReq, userID.(uint)); err != nil {
			h.logger.Error("Failed to batch delete taints for node %s: %v", nodeName, err)
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("为节点 %s 删除污点失败: %s", nodeName, err.Error()),
			})
			return
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量删除污点成功",
	})
}

// BatchAddTaintsWithProgress 带进度推送的批量添加污点
func (h *Handler) BatchAddTaintsWithProgress(c *gin.Context) {
	var req BatchTaintRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind batch add taints with progress request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "请求参数错误: " + err.Error(),
		})
		return
	}

	// 获取当前用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "用户未认证",
		})
		return
	}

	// 检查用户权限：只有 admin 和 user 角色可以批量添加污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量添加污点",
		})
		return
	}

	// 从请求参数中获取集群名称，如果没有提供则从查询参数获取
	clusterName := req.Cluster
	if clusterName == "" {
		clusterName = c.Query("cluster_name")
		if clusterName == "" {
			c.JSON(http.StatusBadRequest, Response{
				Code:    http.StatusBadRequest,
				Message: "缺少集群名称参数",
			})
			return
		}
	}

	// 生成任务ID
	taskID := fmt.Sprintf("taint_batch_%d_%d", userID.(uint), time.Now().UnixNano())

	// 构建批量更新请求
	var taints []k8s.TaintInfo
	for _, taintData := range req.Taints {
		if key, ok := taintData["key"].(string); ok {
			if value, ok := taintData["value"].(string); ok {
				if effect, ok := taintData["effect"].(string); ok {
					taints = append(taints, k8s.TaintInfo{
						Key:    key,
						Value:  value,
						Effect: effect,
					})
				}
			}
		}
	}

	batchReq := taint.BatchUpdateRequest{
		ClusterName: clusterName,
		NodeNames:   req.Nodes,
		Taints:      taints,
		Operation:   "add",
	}

	// 启动异步批量操作
	go func() {
		if err := h.taintSvc.BatchUpdateTaintsWithProgress(batchReq, userID.(uint), taskID); err != nil {
			h.logger.Errorf("Failed to batch add taints with progress: %v", err)
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量添加污点任务已启动",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}
