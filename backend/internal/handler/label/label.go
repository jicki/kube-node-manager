package label

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/service/label"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 标签管理处理器
type Handler struct {
	labelSvc *label.Service
	logger   *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的标签处理器实例
func NewHandler(labelSvc *label.Service, logger *logger.Logger) *Handler {
	return &Handler{
		labelSvc: labelSvc,
		logger:   logger,
	}
}

// UpdateNodeLabels 更新节点标签
// @Summary 更新节点标签
// @Description 更新指定节点的标签
// @Tags labels
// @Accept json
// @Produce json
// @Param request body label.UpdateLabelsRequest true "标签更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /labels/update [post]
func (h *Handler) UpdateNodeLabels(c *gin.Context) {
	var req label.UpdateLabelsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update labels request: %v", err)
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

	if err := h.labelSvc.UpdateNodeLabels(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to update node labels: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update node labels: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node labels updated successfully",
	})
}

// BatchUpdateLabels 批量更新节点标签
// @Summary 批量更新节点标签
// @Description 批量更新多个节点的标签
// @Tags labels
// @Accept json
// @Produce json
// @Param request body label.BatchUpdateRequest true "批量标签更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /labels/batch-update [post]
func (h *Handler) BatchUpdateLabels(c *gin.Context) {
	var req label.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch update labels request: %v", err)
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

	if err := h.labelSvc.BatchUpdateLabels(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to batch update node labels: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to batch update node labels: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node labels batch updated successfully",
	})
}

// CreateTemplate 创建标签模板
// @Summary 创建标签模板
// @Description 创建新的标签模板
// @Tags label-templates
// @Accept json
// @Produce json
// @Param template body label.TemplateCreateRequest true "模板创建请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /labels/templates [post]
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req label.TemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind create template request: %v", err)
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

	template, err := h.labelSvc.CreateTemplate(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to create label template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create label template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Label template created successfully",
		Data:    template,
	})
}

// UpdateTemplate 更新标签模板
// @Summary 更新标签模板
// @Description 更新现有的标签模板
// @Tags label-templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param template body label.TemplateUpdateRequest true "模板更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /labels/templates/{id} [put]
func (h *Handler) UpdateTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid template ID",
		})
		return
	}

	var req label.TemplateUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update template request: %v", err)
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

	template, err := h.labelSvc.UpdateTemplate(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to update label template: %v", err)
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Template not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update label template: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Label template updated successfully",
		Data:    template,
	})
}

// DeleteTemplate 删除标签模板
// @Summary 删除标签模板
// @Description 删除指定的标签模板
// @Tags label-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /labels/templates/{id} [delete]
func (h *Handler) DeleteTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid template ID",
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

	if err := h.labelSvc.DeleteTemplate(uint(id), userID.(uint)); err != nil {
		h.logger.Error("Failed to delete label template: %v", err)
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Template not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to delete label template: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Label template deleted successfully",
	})
}

// ListTemplates 获取标签模板列表
// @Summary 获取标签模板列表
// @Description 获取标签模板列表，支持分页和筛选
// @Tags label-templates
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param name query string false "模板名称筛选"
// @Success 200 {object} Response
// @Router /labels/templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	var req label.TemplateListRequest
	
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

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.labelSvc.ListTemplates(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to list label templates: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list label templates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    result,
	})
}

// ApplyTemplate 应用标签模板
// @Summary 应用标签模板
// @Description 将标签模板应用到指定节点
// @Tags label-templates
// @Accept json
// @Produce json
// @Param request body label.ApplyTemplateRequest true "模板应用请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /labels/templates/apply [post]
func (h *Handler) ApplyTemplate(c *gin.Context) {
	var req label.ApplyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind apply template request: %v", err)
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

	if err := h.labelSvc.ApplyTemplate(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to apply label template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to apply label template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Label template applied successfully",
	})
}

// GetLabelUsage 获取标签使用情况
// @Summary 获取标签使用情况
// @Description 获取集群中标签的使用统计信息
// @Tags labels
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /labels/usage [get]
func (h *Handler) GetLabelUsage(c *gin.Context) {
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

	usage, err := h.labelSvc.GetLabelUsage(clusterName, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get label usage: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get label usage: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    usage,
	})
}

// GetTemplate 获取单个标签模板详情
// @Summary 获取标签模板详情
// @Description 根据模板ID获取标签模板的详细信息
// @Tags label-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /labels/templates/{id} [get]
func (h *Handler) GetTemplate(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid template ID",
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

	// 通过列表接口获取单个模板（简化实现）
	listReq := label.TemplateListRequest{
		Page:     1,
		PageSize: 100,
	}
	
	result, err := h.labelSvc.ListTemplates(listReq, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get label template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get label template: " + err.Error(),
		})
		return
	}

	var template *label.TemplateInfo
	for _, t := range result.Templates {
		if t.ID == uint(id) {
			template = &t
			break
		}
	}

	if template == nil {
		c.JSON(http.StatusNotFound, Response{
			Code:    http.StatusNotFound,
			Message: "Template not found",
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    template,
	})
}