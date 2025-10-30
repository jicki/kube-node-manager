package websocket

import (
	"net/http"
	"time"

	"kube-node-manager/internal/websocket"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
	gorillaws "github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	upgrader = gorillaws.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			// 允许所有来源，生产环境应根据实际需求进行更严格的检查
			return true
		},
	}
)

// Handler WebSocket处理器
type Handler struct {
	hub    *websocket.Hub
	logger *logger.Logger
}

// NewHandler 创建新的WebSocket处理器
func NewHandler(hub *websocket.Hub, logger *logger.Logger) *Handler {
	return &Handler{
		hub:    hub,
		logger: logger,
	}
}

// HandleWebSocket 处理WebSocket连接请求
func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 从URL参数获取集群名称
	clusterName := c.Query("cluster")
	if clusterName == "" {
		h.logger.Warning("WebSocket connection attempt without cluster parameter")
		c.JSON(400, gin.H{"error": "cluster parameter is required"})
		return
	}

	// 可选：从请求中获取用户认证信息
	// userID := c.GetString("user_id") // 从认证中间件获取

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Errorf("Failed to upgrade to WebSocket: %v", err)
		return
	}

	client := websocket.NewClient(h.hub, conn, clusterName)
	h.hub.Register(client)

	h.logger.Infof("WebSocket client connected: cluster=%s, clientID=%s", clusterName, client.ID)

	// 启动读写协程
	go client.WritePump()
	go client.ReadPump()
}
