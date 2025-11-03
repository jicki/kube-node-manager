package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TagHandler 处理标签相关的请求
type TagHandler struct {
	service *ansible.Service
	logger  *logger.Logger
}

// NewTagHandler 创建 TagHandler 实例
func NewTagHandler(service *ansible.Service, logger *logger.Logger) *TagHandler {
	return &TagHandler{
		service: service,
		logger:  logger,
	}
}

// CreateTag 创建标签
func (h *TagHandler) CreateTag(c *gin.Context) {
	userID := c.GetUint("user_id")
	
	var req model.TagCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tag, err := h.service.GetTagService().CreateTag(req, userID)
	if err != nil {
		h.logger.Errorf("Failed to create tag: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "data": tag})
}

// ListTags 获取标签列表
func (h *TagHandler) ListTags(c *gin.Context) {
	var req model.TagListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tags, total, err := h.service.GetTagService().ListTags(req)
	if err != nil {
		h.logger.Errorf("Failed to list tags: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": gin.H{
			"list":  tags,
			"total": total,
		},
	})
}

// UpdateTag 更新标签
func (h *TagHandler) UpdateTag(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
		return
	}

	var req model.TagUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.GetTagService().UpdateTag(uint(id), req, userID); err != nil {
		h.logger.Errorf("Failed to update tag: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success"})
}

// DeleteTag 删除标签
func (h *TagHandler) DeleteTag(c *gin.Context) {
	userID := c.GetUint("user_id")
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid tag id"})
		return
	}

	if err := h.service.GetTagService().DeleteTag(uint(id), userID); err != nil {
		h.logger.Errorf("Failed to delete tag: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success"})
}

// BatchTagOperation 批量标签操作
func (h *TagHandler) BatchTagOperation(c *gin.Context) {
	var req model.BatchTagOperationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tagSvc := h.service.GetTagService()
	var err error
	
	if req.Action == "add" {
		err = tagSvc.BatchAddTagsToTasks(req.TaskIDs, req.TagIDs)
	} else {
		err = tagSvc.BatchRemoveTagsFromTasks(req.TaskIDs, req.TagIDs)
	}

	if err != nil {
		h.logger.Errorf("Failed to batch %s tags: %v", req.Action, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "message": "success"})
}

