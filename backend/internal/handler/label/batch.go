package label

import (
	"fmt"
	"net/http"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/label"

	"github.com/gin-gonic/gin"
)

// BatchAddLabels 批量添加标签
func (h *Handler) BatchAddLabels(c *gin.Context) {
	var req BatchLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch add labels request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以批量添加标签
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量添加标签",
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

	// 为每个节点批量添加标签
	for _, nodeName := range req.Nodes {
		for _, labelData := range req.Labels {
			// 构建标签键值对
			labels := make(map[string]string)
			if key, ok := labelData["key"].(string); ok {
				if value, ok := labelData["value"].(string); ok {
					labels[key] = value
				} else {
					labels[key] = ""
				}
			}

			// 构建BatchUpdateRequest
			batchReq := label.BatchUpdateRequest{
				ClusterName: clusterName,
				NodeNames:   []string{nodeName},
				Labels:      labels,
				Operation:   "add",
			}

			if err := h.labelSvc.BatchUpdateLabels(batchReq, userID.(uint)); err != nil {
				h.logger.Error("Failed to batch add labels for node %s: %v", nodeName, err)
				c.JSON(http.StatusInternalServerError, Response{
					Code:    http.StatusInternalServerError,
					Message: fmt.Sprintf("为节点 %s 添加标签失败: %s", nodeName, err.Error()),
				})
				return
			}
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量添加标签成功",
	})
}

// BatchDeleteLabels 批量删除标签
func (h *Handler) BatchDeleteLabels(c *gin.Context) {
	var req struct {
		Nodes []string `json:"nodes" binding:"required"`
		Keys  []string `json:"keys" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch delete labels request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以批量删除标签
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量删除标签",
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

	// 为每个节点批量删除标签
	for _, nodeName := range req.Nodes {
		// 构建要删除的标签键值对（值设为空字符串表示删除）
		labels := make(map[string]string)
		for _, key := range req.Keys {
			labels[key] = ""
		}

		// 构建BatchUpdateRequest
		batchReq := label.BatchUpdateRequest{
			ClusterName: clusterName,
			NodeNames:   []string{nodeName},
			Labels:      labels,
			Operation:   "remove",
		}

		if err := h.labelSvc.BatchUpdateLabels(batchReq, userID.(uint)); err != nil {
			h.logger.Error("Failed to batch delete labels for node %s: %v", nodeName, err)
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: fmt.Sprintf("为节点 %s 删除标签失败: %s", nodeName, err.Error()),
			})
			return
		}
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量删除标签成功",
	})
}

// BatchAddLabelsWithProgress 带进度推送的批量添加标签
func (h *Handler) BatchAddLabelsWithProgress(c *gin.Context) {
	var req BatchLabelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind batch add labels with progress request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以批量添加标签
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "权限不足。只有管理员和用户角色可以批量添加标签",
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
	taskID := fmt.Sprintf("label_batch_%d_%d", userID.(uint), time.Now().UnixNano())

	// 构建批量更新请求
	labels := make(map[string]string)
	for _, labelData := range req.Labels {
		if key, ok := labelData["key"].(string); ok {
			if value, ok := labelData["value"].(string); ok {
				labels[key] = value
			} else {
				labels[key] = ""
			}
		}
	}

	batchReq := label.BatchUpdateRequest{
		ClusterName: clusterName,
		NodeNames:   req.Nodes,
		Labels:      labels,
		Operation:   "add",
	}

	// 启动异步批量操作
	go func() {
		if err := h.labelSvc.BatchUpdateLabelsWithProgress(batchReq, userID.(uint), taskID); err != nil {
			h.logger.Errorf("Failed to batch add labels with progress: %v", err)
		}
	}()

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "批量添加标签任务已启动",
		Data: map[string]string{
			"task_id": taskID,
		},
	})
}
