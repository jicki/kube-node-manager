package progress

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"kube-node-manager/internal/service/auth"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // 允许跨域连接，生产环境应该更严格
	},
}

// ProgressMessage 进度消息结构
type ProgressMessage struct {
	TaskID      string    `json:"task_id"`
	Type        string    `json:"type"`            // progress, complete, error
	Action      string    `json:"action"`          // batch_label, batch_taint
	Current     int       `json:"current"`         // 当前完成数量
	Total       int       `json:"total"`           // 总数量
	Progress    float64   `json:"progress"`        // 进度百分比 (0-100)
	CurrentNode string    `json:"current_node"`    // 当前处理的节点
	Message     string    `json:"message"`         // 消息内容
	Error       string    `json:"error,omitempty"` // 错误信息
	Timestamp   time.Time `json:"timestamp"`
}

// TaskProgress 任务进度
type TaskProgress struct {
	TaskID    string
	Action    string
	Current   int
	Total     int
	IsRunning bool
	Error     error
}

// Connection WebSocket连接
type Connection struct {
	ws       *websocket.Conn
	send     chan ProgressMessage
	userID   uint
	lastSeen time.Time // 添加最后活跃时间
}

// TokenValidator JWT token验证接口
type TokenValidator interface {
	ValidateToken(tokenString string) (*auth.Claims, error)
}

// Service 进度推送服务
type Service struct {
	// 存储用户连接
	connections map[uint]*Connection
	// 存储任务进度
	tasks map[string]*TaskProgress
	// 保护连接映射
	connMutex sync.RWMutex
	// 保护任务映射
	taskMutex   sync.RWMutex
	logger      *logger.Logger
	authService TokenValidator
}

// NewService 创建进度推送服务
func NewService(logger *logger.Logger) *Service {
	s := &Service{
		connections: make(map[uint]*Connection),
		tasks:       make(map[string]*TaskProgress),
		logger:      logger,
	}

	// 启动定期清理goroutine
	go s.cleanupStaleConnections()

	return s
}

// SetAuthService 设置认证服务
func (s *Service) SetAuthService(authService TokenValidator) {
	s.authService = authService
}

// HandleWebSocket 处理WebSocket连接
func (s *Service) HandleWebSocket(c *gin.Context) {
	s.logger.Infof("WebSocket connection attempt from %s", c.ClientIP())

	// 从查询参数获取token
	token := c.Query("token")
	if token == "" {
		s.logger.Errorf("WebSocket connection failed: no token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证token"})
		return
	}

	s.logger.Infof("WebSocket token received (length: %d)", len(token))

	// 验证token
	if s.authService == nil {
		s.logger.Errorf("Auth service not set for WebSocket authentication")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "认证服务未配置"})
		return
	}

	s.logger.Infof("Validating WebSocket token...")
	claims, err := s.authService.ValidateToken(token)
	if err != nil {
		s.logger.Errorf("WebSocket token validation failed: %v", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}

	if claims.Type != "access" {
		s.logger.Errorf("Invalid token type for WebSocket: %s", claims.Type)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token type"})
		return
	}

	userID := claims.UserID
	s.logger.Infof("WebSocket authentication successful for user %d", userID)

	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Errorf("Failed to upgrade websocket: %v", err)
		return
	}
	defer ws.Close()

	// 创建连接
	conn := &Connection{
		ws:       ws,
		send:     make(chan ProgressMessage, 512), // 增加缓冲区大小
		userID:   userID,
		lastSeen: time.Now(),
	}

	// 注册连接（关闭已存在的连接）
	s.connMutex.Lock()
	if existingConn, exists := s.connections[userID]; exists {
		s.logger.Infof("Closing existing WebSocket connection for user %d", userID)
		close(existingConn.send)
		existingConn.ws.Close()
	}
	s.connections[userID] = conn
	s.connMutex.Unlock()

	s.logger.Infof("WebSocket connected for user %d", userID)

	// 启动消息发送goroutine
	go s.writePump(conn)

	// 启动消息接收goroutine（处理心跳等）
	go s.readPump(conn)

	// 发送连接确认消息
	s.sendToUser(userID, ProgressMessage{
		Type:      "connected",
		Message:   "WebSocket连接成功",
		Timestamp: time.Now(),
	})

	// 检查是否有正在进行或刚完成的任务，发送状态更新
	s.sendCurrentTaskStatus(userID)

	// 等待连接关闭
	select {}
}

