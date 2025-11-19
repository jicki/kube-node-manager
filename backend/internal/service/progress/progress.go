package progress

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/auth"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"
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
	TaskID          string
	Action          string
	Current         int
	Total           int
	IsRunning       bool
	Error           error
	Completed       bool
	CompletedAt     time.Time
	UserID          uint
	PendingMessages []ProgressMessage // 待发送的消息队列
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
	// 存储用户连接 map[userID]map[*Connection]bool
	connections map[uint]map[*Connection]bool
	// 存储任务进度
	tasks map[string]*TaskProgress
	// 保护连接映射
	connMutex sync.RWMutex
	// 保护任务映射
	taskMutex   sync.RWMutex
	logger      *logger.Logger
	authService TokenValidator
	// 新增：完成任务的消息队列，用于重连时恢复
	completedTasks map[uint][]ProgressMessage
	completedMutex sync.RWMutex
	// 数据库进度服务（用于多副本环境）
	dbProgressService *DatabaseProgressService
	useDatabase       bool
}

// NewService 创建进度推送服务
func NewService(logger *logger.Logger) *Service {
	s := &Service{
		connections:    make(map[uint]map[*Connection]bool),
		tasks:          make(map[string]*TaskProgress),
		completedTasks: make(map[uint][]ProgressMessage),
		logger:         logger,
		useDatabase:    false, // 默认使用内存模式
	}

	// 启动定期清理goroutine
	go s.cleanupStaleConnections()

	return s
}

// EnableDatabaseMode 启用数据库模式（用于多副本环境）
func (s *Service) EnableDatabaseMode(db *gorm.DB) {
	s.dbProgressService = NewDatabaseProgressService(db, s.logger, s)
	s.useDatabase = true
	s.logger.Infof("Progress service enabled database mode for multi-replica support")
}

// SetAuthService 设置认证服务
func (s *Service) SetAuthService(authService TokenValidator) {
	s.authService = authService
}

// HandleWebSocket 处理WebSocket连接
func (s *Service) HandleWebSocket(c *gin.Context) {
	// 从查询参数获取token
	token := c.Query("token")
	if token == "" {
		s.logger.Errorf("WebSocket connection failed: no token provided")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "缺少认证token"})
		return
	}

	// 验证token
	if s.authService == nil {
		s.logger.Errorf("Auth service not set for WebSocket authentication")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "认证服务未配置"})
		return
	}

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

	// 注册连接
	s.connMutex.Lock()
	if _, exists := s.connections[userID]; !exists {
		s.connections[userID] = make(map[*Connection]bool)
	}
	s.connections[userID][conn] = true
	s.connMutex.Unlock()

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
	// 在多副本模式下，这一步至关重要，因为内存中可能没有任务状态
	s.sendCurrentTaskStatus(userID)

	// 如果启用了数据库模式，也检查数据库中的未处理消息
	if s.useDatabase && s.dbProgressService != nil {
		go func() {
			time.Sleep(100 * time.Millisecond) // 稍等一下确保连接稳定
			s.dbProgressService.processUnsentMessages()
		}()
	}

	// 等待连接关闭
	select {}
}

// writePump 发送消息到客户端
func (s *Service) writePump(conn *Connection) {
	ticker := time.NewTicker(20 * time.Second) // 缩短心跳间隔以提高连接检测
	defer func() {
		ticker.Stop()
		conn.ws.Close()
		s.removeConnection(conn)
		// 减少日志输出，仅在异常情况下记录
	}()

	for {
		select {
		case message, ok := <-conn.send:
			conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				// 静默关闭，减少日志输出
				conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			// 对于重要消息，多次尝试发送
			maxRetries := 1
			if message.Type == "complete" || message.Type == "error" {
				maxRetries = 3
			}

			var err error
			for i := 0; i <= maxRetries; i++ {
				if i > 0 {
					time.Sleep(100 * time.Millisecond) // 重试间隔
					conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
				}
				err = conn.ws.WriteJSON(message)
				if err == nil {
					break
				}
				s.logger.Warningf("Write attempt %d failed for user %d: %v", i+1, conn.userID, err)
			}

			if err != nil {
				s.logger.Errorf("Failed to write message to user %d after %d attempts: %v", conn.userID, maxRetries+1, err)
				// 对于重要消息，保存到队列中
				if message.Type == "complete" || message.Type == "error" {
					s.queueCompletionMessage(conn.userID, message)
				}
				return
			}

			// 更新活跃时间
			conn.lastSeen = time.Now()

			// 添加发送成功日志（仅对重要消息）
			if message.Type == "complete" || message.Type == "error" {
				s.logger.Infof("Successfully sent %s message to user %d for task %s", message.Type, conn.userID, message.TaskID)
			}

		case <-ticker.C:
			conn.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Errorf("Failed to send ping to user %d: %v", conn.userID, err)
				return
			}
			// 减少心跳日志噪音
			if conn.userID%10 == 0 { // 只为部分用户记录心跳
				s.logger.Infof("Sent ping to user %d", conn.userID)
			}
		}
	}
}

