package ansible

import (
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// EstimationHandler 任务预估处理器
type EstimationHandler struct {
	service *ansible.Service
	logger  *logger.Logger
}

// NewEstimationHandler 创建 EstimationHandler 实例
func NewEstimationHandler(service *ansible.Service, logger *logger.Logger) *EstimationHandler {
	return &EstimationHandler{
		service: service,
		logger:  logger,
	}
}

// EstimateByTemplate 基于模板预估任务执行时间
// @Summary 基于模板预估任务执行时间
// @Tags Ansible
// @Param template_id query int true "模板ID"
// @Success 200 {object} ansible.TaskEstimation
// @Router /api/v1/ansible/estimate/template [get]
func (h *EstimationHandler) EstimateByTemplate(c *gin.Context) {
	templateIDStr := c.Query("template_id")
	if templateIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template_id is required"})
		return
	}

	templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template_id"})
		return
	}

	estimationSvc := h.service.GetEstimationService()
	estimation, err := estimationSvc.EstimateByTemplate(uint(templateID))
	if err != nil {
		h.logger.Infof("Estimation failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无历史数据",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "预估成功",
		"data":    estimation,
	})
}

// EstimateByInventory 基于清单预估任务执行时间
// @Summary 基于清单预估任务执行时间
// @Tags Ansible
// @Param inventory_id query int true "清单ID"
// @Success 200 {object} ansible.TaskEstimation
// @Router /api/v1/ansible/estimate/inventory [get]
func (h *EstimationHandler) EstimateByInventory(c *gin.Context) {
	inventoryIDStr := c.Query("inventory_id")
	if inventoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "inventory_id is required"})
		return
	}

	inventoryID, err := strconv.ParseUint(inventoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory_id"})
		return
	}

	estimationSvc := h.service.GetEstimationService()
	estimation, err := estimationSvc.EstimateByInventory(uint(inventoryID))
	if err != nil {
		h.logger.Infof("Estimation failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无历史数据",
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "预估成功",
		"data":    estimation,
	})
}

// EstimateByTemplateAndInventory 基于模板和清单组合预估
// @Summary 基于模板和清单组合预估任务执行时间
// @Tags Ansible
// @Param template_id query int true "模板ID"
// @Param inventory_id query int true "清单ID"
// @Success 200 {object} ansible.TaskEstimation
// @Router /api/v1/ansible/estimate/combined [get]
func (h *EstimationHandler) EstimateByTemplateAndInventory(c *gin.Context) {
	templateIDStr := c.Query("template_id")
	inventoryIDStr := c.Query("inventory_id")

	if templateIDStr == "" || inventoryIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "template_id and inventory_id are required"})
		return
	}

	templateID, err := strconv.ParseUint(templateIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid template_id"})
		return
	}

	inventoryID, err := strconv.ParseUint(inventoryIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid inventory_id"})
		return
	}

	estimationSvc := h.service.GetEstimationService()
	estimation, err := estimationSvc.EstimateByTemplateAndInventory(uint(templateID), uint(inventoryID))
	if err != nil {
		h.logger.Infof("Estimation failed: %v", err)
		c.JSON(http.StatusOK, gin.H{
			"code":    200,
			"message": "无历史数据",
			"data":    nil,
		})
		return
	}

	summary := estimationSvc.GetEstimationSummary(estimation)

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "预估成功",
		"data":    estimation,
		"summary": summary,
	})
}