// writePump 发送消息到客户端
func (s *Service) writePump(conn *Connection) {
	ticker := time.NewTicker(25 * time.Second) // 进一步缩短心跳间隔
	defer func() {
		ticker.Stop()
		conn.ws.Close()
		s.removeConnection(conn.userID)
		s.logger.Infof("WritePump closed for user %d", conn.userID)
	}()

	for {
		select {
		case message, ok := <-conn.send:
			conn.ws.SetWriteDeadline(time.Now().Add(15 * time.Second)) // 增加写入超时时间
			if !ok {
				s.logger.Infof("Send channel closed for user %d, sending close message", conn.userID)
				conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.ws.WriteJSON(message); err != nil {
				s.logger.Errorf("Failed to write message to user %d: %v", conn.userID, err)
				return
			}

			// 更新活跃时间
			conn.lastSeen = time.Now()

			// 添加发送成功日志（仅对重要消息）
			if message.Type == "complete" || message.Type == "error" {
				s.logger.Infof("Successfully sent %s message to user %d for task %s", message.Type, conn.userID, message.TaskID)
			}

		case <-ticker.C:
			conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Errorf("Failed to send ping to user %d: %v", conn.userID, err)
				return
			}
			s.logger.Infof("Sent ping to user %d", conn.userID) // 添加心跳日志用于调试
		}
	}
}

// readPump 接收客户端消息
func (s *Service) readPump(conn *Connection) {
	defer func() {
		conn.ws.Close()
		s.removeConnection(conn.userID)
		s.logger.Infof("ReadPump closed for user %d", conn.userID)
	}()

	conn.ws.SetReadLimit(1024)
	conn.ws.SetReadDeadline(time.Now().Add(70 * time.Second)) // 增加读取超时时间
	conn.ws.SetPongHandler(func(string) error {
		conn.lastSeen = time.Now() // 更新活跃时间
		conn.ws.SetReadDeadline(time.Now().Add(70 * time.Second))
		return nil
	})

	for {
		messageType, message, err := conn.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure) {
				s.logger.Errorf("WebSocket error for user %d: %v", conn.userID, err)
			} else {
				s.logger.Infof("WebSocket connection closed for user %d: %v", conn.userID, err)
			}
			break
		}

		// 更新活跃时间
		conn.lastSeen = time.Now()

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			s.logger.Infof("Received text message from user %d: %s", conn.userID, string(message))
		case websocket.BinaryMessage:
			s.logger.Infof("Received binary message from user %d", conn.userID)
		case websocket.PingMessage:
			// 响应ping消息
			conn.ws.WriteMessage(websocket.PongMessage, nil)
		}
	}
}

// removeConnection 移除连接
func (s *Service) removeConnection(userID uint) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if conn, exists := s.connections[userID]; exists {
		close(conn.send)
		delete(s.connections, userID)
		s.logger.Infof("WebSocket disconnected for user %d", userID)
	}
}

// sendToUser 发送消息给指定用户
func (s *Service) sendToUser(userID uint, message ProgressMessage) {
	s.connMutex.RLock()
	conn, exists := s.connections[userID]
	s.connMutex.RUnlock()

	if !exists {
		// 只对重要消息记录警告
		if message.Type == "complete" || message.Type == "error" {
			s.logger.Warningf("No WebSocket connection found for user %d (message: %s)", userID, message.Type)
		}
		return
	}

	select {
	case conn.send <- message:
		// 消息发送成功
		if message.Type == "complete" || message.Type == "error" {
			s.logger.Infof("Queued %s message for user %d, task %s", message.Type, userID, message.TaskID)
		}
	case <-time.After(1 * time.Second): // 快速超时，避免阻塞
		s.logger.Warningf("Send queue timeout for user %d (type: %s)", userID, message.Type)
		// 对于重要消息，尝试一次非阻塞发送
		if message.Type == "complete" || message.Type == "error" {
			select {
			case conn.send <- message:
				s.logger.Infof("Retry successful for user %d", userID)
			default:
				s.logger.Warningf("Send queue full for user %d, may need to reconnect", userID)
				// 不立即移除连接，让心跳机制处理
			}
		}
	}
}

