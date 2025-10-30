package ansible

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// InventoryHandler 主机清单 Handler
type InventoryHandler struct {
	service *ansible.InventoryService
	logger  *logger.Logger
}

// NewInventoryHandler 创建主机清单 Handler 实例
func NewInventoryHandler(service *ansible.InventoryService, logger *logger.Logger) *InventoryHandler {
	return &InventoryHandler{
		service: service,
		logger:  logger,
	}
}

// ListInventories 列出主机清单
// @Summary 列出主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Param source_type query string false "来源类型"
// @Param cluster_id query int false "集群ID"
// @Param keyword query string false "关键字"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories [get]
func (h *InventoryHandler) ListInventories(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.InventoryListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	inventories, total, err := h.service.ListInventories(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to list inventories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":  0,
		"data":  inventories,
		"total": total,
	})
}

// GetInventory 获取主机清单详情
// @Summary 获取主机清单详情
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param id path int true "主机清单ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories/{id} [get]
func (h *InventoryHandler) GetInventory(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory id"})
		return
	}

	inventory, err := h.service.GetInventory(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get inventory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"data": inventory,
	})
}

// CreateInventory 创建主机清单
// @Summary 创建主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param inventory body model.InventoryCreateRequest true "主机清单信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories [post]
func (h *InventoryHandler) CreateInventory(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.InventoryCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	inventory, err := h.service.CreateInventory(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to create inventory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    inventory,
		"message": "Inventory created successfully",
	})
}

// UpdateInventory 更新主机清单
// @Summary 更新主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param id path int true "主机清单ID"
// @Param inventory body model.InventoryUpdateRequest true "主机清单信息"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories/{id} [put]
func (h *InventoryHandler) UpdateInventory(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory id"})
		return
	}

	var req model.InventoryUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	inventory, err := h.service.UpdateInventory(uint(id), req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to update inventory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    inventory,
		"message": "Inventory updated successfully",
	})
}

// DeleteInventory 删除主机清单
// @Summary 删除主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param id path int true "主机清单ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories/{id} [delete]
func (h *InventoryHandler) DeleteInventory(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory id"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	if err := h.service.DeleteInventory(uint(id), userID.(uint)); err != nil {
		h.logger.Errorf("Failed to delete inventory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "Inventory deleted successfully",
	})
}

// GenerateFromCluster 从集群生成主机清单
// @Summary 从集群生成主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param request body model.GenerateInventoryRequest true "生成请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories/generate [post]
func (h *InventoryHandler) GenerateFromCluster(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	var req model.GenerateInventoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	inventory, err := h.service.GenerateFromK8s(req, userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to generate inventory from cluster: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    inventory,
		"message": "Inventory generated from cluster successfully",
	})
}

// RefreshInventory 刷新 K8s 来源的主机清单
// @Summary 刷新主机清单
// @Tags Ansible Inventories
// @Accept json
// @Produce json
// @Param id path int true "主机清单ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/inventories/{id}/refresh [post]
func (h *InventoryHandler) RefreshInventory(c *gin.Context) {
	if !checkAdminPermission(c) {
		return
	}

	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory id"})
		return
	}

	// 获取用户ID
	userID, _ := c.Get("user_id")

	inventory, err := h.service.RefreshK8sInventory(uint(id), userID.(uint))
	if err != nil {
		h.logger.Errorf("Failed to refresh inventory: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"data":    inventory,
		"message": "Inventory refreshed successfully",
	})
}