// readPump 接收客户端消息
func (s *Service) readPump(conn *Connection) {
	defer func() {
		conn.ws.Close()
		s.removeConnection(conn)
		// 减少日志输出
	}()

	conn.ws.SetReadLimit(1024)
	conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second)) // 调整读取超时时间
	conn.ws.SetPongHandler(func(string) error {
		conn.lastSeen = time.Now() // 更新活跃时间
		conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		messageType, message, err := conn.ws.ReadMessage()
		if err != nil {
			// 只记录真正的异常关闭错误
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure, websocket.CloseNormalClosure, websocket.CloseNoStatusReceived) {
				s.logger.Errorf("WebSocket unexpected error for user %d: %v", conn.userID, err)
			}
			// 其他正常关闭（包括 CloseNoStatusReceived）不记录日志
			break
		}

		// 更新活跃时间
		conn.lastSeen = time.Now()

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			// 减少日志噪音，只记录重要消息
			if len(message) > 0 && string(message) != "ping" {
				s.logger.Infof("Received text message from user %d: %s", conn.userID, string(message))
			}
		case websocket.BinaryMessage:
			s.logger.Infof("Received binary message from user %d", conn.userID)
		case websocket.PingMessage:
			// 响应ping消息
			conn.ws.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.ws.WriteMessage(websocket.PongMessage, nil); err != nil {
				s.logger.Errorf("Failed to send pong to user %d: %v", conn.userID, err)
				return
			}
		}
	}
}

// removeConnection 移除连接
func (s *Service) removeConnection(conn *Connection) {
	s.connMutex.Lock()
	defer s.connMutex.Unlock()

	if userConns, exists := s.connections[conn.userID]; exists {
		if _, ok := userConns[conn]; ok {
			delete(userConns, conn)
			close(conn.send)
			// 如果用户没有任何连接了，清理map
			if len(userConns) == 0 {
				delete(s.connections, conn.userID)
			}
		}
	}
}

// sendToUser 发送消息给指定用户
func (s *Service) sendToUser(userID uint, message ProgressMessage) {
	s.connMutex.RLock()
	userConns, exists := s.connections[userID]
	// 复制一份连接列表，避免在锁内发送消息（防止阻塞）
	var conns []*Connection
	if exists {
		for conn := range userConns {
			conns = append(conns, conn)
		}
	}
	s.connMutex.RUnlock()

	if !exists || len(conns) == 0 {
		// 如果是重要消息（完成或错误），保存到队列中
		if message.Type == "complete" || message.Type == "error" {
			s.queueCompletionMessage(userID, message)
			s.logger.Warningf("No WebSocket connection found for user %d, queued %s message for task %s", userID, message.Type, message.TaskID)
		}
		return
	}

	// 尝试发送消息到所有连接
	timeout := 1 * time.Second
	if message.Type == "complete" || message.Type == "error" {
		timeout = 3 * time.Second // 重要消息使用更长超时
	}

	for _, conn := range conns {
		go func(c *Connection) {
			select {
			case c.send <- message:
				// 消息发送成功
				if message.Type == "complete" || message.Type == "error" {
					s.logger.Infof("Successfully sent %s message to user %d for task %s", message.Type, userID, message.TaskID)
				}
			case <-time.After(timeout):
				s.logger.Warningf("Send queue timeout for user %d (type: %s)", userID, message.Type)
				// 对于重要消息，保存到队列中并立即尝试重连
				// 注意：在多连接模式下，单个连接超时不代表用户断开，但仍需谨慎处理
				if message.Type == "complete" || message.Type == "error" {
					// 这里可能导致重复排队，但宁可重复也不要丢失
					// 只有当这是唯一连接时才排队？不，简单起见还是排队吧，前端处理重复
					// 但为了避免日志爆炸，我们只在第一个连接超时时记录/排队
					// 简化：每个连接都尝试发送，如果失败则关闭该连接
					
					// 连接可能有问题，标记为需要重连
					go func() {
						time.Sleep(500 * time.Millisecond)
						// 直接关闭连接触发重连逻辑
						c.ws.Close()
					}()
				}
			}
		}(conn)
	}

	// 如果所有连接都失败，消息可能会丢失。
	// 但由于我们是并发发送，很难知道是否"所有"都失败。
	// 这里的逻辑是：只要有一个连接存在，就尝试发送。如果都失败了，客户端会重连并获取状态。
}

