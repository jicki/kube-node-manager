package terminal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/node"
	"kube-node-manager/pkg/logger"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"golang.org/x/crypto/ssh"
)

type Handler struct {
	nodeSvc  *node.Service
	auditSvc *audit.Service
	logger   *logger.Logger
	upgrader websocket.Upgrader
}

func NewHandler(nodeSvc *node.Service, auditSvc *audit.Service, logger *logger.Logger) *Handler {
	return &Handler{
		nodeSvc:  nodeSvc,
		auditSvc: auditSvc,
		logger:   logger,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true // 允许跨域，生产环境应限制
			},
		},
	}
}

// TerminalMessage 定义终端消息协议
type TerminalMessage struct {
	Type string `json:"type"` // "input", "resize", "ping"
	Data string `json:"data,omitempty"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

func (h *Handler) HandleWebSocket(c *gin.Context) {
	// 1. 获取参数
	clusterName := c.Query("cluster_name")
	nodeName := c.Query("node_name")

	if clusterName == "" || nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cluster_name and node_name are required"})
		return
	}

	// 2. 权限检查 (Admin only)
	// 注意：WebSocket 连接通常不能携带自定义 Header，AuthMiddleware 可能需要从 Query Token 获取
	// 假设 AuthMiddleware 已经处理了 Token 验证并放入 Context
	// 如果是独立 Handler，且未经过 Middleware，需要手动验证 Token
	
	userID, exists := c.Get("user_id")
	if !exists {
		// 尝试从 Query 参数获取 token (如果未经过中间件)
		// 这里假设已经通过了中间件，或者在路由配置中使用了中间件
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	
	userRole, _ := c.Get("user_role")
	if userRole != model.RoleAdmin {
		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Admin only"})
		return
	}

	// 3. 升级 WebSocket
	ws, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade websocket: %v", err)
		return
	}
	defer ws.Close()

	// 4. 获取 SSH 配置
	sshKey, host, err := h.nodeSvc.GetNodeSSHConfig(clusterName, nodeName)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nError: %v\r\n", err)))
		return
	}

	// 5. 建立 SSH 连接
	authMethods := []ssh.AuthMethod{}
	if sshKey.Type == model.SSHKeyTypePrivateKey {
		signer, err := ssh.ParsePrivateKey([]byte(sshKey.PrivateKey))
		if err != nil {
			// 尝试带密码的私钥
			if sshKey.Passphrase != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase([]byte(sshKey.PrivateKey), []byte(sshKey.Passphrase))
			}
		}
		if err != nil {
			ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nError parsing private key: %v\r\n", err)))
			return
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	} else {
		authMethods = append(authMethods, ssh.Password(sshKey.Password))
	}

	sshConfig := &ssh.ClientConfig{
		User:            sshKey.Username,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应验证 Host Key
		Timeout:         10 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, sshKey.Port)
	client, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to connect to %s: %v\r\n", addr, err)))
		return
	}
	defer client.Close()

	// 6. 创建 Session
	session, err := client.NewSession()
	if err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to create session: %v\r\n", err)))
		return
	}
	defer session.Close()

	// 7. 请求 PTY
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
	}
	if err := session.RequestPty("xterm", 24, 80, modes); err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to request PTY: %v\r\n", err)))
		return
	}

	// 8. 管道处理
	stdin, err := session.StdinPipe()
	if err != nil {
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		return
	}
	stderr, err := session.StderrPipe()
	if err != nil {
		return
	}

	// 9. 启动 Shell
	if err := session.Shell(); err != nil {
		ws.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("\r\nFailed to start shell: %v\r\n", err)))
		return
	}

	// 审计日志：开始连接
	clusterID, _ := h.auditSvc.GetClusterIDByName(clusterName)
	h.auditSvc.Log(audit.LogRequest{
		UserID:       userID.(uint),
		ClusterID:    &clusterID,
		NodeName:     nodeName,
		Action:       model.ActionConnect,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Started terminal session to node %s (%s)", nodeName, host),
		Status:       model.AuditStatusSuccess,
	})

	// 读取 SSH 输出 -> WebSocket
	go func() {
		defer ws.Close()
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if err != nil {
				if err != io.EOF {
					h.logger.Error("SSH stdout read error: %v", err)
				}
				return
			}
			if n > 0 {
				// 发送二进制或文本，xterm.js 都能处理
				// 为了简单，直接发文本
				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if err != nil {
				if err != io.EOF {
					h.logger.Error("SSH stderr read error: %v", err)
				}
				return
			}
			if n > 0 {
				if err := ws.WriteMessage(websocket.TextMessage, buf[:n]); err != nil {
					return
				}
			}
		}
	}()

	// 读取 WebSocket -> SSH 输入
	for {
		_, message, err := ws.ReadMessage()
		if err != nil {
			break
		}

		// 尝试解析为 JSON 命令
		var msg TerminalMessage
		if err := json.Unmarshal(message, &msg); err == nil && msg.Type != "" {
			switch msg.Type {
			case "input":
				stdin.Write([]byte(msg.Data))
			case "resize":
				session.WindowChange(msg.Rows, msg.Cols)
			case "ping":
				// ignore
			}
		} else {
			// 如果不是 JSON，或者是纯文本输入，直接当做输入
			stdin.Write(message)
		}
	}
	
	// 审计日志：结束连接
	h.auditSvc.Log(audit.LogRequest{
		UserID:       userID.(uint),
		ClusterID:    &clusterID,
		NodeName:     nodeName,
		Action:       model.ActionConnect,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Ended terminal session to node %s", nodeName),
		Status:       model.AuditStatusSuccess,
	})
}

// GetSettings 获取节点 SSH 配置
func (h *Handler) GetSettings(c *gin.Context) {
	clusterName := c.Query("cluster_name")
	nodeName := c.Param("node_name")

	if clusterName == "" || nodeName == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "params required"})
		return
	}

	settings, err := h.nodeSvc.GetNodeSettings(clusterName, nodeName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	
	if settings == nil {
		// 返回默认值（空结构）
		c.JSON(http.StatusOK, gin.H{"data": model.NodeSettings{
			ClusterName: clusterName,
			NodeName:    nodeName,
			SSHPort:     22,
		}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": settings})
}

// UpdateSettings 更新节点 SSH 配置
func (h *Handler) UpdateSettings(c *gin.Context) {
	clusterName := c.Query("cluster_name")
	nodeName := c.Param("node_name")

	var req model.NodeSettings
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.ClusterName = clusterName
	req.NodeName = nodeName

	if err := h.nodeSvc.SaveNodeSettings(&req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "success"})
}
