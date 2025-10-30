package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// TemplateHandler 模板管理 Handler
type TemplateHandler struct {
	service *ansible.TemplateService
	logger  *logger.Logger
}

// NewTemplateHandler 创建模板 Handler 实例
func NewTemplateHandler(service *ansible.TemplateService, logger *logger.Logger) *TemplateHandler {
	return &TemplateHandler{
		service: service,
		logger:  logger,
	}
}

// ListTemplates 列出模板
// @Summary 列出模板
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param keyword query string false "关键字"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates [get]
func (h *TemplateHandler) ListTemplates(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.TemplateListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	templates, total, err := h.service.ListTemplates(req)
	if err != nil {
		h.logger.Errorf("Failed to list templates: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  templates,
		"total": total,
	})
}

// GetTemplate 获取模板详情
// @Summary 获取模板详情
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates/{id} [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	template, err := h.service.GetTemplate(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": template,
	})
}

// CreateTemplate 创建模板
// @Summary 创建模板
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param template body model.TemplateCreateRequest true "模板信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.TemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	template, err := h.service.CreateTemplate(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    template,
		"message": "Template created successfully",
	})
}

// UpdateTemplate 更新模板
// @Summary 更新模板
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param template body model.TemplateUpdateRequest true "模板信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	var req model.TemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	template, err := h.service.UpdateTemplate(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to update template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    template,
		"message": "Template updated successfully",
	})
}

// DeleteTemplate 删除模板
// @Summary 删除模板
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template id"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	if err := h.service.DeleteTemplate(uint(id), userID.(uint)); err != nil {
		h.logger.Errorf("Failed to delete template: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Template deleted successfully",
	})
}

// ValidateTemplate 验证模板
// @Summary 验证模板
// @Tags Ansible Templates
// @Accept json
// @Produce json
// @Param playbook body map[string]string true "Playbook内容"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/templates/validate [post]
func (h *TemplateHandler) ValidateTemplate(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req struct {
		PlaybookContent string `json:"playbook_content" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.service.ValidatePlaybook(req.PlaybookContent); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"valid":   false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"valid":   true,
		"message": "Playbook is valid",
	})
}