// CreateTask 创建新任务
func (s *Service) CreateTask(taskID, action string, total int, userID uint) {
	s.taskMutex.Lock()
	defer s.taskMutex.Unlock()

	// 检查是否已存在相同的任务，如果存在则清理
	if existingTask, exists := s.tasks[taskID]; exists {
		s.logger.Infof("Replacing existing task %s (action: %s)", taskID, existingTask.Action)
		existingTask.IsRunning = false
	}

	s.tasks[taskID] = &TaskProgress{
		TaskID:          taskID,
		Action:          action,
		Current:         0,
		Total:           total,
		IsRunning:       true,
		Completed:       false,
		UserID:          userID,
		PendingMessages: make([]ProgressMessage, 0),
	}

	s.logger.Infof("Created task %s with %d total items for user %d", taskID, total, userID)
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
		task.Completed = true
		task.CompletedAt = time.Now()
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
		// 没有连接时，将消息保存到队列中等待重连
		s.queueCompletionMessage(userID, message)
		s.logger.Warningf("Task %s completed but no WebSocket connection for user %d, message queued for recovery", taskID, userID)
	}

	// 延时清理任务和完成消息
	go func() {
		time.Sleep(60 * time.Second) // 增加到60秒，给更多时间重连
		s.taskMutex.Lock()
		delete(s.tasks, taskID)
		s.taskMutex.Unlock()

		// 清理过期的完成消息
		s.completedMutex.Lock()
		if messages, exists := s.completedTasks[userID]; exists {
			var remaining []ProgressMessage
			for _, msg := range messages {
				if time.Since(msg.Timestamp) < 60*time.Second {
					remaining = append(remaining, msg)
				}
			}
			if len(remaining) > 0 {
				s.completedTasks[userID] = remaining
			} else {
				delete(s.completedTasks, userID)
			}
		}
		s.completedMutex.Unlock()
	}()
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
		var staleConns []*Connection

		s.connMutex.RLock()
		for _, userConns := range s.connections {
			for conn := range userConns {
				// 如果连接超过2分钟没有活动，标记为过期
				if now.Sub(conn.lastSeen) > 2*time.Minute {
					staleConns = append(staleConns, conn)
				}
			}
		}
		s.connMutex.RUnlock()

		// 清理过期连接
		for _, conn := range staleConns {
			s.logger.Warningf("Cleaning up stale connection for user %d", conn.userID)
			s.removeConnection(conn)
		}

		if len(staleConns) > 0 {
			s.logger.Infof("Cleaned up %d stale connections", len(staleConns))
		}
	}
}

// queueCompletionMessage 将完成消息加入队列，等待重连时发送
func (s *Service) queueCompletionMessage(userID uint, message ProgressMessage) {
	s.completedMutex.Lock()
	defer s.completedMutex.Unlock()

	// 初始化用户的消息队列
	if s.completedTasks[userID] == nil {
		s.completedTasks[userID] = make([]ProgressMessage, 0)
	}

	// 添加消息到队列
	s.completedTasks[userID] = append(s.completedTasks[userID], message)
	s.logger.Infof("Queued completion message for user %d, task %s", userID, message.TaskID)
}

