package ansible

import (
	"kube-node-manager/pkg/logger"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// WebSocketHandler WebSocket Handler
type WebSocketHandler struct {
	wsHub   interface{}
	logger  *logger.Logger
	upgrader websocket.Upgrader
}

// NewWebSocketHandler 创建 WebSocket Handler 实例
func NewWebSocketHandler(wsHub interface{}, logger *logger.Logger) *WebSocketHandler {
	return &WebSocketHandler{
		wsHub:  wsHub,
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许所有来源（生产环境应该更严格）
			},
		},
	}
}

// HandleTaskLogStream 处理任务日志流
// @Summary 任务日志 WebSocket 流
// @Tags Ansible WebSocket
// @Accept json
// @Produce json
// @Param id path int true "任务ID"
// @Param token query string true "认证Token"
// @Router /api/v1/ansible/tasks/{id}/ws [get]
func (h *WebSocketHandler) HandleTaskLogStream(c *gin.Context) {
	// 获取任务ID
	taskID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid task id"})
		return
	}

	// 获取用户ID（从认证中间件或 context）
	// WebSocket 连接时，认证中间件可能已经设置了 user_id
	userID, exists := c.Get("user_id")
	if !exists {
		// 如果中间件没有设置，返回未授权错误
		h.logger.Warningf("WebSocket connection attempt without authentication for task %d", taskID)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized - missing authentication"})
		return
	}

	// 升级到 WebSocket
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Failed to upgrade to websocket: %v", err)
		return
	}
	defer conn.Close()

	h.logger.Infof("WebSocket connection established for task %d by user %v", taskID, userID)

	// 设置读超时和写超时
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// 设置 pong 处理器
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	// 启动 ping 定时器
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	// 创建消息通道
	messageChan := make(chan interface{}, 100)
	defer close(messageChan)

	// 注册到 WebSocket Hub（如果有）
	if h.wsHub != nil {
		type WSHub interface {
			RegisterTaskClient(taskID uint, userID uint, messageChan chan interface{})
			UnregisterTaskClient(taskID uint, userID uint)
		}

		if hub, ok := h.wsHub.(WSHub); ok {
			hub.RegisterTaskClient(uint(taskID), userID.(uint), messageChan)
			defer hub.UnregisterTaskClient(uint(taskID), userID.(uint))
		}
	}

	// 发送连接成功消息
	if err := conn.WriteJSON(map[string]interface{}{
		"type":    "connected",
		"message": "WebSocket connection established",
		"task_id": taskID,
	}); err != nil {
		h.logger.Errorf("Failed to send connection message: %v", err)
		return
	}

	// 处理消息循环
	done := make(chan struct{})
	go h.readPump(conn, done)

	for {
		select {
		case <-done:
			// 连接关闭
			h.logger.Infof("WebSocket connection closed for task %d", taskID)
			return

		case message := <-messageChan:
			// 发送日志消息
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteJSON(message); err != nil {
				h.logger.Errorf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			// 发送 ping
			conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				h.logger.Errorf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// readPump 读取客户端消息（主要用于保持连接和处理 pong）
func (h *WebSocketHandler) readPump(conn *websocket.Conn, done chan struct{}) {
	defer close(done)

	for {
		_, _, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Errorf("Unexpected websocket close: %v", err)
			}
			return
		}
	}
}

