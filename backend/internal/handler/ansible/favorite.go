package ansible

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// FavoriteHandler 收藏处理器
type FavoriteHandler struct {
	*Handler
}

// NewFavoriteHandler 创建收藏处理器
func NewFavoriteHandler(h *Handler) *FavoriteHandler {
	return &FavoriteHandler{Handler: h}
}

// AddFavorite 添加收藏
// @Summary 添加收藏
// @Tags Ansible
// @Param body body object true "收藏信息"
// @Success 200
// @Router /ansible/favorites [post]
func (h *FavoriteHandler) AddFavorite(c *gin.Context) {
	userID := h.getUserID(c)
	
	var req struct {
		TargetType string `json:"target_type" binding:"required"` // task/template/inventory
		TargetID   uint   `json:"target_id" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// 验证目标类型
	if req.TargetType != "task" && req.TargetType != "template" && req.TargetType != "inventory" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_type, must be task/template/inventory"})
		return
	}
	
	favSvc := h.service.GetFavoriteService()
	if err := favSvc.AddFavorite(userID, req.TargetType, req.TargetID); err != nil {
		if err.Error() == "already in favorites" {
			c.JSON(http.StatusConflict, gin.H{"error": "已在收藏夹中"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "收藏成功"})
}

// RemoveFavorite 移除收藏
// @Summary 移除收藏
// @Tags Ansible
// @Param target_type query string true "目标类型"
// @Param target_id query int true "目标ID"
// @Success 200
// @Router /ansible/favorites [delete]
func (h *FavoriteHandler) RemoveFavorite(c *gin.Context) {
	userID := h.getUserID(c)
	
	targetType := c.Query("target_type")
	targetIDStr := c.Query("target_id")
	
	if targetType == "" || targetIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "target_type and target_id are required"})
		return
	}
	
	targetID, err := strconv.ParseUint(targetIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid target_id"})
		return
	}
	
	favSvc := h.service.GetFavoriteService()
	if err := favSvc.RemoveFavorite(userID, targetType, uint(targetID)); err != nil {
		if err.Error() == "favorite not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "收藏不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "取消收藏成功"})
}

// ListFavorites 列出收藏
// @Summary 列出收藏
// @Tags Ansible
// @Param target_type query string false "目标类型"
// @Success 200 {array} model.AnsibleFavorite
// @Router /ansible/favorites [get]
func (h *FavoriteHandler) ListFavorites(c *gin.Context) {
	userID := h.getUserID(c)
	targetType := c.Query("target_type")
	
	favSvc := h.service.GetFavoriteService()
	favorites, err := favSvc.ListFavorites(userID, targetType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"data":  favorites,
		"total": len(favorites),
	})
}

// GetRecentTasks 获取最近使用的任务
// @Summary 获取最近使用的任务
// @Tags Ansible
// @Param limit query int false "限制数量" default(10)
// @Success 200 {array} model.AnsibleTaskHistory
// @Router /ansible/recent-tasks [get]
func (h *FavoriteHandler) GetRecentTasks(c *gin.Context) {
	userID := h.getUserID(c)
	
	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}
	
	favSvc := h.service.GetFavoriteService()
	history, err := favSvc.GetRecentTaskHistory(userID, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"data":  history,
		"total": len(history),
	})
}

// GetTaskHistory 获取任务历史详情
// @Summary 获取任务历史详情
// @Tags Ansible
// @Param id path int true "历史记录ID"
// @Success 200 {object} model.AnsibleTaskHistory
// @Router /ansible/task-history/:id [get]
func (h *FavoriteHandler) GetTaskHistory(c *gin.Context) {
	userID := h.getUserID(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	
	favSvc := h.service.GetFavoriteService()
	history, err := favSvc.GetTaskHistory(uint(id), userID)
	if err != nil {
		if err.Error() == "task history not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "历史记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": history})
}

// DeleteTaskHistory 删除任务历史
// @Summary 删除任务历史
// @Tags Ansible
// @Param id path int true "历史记录ID"
// @Success 200
// @Router /ansible/task-history/:id [delete]
func (h *FavoriteHandler) DeleteTaskHistory(c *gin.Context) {
	userID := h.getUserID(c)
	
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	
	favSvc := h.service.GetFavoriteService()
	if err := favSvc.DeleteTaskHistory(uint(id), userID); err != nil {
		if err.Error() == "task history not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "历史记录不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}

