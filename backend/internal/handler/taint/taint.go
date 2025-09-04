package taint

import (
	"net/http"
	"strconv"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/taint"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 污点管理处理器
type Handler struct {
	taintSvc *taint.Service
	logger   *logger.Logger
}

// Response 通用响应结构
type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewHandler 创建新的污点处理器实例
func NewHandler(taintSvc *taint.Service, logger *logger.Logger) *Handler {
	return &Handler{
		taintSvc: taintSvc,
		logger:   logger,
	}
}

// UpdateNodeTaints 更新节点污点
// @Summary 更新节点污点
// @Description 更新指定节点的污点
// @Tags taints
// @Accept json
// @Produce json
// @Param request body taint.UpdateTaintsRequest true "污点更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/update [post]
func (h *Handler) UpdateNodeTaints(c *gin.Context) {
	var req taint.UpdateTaintsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind update taints request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以更新节点污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can update node taints",
		})
		return
	}

	if err := h.taintSvc.UpdateNodeTaints(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to update node taints: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to update node taints: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node taints updated successfully",
	})
}

// BatchUpdateTaints 批量更新节点污点
// @Summary 批量更新节点污点
// @Description 批量更新多个节点的污点
// @Tags taints
// @Accept json
// @Produce json
// @Param request body taint.BatchUpdateRequest true "批量污点更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/batch-update [post]
func (h *Handler) BatchUpdateTaints(c *gin.Context) {
	var req taint.BatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind batch update taints request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以批量更新节点污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can batch update node taints",
		})
		return
	}

	if err := h.taintSvc.BatchUpdateTaints(req, userID.(uint)); err != nil {
		h.logger.Error("Failed to batch update node taints: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to batch update node taints: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Node taints batch updated successfully",
	})
}

// RemoveTaint 移除指定污点
// @Summary 移除节点污点
// @Description 从节点移除指定的污点
// @Tags taints
// @Accept json
// @Produce json
// @Param request body map[string]string true "移除污点请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/remove [post]
func (h *Handler) RemoveTaint(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Failed to bind remove taint request: %v", err)
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "Invalid request parameters: " + err.Error(),
		})
		return
	}

	clusterName, ok := req["cluster_name"]
	if !ok || clusterName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "cluster_name is required",
		})
		return
	}

	nodeName, ok := req["node_name"]
	if !ok || nodeName == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "node_name is required",
		})
		return
	}

	taintKey, ok := req["taint_key"]
	if !ok || taintKey == "" {
		c.JSON(http.StatusBadRequest, Response{
			Code:    http.StatusBadRequest,
			Message: "taint_key is required",
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

	// 检查用户权限：只有 admin 和 user 角色可以移除节点污点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can remove node taints",
		})
		return
	}

	if err := h.taintSvc.RemoveTaint(clusterName, nodeName, taintKey, userID.(uint)); err != nil {
		h.logger.Error("Failed to remove taint: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to remove taint: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Taint removed successfully",
	})
}

// CreateTemplate 创建污点模板
// @Summary 创建污点模板
// @Description 创建新的污点模板
// @Tags taint-templates
// @Accept json
// @Produce json
// @Param template body taint.TemplateCreateRequest true "模板创建请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/templates [post]
func (h *Handler) CreateTemplate(c *gin.Context) {
	var req taint.TemplateCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind create template request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以创建污点模板
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can create taint templates",
		})
		return
	}

	template, err := h.taintSvc.CreateTemplate(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create taint template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to create taint template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Taint template created successfully",
		Data:    template,
	})
}

// UpdateTemplate 更新污点模板
// @Summary 更新污点模板
// @Description 更新现有的污点模板
// @Tags taint-templates
// @Accept json
// @Produce json
// @Param id path int true "模板ID"
// @Param template body taint.TemplateUpdateRequest true "模板更新请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /taints/templates/{id} [put]
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

	var req taint.TemplateUpdateRequest
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

	// 检查用户权限：只有 admin 和 user 角色可以更新污点模板
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can update taint templates",
		})
		return
	}

	template, err := h.taintSvc.UpdateTemplate(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to update taint template: %v", err)
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Template not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to update taint template: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Taint template updated successfully",
		Data:    template,
	})
}

