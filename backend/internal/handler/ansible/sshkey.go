package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// SSHKeyHandler SSH 密钥 Handler
type SSHKeyHandler struct {
	service *ansible.SSHKeyService
	logger  *logger.Logger
}

// NewSSHKeyHandler 创建 SSH 密钥 Handler 实例
func NewSSHKeyHandler(service *ansible.SSHKeyService, logger *logger.Logger) *SSHKeyHandler {
	return &SSHKeyHandler{
		service: service,
		logger:  logger,
	}
}

// List 列出 SSH 密钥
// @Summary 列出 SSH 密钥
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "关键字"
// @Param type query string false "密钥类型"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys [get]
func (h *SSHKeyHandler) List(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.SSHKeyListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	keys, total, err := h.service.List(req)
	if err != nil {
		h.logger.Errorf("Failed to list SSH keys: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    keys,
		"total":   total,
	})
}

// Get 获取 SSH 密钥详情
// @Summary 获取 SSH 密钥详情
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param id path int true "SSH 密钥 ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys/{id} [get]
func (h *SSHKeyHandler) Get(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ssh key id"})
		return
	}

	key, err := h.service.GetByID(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Success",
		"data":    key,
	})
}

// Create 创建 SSH 密钥
// @Summary 创建 SSH 密钥
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param ssh_key body model.SSHKeyCreateRequest true "SSH 密钥信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys [post]
func (h *SSHKeyHandler) Create(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.SSHKeyCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户 ID
	userID, _ := c.Get("user_id")

	key, err := h.service.Create(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "SSH key created successfully",
		"data":    key,
	})
}

// Update 更新 SSH 密钥
// @Summary 更新 SSH 密钥
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param id path int true "SSH 密钥 ID"
// @Param ssh_key body model.SSHKeyUpdateRequest true "SSH 密钥信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys/{id} [put]
func (h *SSHKeyHandler) Update(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ssh key id"})
		return
	}

	var req model.SSHKeyUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户 ID
	userID, _ := c.Get("user_id")

	key, err := h.service.Update(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to update SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "SSH key updated successfully",
		"data":    key,
	})
}

// Delete 删除 SSH 密钥
// @Summary 删除 SSH 密钥
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param id path int true "SSH 密钥 ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys/{id} [delete]
func (h *SSHKeyHandler) Delete(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ssh key id"})
		return
	}

	// 获取用户 ID
	userID, _ := c.Get("user_id")

	if err := h.service.Delete(uint(id), userID.(uint)); err != nil {
		h.logger.Errorf("Failed to delete SSH key: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "SSH key deleted successfully",
	})
}

// TestConnection 测试 SSH 连接
// @Summary 测试 SSH 连接
// @Tags Ansible SSH Keys
// @Accept json
// @Produce json
// @Param id path int true "SSH 密钥 ID"
// @Param request body map[string]string true "测试主机信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/ssh-keys/{id}/test [post]
func (h *SSHKeyHandler) TestConnection(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ssh key id"})
		return
	}

	var req struct {
		Host string `json:"host" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.TestConnection(uint(id), req.Host); err != nil {
		h.logger.Errorf("SSH connection test failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"error":   err.Error(),
			"message": "Connection test failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "Connection test successful",
	})
}

