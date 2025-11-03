package ansible

import (
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/pkg/logger"
	"net/http"

	"github.com/gin-gonic/gin"
)

// QueueHandler 处理任务队列相关的请求
type QueueHandler struct {
	service *ansible.Service
	logger  *logger.Logger
}

// NewQueueHandler 创建 QueueHandler 实例
func NewQueueHandler(service *ansible.Service, logger *logger.Logger) *QueueHandler {
	return &QueueHandler{
		service: service,
		logger:  logger,
	}
}

// GetQueueStats 获取队列统计信息
// @Summary 获取任务队列统计信息
// @Tags Ansible
// @Success 200 {object} ansible.QueueStats
// @Router /api/v1/ansible/queue/stats [get]
func (h *QueueHandler) GetQueueStats(c *gin.Context) {
	queueSvc := h.service.GetQueueService()
	stats, err := queueSvc.GetQueueStats()
	if err != nil {
		h.logger.Errorf("Failed to get queue stats: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 转换统计数据为友好的响应格式
	response := gin.H{
		"code": 200,
		"data": gin.H{
			"total_pending":   stats.TotalPending,
			"total_running":   stats.TotalRunning,
			"by_priority":     stats.ByPriority,
			"by_user":         stats.ByUser,
			"avg_wait_seconds": int(stats.AvgWaitDuration.Seconds()),
			"max_wait_seconds": int(stats.MaxWaitDuration.Seconds()),
			"max_wait_task_id": stats.MaxWaitTaskID,
		},
	}

	c.JSON(http.StatusOK, response)
}

