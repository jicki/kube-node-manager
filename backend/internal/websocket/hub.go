package websocket

import (
	"encoding/json"
	"sync"
	"time"

	"kube-node-manager/internal/informer"
	"kube-node-manager/pkg/logger"

	"github.com/gorilla/websocket"
)

// Message WebSocket 消息
type Message struct {
	Type      string      `json:"type"`       // 消息类型：node_add, node_update, node_delete, ping, pong
	ClusterName string    `json:"cluster_name,omitempty"` // 集群名称
	NodeName  string      `json:"node_name,omitempty"`    // 节点名称
	Data      interface{} `json:"data,omitempty"`         // 消息数据
	Timestamp time.Time   `json:"timestamp"`              // 时间戳
	Changes   []string    `json:"changes,omitempty"`      // 变化的字段
}

// Client WebSocket 客户端
type Client struct {
	ID          string          // 客户端 ID
	Conn        *websocket.Conn // WebSocket 连接
	Send        chan Message    // 发送消息通道
	Hub         *Hub            // 所属的 Hub
	Clusters    map[string]bool // 订阅的集群列表
	mu          sync.RWMutex    // 保护 Clusters
	LastPing    time.Time       // 最后 ping 时间
	UserID      uint            // 用户 ID
}

// Hub WebSocket 中心，管理所有客户端连接和消息广播
type Hub struct {
	// 注册的客户端
	clients sync.Map // clientID -> *Client

	// 集群订阅：cluster -> map[clientID]bool
	subscriptions sync.Map

	// 任务订阅：taskID -> map[userID]chan interface{}
	taskSubscriptions sync.Map

	// 注册通道
	register chan *Client

	// 注销通道
	unregister chan *Client

	// 广播消息通道
	broadcast chan Message

	logger *logger.Logger

	// 心跳间隔
	pingInterval time.Duration
	pongTimeout  time.Duration
}

// NewHub 创建新的 Hub
func NewHub(logger *logger.Logger) *Hub {
	return &Hub{
		register:     make(chan *Client, 256),
		unregister:   make(chan *Client, 256),
		broadcast:    make(chan Message, 1024),
		logger:       logger,
		pingInterval: 30 * time.Second,
		pongTimeout:  60 * time.Second,
	}
}

// NewClient 创建新的客户端
func NewClient(hub *Hub, conn *websocket.Conn, clusterName string) *Client {
	client := &Client{
		ID:       generateClientID(),
		Conn:     conn,
		Send:     make(chan Message, 256),
		Hub:      hub,
		Clusters: make(map[string]bool),
		LastPing: time.Now(),
	}
	
	// 订阅指定的集群
	if clusterName != "" {
		client.Clusters[clusterName] = true
	}
	
	return client
}

// generateClientID 生成唯一的客户端 ID
func generateClientID() string {
	return time.Now().Format("20060102150405.000000")
}

// Run 启动 Hub
func (h *Hub) Run() {
	h.logger.Info("WebSocket Hub started")

	// 启动心跳检测
	go h.pingClients()

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)
		}
	}
}

// Register 注册客户端
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister 注销客户端
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast 广播消息
func (h *Hub) Broadcast(message Message) {
	h.broadcast <- message
}

// OnNodeEvent 实现 NodeEventHandler 接口，接收 Informer 事件并转换为 WebSocket 消息
func (h *Hub) OnNodeEvent(event informer.NodeEvent) {
	var messageType string
	switch event.Type {
	case informer.EventTypeAdd:
		messageType = "node_add"
	case informer.EventTypeUpdate:
		messageType = "node_update"
	case informer.EventTypeDelete:
		messageType = "node_delete"
	default:
		return
	}

	// 构造消息
	message := Message{
		Type:        messageType,
		ClusterName: event.ClusterName,
		NodeName:    event.Node.Name,
		Data:        event.Node, // 发送完整的节点对象
		Timestamp:   event.Timestamp,
		Changes:     event.Changes,
	}

	// 广播消息
	h.Broadcast(message)
}

// registerClient 注册客户端
func (h *Hub) registerClient(client *Client) {
	h.clients.Store(client.ID, client)
	h.logger.Infof("WebSocket client registered: %s (UserID: %d)", client.ID, client.UserID)

	// 订阅客户端关注的集群
	client.mu.RLock()
	for clusterName := range client.Clusters {
		h.subscribeCluster(clusterName, client.ID)
	}
	client.mu.RUnlock()

	// 发送欢迎消息
	welcome := Message{
		Type:      "connected",
		Data:      map[string]string{"message": "WebSocket connected successfully"},
		Timestamp: time.Now(),
	}
	client.Send <- welcome
}

