package sshkey

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/sshkey"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler SSH 密钥处理器
type Handler struct {
	service *sshkey.Service
	logger  *logger.Logger
}

// NewHandler 创建 SSH 密钥处理器
func NewHandler(service *sshkey.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service: service,
		logger:  logger,
	}
}

// List 获取 SSH 密钥列表
func (h *Handler) List(c *gin.Context) {
	var req model.SSHKeyListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 设置默认值
	if req.Page == 0 {
		req.Page = 1
	}
	if req.PageSize == 0 {
		req.PageSize = 10
	}

	keys, total, err := h.service.List(req)
	if err != nil {
		h.logger.Errorf("Failed to list SSH keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  keys,
		"total": total,
		"page":  req.Page,
		"size":  req.PageSize,
	})
}

// Get 根据 ID 获取 SSH 密钥
func (h *Handler) Get(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	key, err := h.service.GetByID(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get SSH key: %v", err)
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, key)
}

// Create 创建 SSH 密钥
func (h *Handler) Create(c *gin.Context) {
	var req model.SSHKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	key, err := h.service.Create(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Infof("SSH key created: %s (ID: %d) by user %v", key.Name, key.ID, userID)
	c.JSON(http.StatusCreated, key)
}

// Update 更新 SSH 密钥
func (h *Handler) Update(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	var req model.SSHKeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取当前用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	key, err := h.service.Update(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to update SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Infof("SSH key updated: %s (ID: %d) by user %v", key.Name, key.ID, userID)
	c.JSON(http.StatusOK, key)
}

// Delete 删除 SSH 密钥
func (h *Handler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ID"})
		return
	}

	// 获取当前用户 ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	if err := h.service.Delete(uint(id)); err != nil {
		h.logger.Errorf("Failed to delete SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	h.logger.Infof("SSH key deleted: ID %d by user %v", id, userID)
	c.JSON(http.StatusOK, gin.H{"message": "SSH key deleted successfully"})
}

