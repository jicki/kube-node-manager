package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/feishu"
	"kube-node-manager/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles Feishu-related HTTP requests
type Handler struct {
	service      *feishu.Service
	auditService *audit.Service
	logger       *logger.Logger
}

// NewHandler creates a new Feishu handler
func NewHandler(service *feishu.Service, auditService *audit.Service, logger *logger.Logger) *Handler {
	return &Handler{
		service:      service,
		auditService: auditService,
		logger:       logger,
	}
}

// GetSettings retrieves Feishu settings
// GET /api/v1/feishu/settings
func (h *Handler) GetSettings(c *gin.Context) {
	settings, err := h.service.GetSettingsWithStatus()
	if err != nil {
		h.logger.Error("Failed to get Feishu settings: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get Feishu settings"})
		return
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates Feishu settings
// PUT /api/v1/feishu/settings
func (h *Handler) UpdateSettings(c *gin.Context) {
	var req struct {
		Enabled    bool   `json:"enabled"`
		AppID      string `json:"app_id"`
		AppSecret  string `json:"app_secret"`
		BotEnabled bool   `json:"bot_enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	// Validate app_id if enabled
	if req.Enabled && req.AppID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID is required when Feishu is enabled"})
		return
	}

	settings, err := h.service.UpdateSettings(req.Enabled, req.AppID, req.AppSecret, req.BotEnabled)
	if err != nil {
		h.logger.Error("Failed to update Feishu settings: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update Feishu settings"})
		return
	}

	// Record audit log
	userID, _ := c.Get("user_id")
	h.auditService.Log(audit.LogRequest{
		UserID:       userID.(uint),
		Action:       "update",
		ResourceType: "feishu_settings",
		Details:      "更新飞书配置（长连接模式）",
		Status:       "success",
		IPAddress:    c.ClientIP(),
	})

	h.logger.Info("Feishu settings updated successfully")

	// 返回包含连接状态的响应
	response := settings.ToResponse()
	response.BotConnected = h.service.IsEventClientConnected()
	c.JSON(http.StatusOK, response)
}

// TestConnection tests Feishu API connection
// POST /api/v1/feishu/test
func (h *Handler) TestConnection(c *gin.Context) {
	var req struct {
		AppID     string `json:"app_id" binding:"required"`
		AppSecret string `json:"app_secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "App ID and App Secret are required"})
		return
	}

	if err := h.service.TestConnection(req.AppID, req.AppSecret); err != nil {
		h.logger.Error("Feishu connection test failed: " + err.Error())
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Record audit log
	userID, _ := c.Get("user_id")
	h.auditService.Log(audit.LogRequest{
		UserID:       userID.(uint),
		Action:       "test",
		ResourceType: "feishu_settings",
		Details:      "测试飞书连接",
		Status:       "success",
		IPAddress:    c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "Connection successful"})
}

// QueryGroup queries information about a specific chat group
// POST /api/v1/feishu/groups/query
func (h *Handler) QueryGroup(c *gin.Context) {
	var req struct {
		ChatID string `json:"chat_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Chat ID is required"})
		return
	}

	chatInfo, err := h.service.GetChatInfo(req.ChatID)
	if err != nil {
		h.logger.Error("Failed to query chat group: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Record audit log
	userID, _ := c.Get("user_id")
	h.auditService.Log(audit.LogRequest{
		UserID:       userID.(uint),
		Action:       "query",
		ResourceType: "feishu_group",
		Details:      fmt.Sprintf("查询飞书群组信息: %s", req.ChatID),
		Status:       "success",
		IPAddress:    c.ClientIP(),
	})

	c.JSON(http.StatusOK, chatInfo)
}

// ListGroups lists all chat groups the bot is a member of
// GET /api/v1/feishu/groups
func (h *Handler) ListGroups(c *gin.Context) {
	chats, err := h.service.ListChats()
	if err != nil {
		h.logger.Error("Failed to list chat groups: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, chats)
}

// GetBinding retrieves the current user's Feishu binding
// GET /api/v1/feishu/bind
func (h *Handler) GetBinding(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	binding, err := h.service.GetBindingByUserID(userID.(uint))
	if err != nil {
		// User not bound, return empty
		c.JSON(http.StatusOK, gin.H{"bound": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"bound":          true,
		"feishu_user_id": binding.FeishuUserID,
		"feishu_name":    binding.FeishuName,
		"created_at":     binding.CreatedAt,
	})
}

// BindUser binds a Feishu user to the current system user
// POST /api/v1/feishu/bind
func (h *Handler) BindUser(c *gin.Context) {
	var req struct {
		FeishuUserID string `json:"feishu_user_id" binding:"required"`
		FeishuName   string `json:"feishu_name"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	username, _ := c.Get("username")

	_, err := h.service.BindUser(req.FeishuUserID, userID.(uint), username.(string), req.FeishuName)
	if err != nil {
		h.logger.Error("Failed to bind Feishu user: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to bind user"})
		return
	}

	// Record audit log
	h.auditService.Log(audit.LogRequest{
		UserID:       userID.(uint),
		Action:       "bind",
		ResourceType: "feishu_user",
		Details:      fmt.Sprintf("绑定飞书账号: %s", req.FeishuUserID),
		Status:       "success",
		IPAddress:    c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "User bound successfully"})
}

// UnbindUser unbinds the current user's Feishu account
// DELETE /api/v1/feishu/bind
func (h *Handler) UnbindUser(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	if err := h.service.UnbindUser(userID.(uint)); err != nil {
		h.logger.Error("Failed to unbind Feishu user: " + err.Error())
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unbind user"})
		return
	}

	// Record audit log
	h.auditService.Log(audit.LogRequest{
		UserID:       userID.(uint),
		Action:       "unbind",
		ResourceType: "feishu_user",
		Details:      "解绑飞书账号",
		Status:       "success",
		IPAddress:    c.ClientIP(),
	})

	c.JSON(http.StatusOK, gin.H{"message": "User unbound successfully"})
}