// unregisterClient 注销客户端
func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients.LoadAndDelete(client.ID); ok {
		// 取消所有集群订阅
		client.mu.RLock()
		for clusterName := range client.Clusters {
			h.unsubscribeCluster(clusterName, client.ID)
		}
		client.mu.RUnlock()

		close(client.Send)
		h.logger.Infof("WebSocket client unregistered: %s", client.ID)
	}
}

// broadcastMessage 广播消息到所有订阅了该集群的客户端
func (h *Hub) broadcastMessage(message Message) {
	// 如果没有指定集群，广播给所有客户端
	if message.ClusterName == "" {
		h.clients.Range(func(_, value interface{}) bool {
			client := value.(*Client)
			select {
			case client.Send <- message:
			default:
				// 发送失败，客户端可能已断开，注销它
				h.Unregister(client)
			}
			return true
		})
		return
	}

	// 广播给订阅了该集群的客户端
	if subsInterface, ok := h.subscriptions.Load(message.ClusterName); ok {
		subs := subsInterface.(*sync.Map)
		subs.Range(func(key, _ interface{}) bool {
			clientID := key.(string)
			if clientInterface, ok := h.clients.Load(clientID); ok {
				client := clientInterface.(*Client)
				select {
				case client.Send <- message:
				default:
					// 发送失败，客户端可能已断开，注销它
					h.Unregister(client)
				}
			}
			return true
		})
	}
}

// subscribeCluster 订阅集群
func (h *Hub) subscribeCluster(clusterName, clientID string) {
	subsInterface, _ := h.subscriptions.LoadOrStore(clusterName, &sync.Map{})
	subs := subsInterface.(*sync.Map)
	subs.Store(clientID, true)
	h.logger.Infof("Client %s subscribed to cluster %s", clientID, clusterName)
}

// unsubscribeCluster 取消订阅集群
func (h *Hub) unsubscribeCluster(clusterName, clientID string) {
	if subsInterface, ok := h.subscriptions.Load(clusterName); ok {
		subs := subsInterface.(*sync.Map)
		subs.Delete(clientID)
		h.logger.Infof("Client %s unsubscribed from cluster %s", clientID, clusterName)
	}
}

// pingClients 定期向所有客户端发送 ping 消息
func (h *Hub) pingClients() {
	ticker := time.NewTicker(h.pingInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		h.clients.Range(func(_, value interface{}) bool {
			client := value.(*Client)

			// 检查是否超时
			if now.Sub(client.LastPing) > h.pongTimeout {
				h.logger.Warningf("Client %s ping timeout, disconnecting", client.ID)
				h.Unregister(client)
				return true
			}

			// 发送 ping
			ping := Message{
				Type:      "ping",
				Timestamp: now,
			}

			select {
			case client.Send <- ping:
			default:
				h.Unregister(client)
			}

			return true
		})
	}
}

// GetStats 获取 Hub 统计信息
func (h *Hub) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	// 统计客户端数量
	clientCount := 0
	h.clients.Range(func(_, _ interface{}) bool {
		clientCount++
		return true
	})
	stats["client_count"] = clientCount

	// 统计订阅数量
	subscriptionCount := 0
	h.subscriptions.Range(func(key, value interface{}) bool {
		subs := value.(*sync.Map)
		count := 0
		subs.Range(func(_, _ interface{}) bool {
			count++
			return true
		})
		subscriptionCount += count
		return true
	})
	stats["subscription_count"] = subscriptionCount

	return stats
}

// ReadPump 从客户端读取消息
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(c.Hub.pongTimeout))
	c.Conn.SetPongHandler(func(string) error {
		c.LastPing = time.Now()
		c.Conn.SetReadDeadline(time.Now().Add(c.Hub.pongTimeout))
		return nil
	})

	for {
		var msg Message
		err := c.Conn.ReadJSON(&msg)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Hub.logger.Errorf("WebSocket read error: %v", err)
			}
			break
		}

		// 处理客户端消息
		c.handleMessage(msg)
	}
}