// sendCurrentTaskStatus 发送当前任务状态给重连的用户
func (s *Service) sendCurrentTaskStatus(userID uint) {
	sentCount := 0

	// 先发送待处理的完成消息
	s.completedMutex.Lock()
	if messages, exists := s.completedTasks[userID]; exists {
		for _, message := range messages {
			// 只发送未过期的消息（60秒内）
			if time.Since(message.Timestamp) < 60*time.Second {
				s.sendToUser(userID, message)
				s.logger.Infof("Sent recovery task status for %s to user %d: %s (%.1f%%)", message.TaskID, userID, message.Type, message.Progress)
				sentCount++
			}
		}
		// 清理已发送的消息
		delete(s.completedTasks, userID)
	}
	s.completedMutex.Unlock()

	// 再发送当前正在运行的任务状态
	s.taskMutex.RLock()
	for taskID, task := range s.tasks {
		// 只处理当前用户的任务
		if task.UserID != userID {
			continue
		}

		var messageType string
		var progress float64
		var message string

		if task.IsRunning {
			messageType = "progress"
			progress = float64(task.Current) / float64(task.Total) * 100
			message = fmt.Sprintf("正在处理 (%d/%d)", task.Current, task.Total)
		} else if task.Completed {
			messageType = "complete"
			progress = 100
			message = fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total)
		} else {
			continue // 跳过已停止但未完成的任务
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
	s.taskMutex.RUnlock()

	// 如果启用了数据库模式，检查数据库中的任务状态
	if s.useDatabase && s.dbProgressService != nil {
		// 获取所有相关状态的任务（运行中、已完成、失败）
		statuses := []model.TaskStatus{
			model.TaskStatusRunning,
			model.TaskStatusCompleted,
			model.TaskStatusFailed,
		}

		for _, status := range statuses {
			tasks, err := s.dbProgressService.GetUserTasks(userID, status)
			if err != nil {
				s.logger.Errorf("Failed to get DB tasks for user %d (status: %s): %v", userID, status, err)
				continue
			}

			for _, task := range tasks {
				var msgType string
				var progress float64
				var message string

				if task.Status == model.TaskStatusRunning {
					// 如果状态是运行中但进度已满，视为完成（修复潜在的状态不一致）
					if task.Total > 0 && task.Current >= task.Total {
						msgType = "complete"
						progress = 100
						message = fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total)
					} else {
						msgType = "progress"
						if task.Total > 0 {
							progress = float64(task.Current) / float64(task.Total) * 100
						}
						message = fmt.Sprintf("正在处理 (%d/%d)", task.Current, task.Total)
					}
				} else if task.Status == model.TaskStatusCompleted {
					msgType = "complete"
					progress = 100
					message = fmt.Sprintf("批量操作完成，共处理 %d 个节点", task.Total)
				} else {
					msgType = "error"
					progress = task.Progress
					message = task.ErrorMsg
				}

				// 构造消息
				progressMessage := ProgressMessage{
					TaskID:      task.TaskID,
					Type:        msgType,
					Action:      task.Action,
					Current:     task.Current,
					Total:       task.Total,
					Progress:    progress,
					CurrentNode: task.CurrentNode,
					Message:     message,
					Error:       task.ErrorMsg,
					Timestamp:   task.UpdatedAt,
				}

				// 对于完成/失败状态，延长过期时间到5分钟，确保用户有足够时间看到结果
				if (msgType == "complete" || msgType == "error") && task.CompletedAt != nil {
					if time.Since(*task.CompletedAt) > 5*time.Minute {
						continue
					}
				}

				s.sendToUser(userID, progressMessage)
				s.logger.Infof("Sent recovery DB task status for %s to user %d: %s (%.1f%%)", task.TaskID, userID, msgType, progress)
				sentCount++
			}
		}
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
	// 如果启用了数据库模式，使用数据库进度服务
	if s.useDatabase && s.dbProgressService != nil {
		return s.dbProgressService.ProcessBatchWithProgress(ctx, taskID, action, nodeNames, userID, maxConcurrency, processor)
	}

	// 否则使用原有的内存模式
	total := len(nodeNames)

	// 创建任务
	s.CreateTask(taskID, action, total, userID)

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

			// 先获取当前索引用于日志
			mu.Lock()
			processed++
			currentIndex := processed
			mu.Unlock()

			// 发送开始处理的进度消息
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

			// 处理完成后再次更新进度，确保前端收到最新状态
			s.UpdateProgress(taskID, currentIndex, node, userID)
		}(i, nodeName)
	}

	// 等待所有任务完成
	wg.Wait()

	s.logger.Infof("All nodes processed for task %s, processed=%d, errors=%d", taskID, processed, len(errors))

	// 确保最后一次进度更新显示 100%
	if len(errors) == 0 {
		s.logger.Infof("Sending final 100%% progress update for task %s", taskID)
		s.UpdateProgress(taskID, total, "完成", userID)
	}

	// 处理结果
	if len(errors) > 0 {
		errorMsg := fmt.Sprintf("部分节点处理失败: %s", errors[0])
		if len(errors) > 1 {
			errorMsg = fmt.Sprintf("部分节点处理失败: %s 等 %d 个错误", errors[0], len(errors))
		}
		s.logger.Errorf("Task %s failed with %d errors, calling ErrorTask", taskID, len(errors))
		err := fmt.Errorf("%s", errorMsg)
		s.ErrorTask(taskID, err, userID)
		return err
	}

	s.logger.Infof("Task %s completed successfully, calling CompleteTask for user %d", taskID, userID)
	s.CompleteTask(taskID, userID)
	return nil
}