// DeleteTemplate 删除污点模板
// @Summary 删除污点模板
// @Description 删除指定的污点模板
// @Tags taint-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /taints/templates/{id} [delete]
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

	// 检查用户权限：只有 admin 和 user 角色可以删除污点模板
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can delete taint templates",
		})
		return
	}

	if err := h.taintSvc.DeleteTemplate(uint(id), userID.(uint)); err != nil {
		h.logger.Error("Failed to delete taint template: %v", err)
		if err.Error() == "template not found" {
			c.JSON(http.StatusNotFound, Response{
				Code:    http.StatusNotFound,
				Message: "Template not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, Response{
				Code:    http.StatusInternalServerError,
				Message: "Failed to delete taint template: " + err.Error(),
			})
		}
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Taint template deleted successfully",
	})
}

// ListTemplates 获取污点模板列表
// @Summary 获取污点模板列表
// @Description 获取污点模板列表，支持分页和筛选
// @Tags taint-templates
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param name query string false "模板名称筛选"
// @Param search query string false "搜索关键词（名称、描述、污点Key）"
// @Param effect query string false "按效果筛选（NoSchedule|PreferNoSchedule|NoExecute）"
// @Success 200 {object} Response
// @Router /taints/templates [get]
func (h *Handler) ListTemplates(c *gin.Context) {
	var req taint.TemplateListRequest

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
	req.Search = c.Query("search")
	req.Effect = c.Query("effect")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, Response{
			Code:    http.StatusUnauthorized,
			Message: "User not authenticated",
		})
		return
	}

	result, err := h.taintSvc.ListTemplates(req, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to list taint templates: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to list taint templates: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    result,
	})
}

// GetTemplate 获取单个污点模板详情
// @Summary 获取污点模板详情
// @Description 根据模板ID获取污点模板的详细信息
// @Tags taint-templates
// @Produce json
// @Param id path int true "模板ID"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 404 {object} Response
// @Router /taints/templates/{id} [get]
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
	listReq := taint.TemplateListRequest{
		Page:     1,
		PageSize: 100,
	}

	result, err := h.taintSvc.ListTemplates(listReq, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get taint template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get taint template: " + err.Error(),
		})
		return
	}

	var template *taint.TemplateInfo
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

// ApplyTemplate 应用污点模板
// @Summary 应用污点模板
// @Description 将污点模板应用到指定节点
// @Tags taint-templates
// @Accept json
// @Produce json
// @Param request body taint.ApplyTemplateRequest true "模板应用请求"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/templates/apply [post]
func (h *Handler) ApplyTemplate(c *gin.Context) {
	var req taint.ApplyTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Errorf("Failed to bind apply template request: %v", err)
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

	// 检查用户权限：只有 admin 和 user 角色可以应用污点模板到节点
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin && userRole != model.RoleUser {
		c.JSON(http.StatusForbidden, Response{
			Code:    http.StatusForbidden,
			Message: "Insufficient permissions. Only admin and user roles can apply taint templates to nodes",
		})
		return
	}

	if err := h.taintSvc.ApplyTemplate(req, userID.(uint)); err != nil {
		h.logger.Errorf("Failed to apply taint template: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to apply taint template: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Taint template applied successfully",
	})
}

// GetTaintUsage 获取污点使用情况
// @Summary 获取污点使用情况
// @Description 获取集群中污点的使用统计信息
// @Tags taints
// @Produce json
// @Param cluster_name query string true "集群名称"
// @Success 200 {object} Response
// @Failure 400 {object} Response
// @Failure 500 {object} Response
// @Router /taints/usage [get]
func (h *Handler) GetTaintUsage(c *gin.Context) {
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

	usage, err := h.taintSvc.GetTaintUsage(clusterName, userID.(uint))
	if err != nil {
		h.logger.Error("Failed to get taint usage: %v", err)
		c.JSON(http.StatusInternalServerError, Response{
			Code:    http.StatusInternalServerError,
			Message: "Failed to get taint usage: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Response{
		Code:    http.StatusOK,
		Message: "Success",
		Data:    usage,
	})
}