// WritePump 向客户端发送消息
func (c *Client) WritePump() {
	ticker := time.NewTicker(c.Hub.pingInterval)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 通道已关闭
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 发送消息
			if err := c.Conn.WriteJSON(message); err != nil {
				c.Hub.logger.Errorf("WebSocket write error: %v", err)
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage 处理客户端消息
func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case "subscribe":
		// 订阅集群
		if clusterName, ok := msg.Data.(string); ok {
			c.mu.Lock()
			c.Clusters[clusterName] = true
			c.mu.Unlock()
			c.Hub.subscribeCluster(clusterName, c.ID)
			c.Hub.logger.Infof("Client %s subscribed to cluster %s", c.ID, clusterName)
		}

	case "unsubscribe":
		// 取消订阅集群
		if clusterName, ok := msg.Data.(string); ok {
			c.mu.Lock()
			delete(c.Clusters, clusterName)
			c.mu.Unlock()
			c.Hub.unsubscribeCluster(clusterName, c.ID)
			c.Hub.logger.Infof("Client %s unsubscribed from cluster %s", c.ID, clusterName)
		}

	case "pong":
		// 收到 pong 响应
		c.LastPing = time.Now()

	default:
		c.Hub.logger.Warningf("Unknown message type from client %s: %s", c.ID, msg.Type)
	}
}

// SendNodeUpdate 向订阅了指定集群的客户端发送节点更新
func (h *Hub) SendNodeUpdate(clusterName, nodeName string, changes []string, data interface{}) {
	message := Message{
		Type:        "node_update",
		ClusterName: clusterName,
		NodeName:    nodeName,
		Changes:     changes,
		Data:        data,
		Timestamp:   time.Now(),
	}
	h.Broadcast(message)
}

// SendJSON 向客户端发送 JSON 消息（辅助方法）
func SendJSON(conn *websocket.Conn, v interface{}) error {
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.WriteJSON(v)
}

// SendError 向客户端发送错误消息
func SendError(conn *websocket.Conn, message string) error {
	errorMsg := Message{
		Type: "error",
		Data: map[string]string{
			"message": message,
		},
		Timestamp: time.Now(),
	}
	return SendJSON(conn, errorMsg)
}

// MarshalMessage 将消息序列化为 JSON
func MarshalMessage(msg Message) ([]byte, error) {
	return json.Marshal(msg)
}

// UnmarshalMessage 将 JSON 反序列化为消息
func UnmarshalMessage(data []byte) (*Message, error) {
	var msg Message
	err := json.Unmarshal(data, &msg)
	return &msg, err
}

// RegisterTaskClient 注册任务日志客户端
func (h *Hub) RegisterTaskClient(taskID uint, userID uint, messageChan chan interface{}) {
	// 获取或创建任务订阅映射
	subsInterface, _ := h.taskSubscriptions.LoadOrStore(taskID, &sync.Map{})
	subs := subsInterface.(*sync.Map)
	
	// 添加用户订阅
	subs.Store(userID, messageChan)
	
	h.logger.Infof("Registered task client: taskID=%d, userID=%d", taskID, userID)
}

// UnregisterTaskClient 注销任务日志客户端
func (h *Hub) UnregisterTaskClient(taskID uint, userID uint) {
	// 获取任务订阅映射
	if subsInterface, ok := h.taskSubscriptions.Load(taskID); ok {
		subs := subsInterface.(*sync.Map)
		
		// 删除用户订阅
		subs.Delete(userID)
		
		h.logger.Infof("Unregistered task client: taskID=%d, userID=%d", taskID, userID)
		
		// 检查是否还有订阅者，如果没有则删除任务订阅映射
		hasSubscribers := false
		subs.Range(func(_, _ interface{}) bool {
			hasSubscribers = true
			return false // 只需检查是否有至少一个订阅者
		})
		
		if !hasSubscribers {
			h.taskSubscriptions.Delete(taskID)
			h.logger.Infof("Removed task subscription mapping for taskID=%d (no more subscribers)", taskID)
		}
	}
}

// BroadcastToTask 向任务的所有订阅者广播消息
func (h *Hub) BroadcastToTask(taskID uint, message interface{}) {
	// 获取任务订阅映射
	if subsInterface, ok := h.taskSubscriptions.Load(taskID); ok {
		subs := subsInterface.(*sync.Map)
		
		// 向所有订阅者发送消息
		subs.Range(func(key, value interface{}) bool {
			userID := key.(uint)
			messageChan := value.(chan interface{})
			
			select {
			case messageChan <- message:
				// 消息发送成功
			default:
				// 通道已满，记录警告
				h.logger.Warningf("Task message channel full for userID=%d, taskID=%d", userID, taskID)
			}
			
			return true
		})
	}
}

