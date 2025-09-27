package progress

import (
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
)

// Handler 进度推送处理器
type Handler struct {
	progressSvc *progress.Service
	logger      *logger.Logger
}

// NewHandler 创建进度推送处理器实例
func NewHandler(progressSvc *progress.Service, logger *logger.Logger) *Handler {
	return &Handler{
		progressSvc: progressSvc,
		logger:      logger,
	}
}

// HandleWebSocket 处理WebSocket连接
func (h *Handler) HandleWebSocket(c *gin.Context) {
	h.progressSvc.HandleWebSocket(c)
}