// CreateTask 创建新任务
func (s *Service) CreateTask(taskID, action string, total int) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	// 检查是否已存在相同的任务，如果存在则清理
	if existingTask, exists := s.tasks[taskID]; exists {
		s.logger.Infof("Replacing existing task %s (action: %s)", taskID, existingTask.Action)
		existingTask.IsRunning = false
	}

	s.tasks[taskID] = &TaskProgress{
		TaskID:    taskID,
		Action:    action,
		Current:   0,
		Total:     total,
		IsRunning: true,
	}

	s.logger.Infof("Created task %s with %d total items", taskID, total)
}

// UpdateProgress 更新任务进度
func (s *Service) UpdateProgress(taskID string, current int, currentNode string, userID uint) {
	s.taskMutex.RLock()
	task, exists := s.tasks[taskID]
	s.taskMutex.RUnlock()

	if !exists {
		s.logger.Warningf("Task %s not found for progress update", taskID)
		return
	}

	if !task.IsRunning {
		s.logger.Warningf("Task %s is not running, ignoring progress update", taskID)
		return
	}

	task.Current = current
	progress := float64(current) / float64(task.Total) * 100

	// 检查WebSocket连接状态，避免无效发送
	s.connMutex.RLock()
	_, hasConnection := s.connections[userID]
	s.connMutex.RUnlock()

	if hasConnection {
		message := ProgressMessage{
			TaskID:      taskID,
			Type:        "progress",
			Action:      task.Action,
			Current:     current,
			Total:       task.Total,
			Progress:    progress,
			CurrentNode: currentNode,
			Message:     fmt.Sprintf("正在处理节点 %s (%d/%d)", currentNode, current, task.Total),
			Timestamp:   time.Now(),
		}

		s.sendToUser(userID, message)
		s.logger.Infof("Progress updated for task %s: %d/%d", taskID, current, task.Total)
	} else {
		// 只在关键节点记录连接缺失（减少日志噪音）
		if current == task.Total || current%5 == 0 {
			s.logger.Infof("Progress update for task %s (%d/%d) - no WebSocket connection for user %d", taskID, current, task.Total, userID)
		}
	}
}

// CompleteTask 完成任务
func (s *Service) CompleteTask(taskID string, userID uint) {
	s.taskMutex.Lock()
	task, exists := s.tasks[taskID]
	if exists {
		if !task.IsRunning {
			s.taskMutex.Unlock()
			s.logger.Warningf("Task %s was already completed or stopped", taskID)
			return
		}
		task.IsRunning = false
		// 保留任务状态一段时间，以便重连的客户端获取状态
		// 使用延时删除，给重连客户端一些时间获取状态
		go func() {
			time.Sleep(30 * time.Second) // 保留30秒
			s.taskMutex.Lock()
			delete(s.tasks, taskID)
			s.taskMutex.Unlock()
		}()
	}
	s.taskMutex.Unlock()

	if !exists {
		s.logger.Warningf("Task %s not found for completion", taskID)
		return
	}

	message := ProgressMessage{
		TaskID:    taskID,
		Type:      "complete",
		Action:    task.Action,
		Current:   task.Total,
		Total:     task.Total,
		Progress:  100,
		Message:   fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total),
		Timestamp: time.Now(),
	}

	// 检查连接状态并发送完成消息
	s.connMutex.RLock()
	_, hasConnection := s.connections[userID]
	s.connMutex.RUnlock()

	if hasConnection {
		// 发送完成消息
		s.sendToUser(userID, message)
		s.logger.Infof("Task %s completed successfully, sent completion message to connected user %d", taskID, userID)
	} else {
		s.logger.Warningf("Task %s completed but no WebSocket connection for user %d, task status preserved for recovery", taskID, userID)
	}
}

// ErrorTask 任务错误
func (s *Service) ErrorTask(taskID string, err error, userID uint) {
	s.taskMutex.Lock()
	task, exists := s.tasks[taskID]
	if exists {
		task.IsRunning = false
		task.Error = err
		delete(s.tasks, taskID)
	}
	s.taskMutex.Unlock()

	if !exists {
		return
	}

	message := ProgressMessage{
		TaskID:    taskID,
		Type:      "error",
		Action:    task.Action,
		Message:   "批量操作失败",
		Error:     err.Error(),
		Timestamp: time.Now(),
	}

	s.sendToUser(userID, message)
}

