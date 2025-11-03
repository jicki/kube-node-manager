package ansible

import (
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

// VisualizationHandler 处理任务执行可视化相关的请求
type VisualizationHandler struct {
	service *ansible.Service
	logger  *logger.Logger
}

// NewVisualizationHandler 创建 VisualizationHandler 实例
func NewVisualizationHandler(service *ansible.Service, logger *logger.Logger) *VisualizationHandler {
	return &VisualizationHandler{
		service: service,
		logger:  logger,
	}
}

// GetTaskVisualization 获取任务执行可视化数据
// @Summary 获取任务执行可视化数据
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200 {object} model.TaskExecutionVisualization
// @Router /api/v1/ansible/tasks/{id}/visualization [get]
func (h *VisualizationHandler) GetTaskVisualization(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	viz, err := h.service.GetVisualizationService().GetTaskVisualization(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get task visualization: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": viz,
	})
}

// GetTaskTimelineSummary 获取任务时间线摘要
// @Summary 获取任务时间线摘要
// @Tags Ansible
// @Param id path int true "任务ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ansible/tasks/{id}/timeline-summary [get]
func (h *VisualizationHandler) GetTaskTimelineSummary(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	summary, err := h.service.GetVisualizationService().GetTaskTimelineSummary(uint(id))
	if err != nil {
		h.logger.Errorf("Failed to get timeline summary: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"data": summary,
	})
}

