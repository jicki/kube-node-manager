package features

import (
	"fmt"
	"net/http"

	"kube-node-manager/internal/service/features"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 功能特性处理器
type Handler struct {
	featuresSvc *features.Service
	logger      *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的功能特性处理器实例
func NewHandler(featuresSvc *features.Service, logger *logger.Logger) *Handler {
	return &Handler{
		featuresSvc: featuresSvc,
		logger:      logger,
	}
}

// GetFeatures 获取所有功能特性状态
// @Summary 获取功能特性状态
// @Description 获取所有功能特性的启用状态和配置
// @Tags features
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/features [get]
func (h *Handler) GetFeatures(c *gin.Context) {
	status, err := h.featuresSvc.GetFeatureStatus()
	if err != nil {
		h.logger.Errorf("Failed to get feature status: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get feature status: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    status,
	})
}

// UpdateAutomationEnabled 更新自动化主开关
// @Summary 更新自动化主开关
// @Description 启用或禁用自动化功能模块
// @Tags features
// @Accept json
// @Produce json
// @Param request body map[string]bool true "启用状态"
// @Success 200 {object} Response
// @Router /api/v1/features/automation/enabled [put]
func (h *Handler) UpdateAutomationEnabled(c *gin.Context) {
	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
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

	// 检查用户权限：只有 admin 可以修改功能开关
	userRole, _ := c.Get("user_role")
	roleStr := ""
	if role, ok := userRole.(string); ok {
		roleStr = role
	} else {
		roleStr = fmt.Sprintf("%v", userRole)
	}
	if roleStr != "admin" {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Only admin can modify feature settings",
		})
		return
	}

	if err := h.featuresSvc.UpdateAutomationEnabled(req.Enabled, userID.(uint)); err != nil {
		h.logger.Errorf("Failed to update automation enabled: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update automation enabled: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Automation feature updated successfully",
	})
}

// UpdateAnsibleConfig 更新 Ansible 配置
// @Summary 更新 Ansible 配置
// @Description 更新 Ansible 功能的配置参数
// @Tags features
// @Accept json
// @Produce json
// @Param request body features.AnsibleFeatures true "Ansible 配置"
// @Success 200 {object} Response
// @Router /api/v1/features/automation/ansible [put]
func (h *Handler) UpdateAnsibleConfig(c *gin.Context) {
	var req features.AnsibleFeatures

	if err := c.ShouldBindJSON(&req); err != nil {
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

	// 检查用户权限
	userRole, _ := c.Get("user_role")
	roleStr := ""
	if role, ok := userRole.(string); ok {
		roleStr = role
	} else {
		roleStr = fmt.Sprintf("%v", userRole)
	}
	if roleStr != "admin" {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Only admin can modify feature settings",
		})
		return
	}

	if err := h.featuresSvc.UpdateAnsibleConfig(req, userID.(uint)); err != nil {
		h.logger.Errorf("Failed to update ansible config: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update ansible config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Ansible config updated successfully",
	})
}

// UpdateSSHConfig 更新 SSH 配置
// @Summary 更新 SSH 配置
// @Description 更新 SSH 功能的配置参数
// @Tags features
// @Accept json
// @Produce json
// @Param request body features.SSHFeatures true "SSH 配置"
// @Success 200 {object} Response
// @Router /api/v1/features/automation/ssh [put]
func (h *Handler) UpdateSSHConfig(c *gin.Context) {
	var req features.SSHFeatures

	if err := c.ShouldBindJSON(&req); err != nil {
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

	// 检查用户权限
	userRole, _ := c.Get("user_role")
	roleStr := ""
	if role, ok := userRole.(string); ok {
		roleStr = role
	} else {
		roleStr = fmt.Sprintf("%v", userRole)
	}
	if roleStr != "admin" {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Only admin can modify feature settings",
		})
		return
	}

	if err := h.featuresSvc.UpdateSSHConfig(req, userID.(uint)); err != nil {
		h.logger.Errorf("Failed to update ssh config: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update ssh config: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "SSH config updated successfully",
	})
}
