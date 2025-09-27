package progress

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

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
	ws     *websocket.Conn
	send   chan ProgressMessage
	userID uint
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
	taskMutex sync.RWMutex
	logger    *logger.Logger
}

// NewService 创建进度推送服务
func NewService(logger *logger.Logger) *Service {
	return &Service{
		connections: make(map[uint]*Connection),
		tasks:       make(map[string]*TaskProgress),
		logger:      logger,
	}
}

// HandleWebSocket 处理WebSocket连接
func (s *Service) HandleWebSocket(c *gin.Context) {
	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "用户未认证"})
		return
	}

	// 升级HTTP连接为WebSocket
	ws, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.logger.Errorf("Failed to upgrade websocket: %v", err)
		return
	}
	defer ws.Close()

	// 创建连接
	conn := &Connection{
		ws:     ws,
		send:   make(chan ProgressMessage, 256),
		userID: userID.(uint),
	}

	// 注册连接
	s.connMutex.Lock()
	s.connections[userID.(uint)] = conn
	s.connMutex.Unlock()

	s.logger.Infof("WebSocket connected for user %d", userID.(uint))

	// 启动消息发送goroutine
	go s.writePump(conn)

	// 启动消息接收goroutine（处理心跳等）
	go s.readPump(conn)

	// 发送连接确认消息
	s.sendToUser(userID.(uint), ProgressMessage{
		Type:      "connected",
		Message:   "WebSocket连接成功",
		Timestamp: time.Now(),
	})

	// 等待连接关闭
	select {}
}

// writePump 发送消息到客户端
func (s *Service) writePump(conn *Connection) {
	ticker := time.NewTicker(54 * time.Second) // 心跳间隔
	defer func() {
		ticker.Stop()
		conn.ws.Close()
		s.removeConnection(conn.userID)
	}()

	for {
		select {
		case message, ok := <-conn.send:
			conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				conn.ws.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := conn.ws.WriteJSON(message); err != nil {
				s.logger.Errorf("Failed to write message: %v", err)
				return
			}

		case <-ticker.C:
			conn.ws.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := conn.ws.WriteMessage(websocket.PingMessage, nil); err != nil {
				s.logger.Errorf("Failed to send ping: %v", err)
				return
			}
		}
	}
}

// readPump 接收客户端消息
func (s *Service) readPump(conn *Connection) {
	defer func() {
		conn.ws.Close()
		s.removeConnection(conn.userID)
	}()

	conn.ws.SetReadLimit(512)
	conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.ws.SetPongHandler(func(string) error {
		conn.ws.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, _, err := conn.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Errorf("WebSocket error: %v", err)
			}
			break
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
		return
	}

	select {
	case conn.send <- message:
	case <-time.After(1 * time.Second):
		s.logger.Warningf("Failed to send message to user %d: timeout", userID)
		s.removeConnection(userID)
	}
}

// CreateTask 创建新任务
func (s *Service) CreateTask(taskID, action string, total int) {
	s.taskMutex.Lock()
	s.tasks[taskID] = &TaskProgress{
		TaskID:    taskID,
		Action:    action,
		Current:   0,
		Total:     total,
		IsRunning: true,
	}
	s.taskMutex.Unlock()
}

// UpdateProgress 更新任务进度
func (s *Service) UpdateProgress(taskID string, current int, currentNode string, userID uint) {
	s.taskMutex.RLock()
	task, exists := s.tasks[taskID]
	s.taskMutex.RUnlock()

	if !exists {
		return
	}

	task.Current = current
	progress := float64(current) / float64(task.Total) * 100

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
}

// CompleteTask 完成任务
func (s *Service) CompleteTask(taskID string, userID uint) {
	s.taskMutex.Lock()
	task, exists := s.tasks[taskID]
	if exists {
		task.IsRunning = false
		delete(s.tasks, taskID)
	}
	s.taskMutex.Unlock()

	if !exists {
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

	s.sendToUser(userID, message)
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