// cleanupStaleConnections 定期清理不活跃的连接
func (s *Service) cleanupStaleConnections() {
	ticker := time.NewTicker(60 * time.Second) // 每60秒检查一次
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		var staleUsers []uint

		s.connMutex.RLock()
		for userID, conn := range s.connections {
			// 如果连接超过2分钟没有活动，标记为过期
			if now.Sub(conn.lastSeen) > 2*time.Minute {
				staleUsers = append(staleUsers, userID)
			}
		}
		s.connMutex.RUnlock()

		// 清理过期连接
		for _, userID := range staleUsers {
			s.logger.Warningf("Cleaning up stale connection for user %d", userID)
			s.removeConnection(userID)
		}

		if len(staleUsers) > 0 {
			s.logger.Infof("Cleaned up %d stale connections", len(staleUsers))
		}
	}
}

// sendCurrentTaskStatus 发送当前任务状态给重连的用户
func (s *Service) sendCurrentTaskStatus(userID uint) {
	s.taskMutex.RLock()
	defer s.taskMutex.RUnlock()

	sentCount := 0
	// 查找当前用户的相关任务
	for taskID, task := range s.tasks {
		// 发送任务状态
		var messageType string
		var progress float64
		var message string

		if task.IsRunning {
			messageType = "progress"
			progress = float64(task.Current) / float64(task.Total) * 100
			message = fmt.Sprintf("正在处理 (%d/%d)", task.Current, task.Total)
		} else {
			messageType = "complete"
			progress = 100
			message = fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total)
		}

		statusMessage := ProgressMessage{
			TaskID:    taskID,
			Type:      messageType,
			Action:    task.Action,
			Current:   task.Current,
			Total:     task.Total,
			Progress:  progress,
			Message:   message,
			Timestamp: time.Now(),
		}

		s.sendToUser(userID, statusMessage)
		s.logger.Infof("Sent recovery task status for %s to user %d: %s (%.1f%%)", taskID, userID, messageType, progress)
		sentCount++
	}

	if sentCount == 0 {
		s.logger.Infof("No pending tasks found for user %d on reconnection", userID)
	} else {
		s.logger.Infof("Sent %d task status updates to user %d on reconnection", sentCount, userID)
	}
}

// NodeResult 单个节点处理结果
type NodeResult struct {
	NodeName string
	Success  bool
	Error    string
}

// BatchProcessor 批量处理器接口
type BatchProcessor interface {
	ProcessNode(ctx context.Context, nodeName string, index int) error
}

// ProcessBatchWithProgress 带进度的批量处理
func (s *Service) ProcessBatchWithProgress(
	ctx context.Context,
	taskID string,
	action string,
	nodeNames []string,
	userID uint,
	maxConcurrency int,
	processor BatchProcessor,
) error {
	total := len(nodeNames)

	// 创建任务
	s.CreateTask(taskID, action, total)

	// 使用信号量控制并发
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex
	var errors []string
	processed := 0

	for i, nodeName := range nodeNames {
		wg.Add(1)
		go func(index int, node string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 更新进度
			mu.Lock()
			processed++
			currentIndex := processed
			mu.Unlock()

			s.UpdateProgress(taskID, currentIndex, node, userID)

			// 处理节点
			if err := processor.ProcessNode(ctx, node, index); err != nil {
				mu.Lock()
				errors = append(errors, fmt.Sprintf("%s: %v", node, err))
				mu.Unlock()
				s.logger.Errorf("Failed to process node %s: %v", node, err)
			} else {
				s.logger.Infof("Successfully processed node %s (%d/%d)", node, currentIndex, total)
			}
		}(i, nodeName)
	}

	// 等待所有任务完成
	wg.Wait()

	// 处理结果
	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("部分节点处理失败: %s", errors[0])
		if len(errors) > 1 {
			errorMsg = fmt.Sprintf("部分节点处理失败: %s 等 %d 个错误", errors[0], len(errors))
		}
		err := fmt.Errorf("%s", errorMsg)
		s.ErrorTask(taskID, err, userID)
		return err
	}

	s.CompleteTask(taskID, userID)
	return nil
}
