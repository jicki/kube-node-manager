package automation

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"golang.org/x/crypto/ssh"
)

// SSHClient SSH 客户端包装器
type SSHClient struct {
	client     *ssh.Client
	host       string
	credential *model.SSHCredential
	lastUsed   time.Time
	mu         sync.Mutex
}

// SSHClientPool SSH 客户端连接池
type SSHClientPool struct {
	logger         *logger.Logger
	credentialMgr  *CredentialManager
	clients        map[string]*SSHClient // key: host:port
	maxPoolSize    int
	idleTimeout    time.Duration
	connectTimeout time.Duration
	mu             sync.RWMutex
}

// SSHExecuteResult SSH 命令执行结果
type SSHExecuteResult struct {
	Host     string
	Command  string
	ExitCode int
	Stdout   string
	Stderr   string
	Error    string
	Duration time.Duration
}

// NewSSHClientPool 创建 SSH 客户端连接池
func NewSSHClientPool(logger *logger.Logger, credentialMgr *CredentialManager, maxPoolSize int, idleTimeout time.Duration) *SSHClientPool {
	pool := &SSHClientPool{
		logger:         logger,
		credentialMgr:  credentialMgr,
		clients:        make(map[string]*SSHClient),
		maxPoolSize:    maxPoolSize,
		idleTimeout:    idleTimeout,
		connectTimeout: 30 * time.Second,
	}

	// 启动空闲连接清理
	go pool.cleanupIdleConnections()

	return pool
}

// GetClient 获取或创建 SSH 客户端
func (p *SSHClientPool) GetClient(host string, credential *model.SSHCredential) (*SSHClient, error) {
	key := fmt.Sprintf("%s:%d", host, credential.Port)

	p.mu.Lock()
	defer p.mu.Unlock()

	// 检查是否已存在连接
	if client, exists := p.clients[key]; exists {
		client.mu.Lock()
		// 检查连接是否仍然有效
		if p.isConnected(client.client) {
			client.lastUsed = time.Now()
			client.mu.Unlock()
			return client, nil
		}
		// 连接已断开，关闭并移除
		client.client.Close()
		delete(p.clients, key)
		client.mu.Unlock()
	}

	// 检查连接池大小
	if len(p.clients) >= p.maxPoolSize {
		// 移除最久未使用的连接
		p.removeOldestConnection()
	}

	// 创建新连接
	client, err := p.createClient(host, credential)
	if err != nil {
		return nil, fmt.Errorf("failed to create SSH client: %w", err)
	}

	p.clients[key] = client
	return client, nil
}

// createClient 创建新的 SSH 客户端连接
func (p *SSHClientPool) createClient(host string, credential *model.SSHCredential) (*SSHClient, error) {
	// 构建 SSH 配置
	config := &ssh.ClientConfig{
		User:            credential.Username,
		Timeout:         p.connectTimeout,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 生产环境应该验证主机密钥
	}

	// 配置认证方式
	if credential.AuthType == "privatekey" && credential.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(credential.PrivateKey))
		if err != nil {
			// 尝试使用 passphrase
			if credential.Passphrase != "" {
				signer, err = ssh.ParsePrivateKeyWithPassphrase(
					[]byte(credential.PrivateKey),
					[]byte(credential.Passphrase),
				)
			}
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		}
		config.Auth = []ssh.AuthMethod{ssh.PublicKeys(signer)}
	} else if credential.AuthType == "password" && credential.Password != "" {
		config.Auth = []ssh.AuthMethod{ssh.Password(credential.Password)}
	} else {
		return nil, fmt.Errorf("invalid credential configuration")
	}

	// 连接到 SSH 服务器
	addr := fmt.Sprintf("%s:%d", host, credential.Port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", addr, err)
	}

	p.logger.Infof("SSH connection established to %s", addr)

	return &SSHClient{
		client:     client,
		host:       host,
		credential: credential,
		lastUsed:   time.Now(),
	}, nil
}

// Execute 在远程主机上执行命令
func (c *SSHClient) Execute(command string, timeout time.Duration) (*SSHExecuteResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()

	result := &SSHExecuteResult{
		Host:    c.host,
		Command: command,
	}

	// 创建会话
	session, err := c.client.NewSession()
	if err != nil {
		result.Error = fmt.Sprintf("failed to create session: %v", err)
		return result, err
	}
	defer session.Close()

	// 设置输出缓冲区
	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// 执行命令（带超时）
	done := make(chan error, 1)
	go func() {
		done <- session.Run(command)
	}()

	select {
	case err := <-done:
		result.Stdout = stdout.String()
		result.Stderr = stderr.String()
		result.Duration = time.Since(startTime)

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.Error = err.Error()
				return result, err
			}
		} else {
			result.ExitCode = 0
		}
		return result, nil

	case <-time.After(timeout):
		// 超时，关闭会话
		session.Close()
		result.Error = "command execution timeout"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command execution timeout after %v", timeout)
	}
}

// ExecuteWithProgress 执行命令并实时输出
func (c *SSHClient) ExecuteWithProgress(command string, timeout time.Duration, outputCallback func(line string)) (*SSHExecuteResult, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	startTime := time.Now()

	result := &SSHExecuteResult{
		Host:    c.host,
		Command: command,
	}

	// 创建会话
	session, err := c.client.NewSession()
	if err != nil {
		result.Error = fmt.Sprintf("failed to create session: %v", err)
		return result, err
	}
	defer session.Close()

	// 获取输出管道
	stdout, err := session.StdoutPipe()
	if err != nil {
		result.Error = fmt.Sprintf("failed to get stdout pipe: %v", err)
		return result, err
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		result.Error = fmt.Sprintf("failed to get stderr pipe: %v", err)
		return result, err
	}

	// 启动命令
	if err := session.Start(command); err != nil {
		result.Error = fmt.Sprintf("failed to start command: %v", err)
		return result, err
	}

	// 读取输出
	var stdoutBuf, stderrBuf bytes.Buffer
	var wg sync.WaitGroup

	// 读取 stdout
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 1024)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				line := string(buf[:n])
				stdoutBuf.WriteString(line)
				if outputCallback != nil {
					outputCallback(line)
				}
			}
			if err != nil {
				break
			}
		}
	}()

	// 读取 stderr
	wg.Add(1)
	go func() {
		defer wg.Done()
		io.Copy(&stderrBuf, stderr)
	}()

	// 等待命令完成（带超时）
	done := make(chan error, 1)
	go func() {
		wg.Wait()
		done <- session.Wait()
	}()

	select {
	case err := <-done:
		result.Stdout = stdoutBuf.String()
		result.Stderr = stderrBuf.String()
		result.Duration = time.Since(startTime)

		if err != nil {
			if exitErr, ok := err.(*ssh.ExitError); ok {
				result.ExitCode = exitErr.ExitStatus()
			} else {
				result.Error = err.Error()
			}
		} else {
			result.ExitCode = 0
		}
		return result, nil

	case <-time.After(timeout):
		session.Close()
		result.Error = "command execution timeout"
		result.Duration = time.Since(startTime)
		return result, fmt.Errorf("command execution timeout after %v", timeout)
	}
}

// isConnected 检查连接是否仍然有效
func (p *SSHClientPool) isConnected(client *ssh.Client) bool {
	if client == nil {
		return false
	}

	// 尝试发送一个简单的请求
	session, err := client.NewSession()
	if err != nil {
		return false
	}
	session.Close()

	return true
}

// removeOldestConnection 移除最久未使用的连接
func (p *SSHClientPool) removeOldestConnection() {
	var oldestKey string
	var oldestTime time.Time

	for key, client := range p.clients {
		client.mu.Lock()
		if oldestKey == "" || client.lastUsed.Before(oldestTime) {
			oldestKey = key
			oldestTime = client.lastUsed
		}
		client.mu.Unlock()
	}

	if oldestKey != "" {
		if client, exists := p.clients[oldestKey]; exists {
			client.mu.Lock()
			client.client.Close()
			client.mu.Unlock()
			delete(p.clients, oldestKey)
			p.logger.Infof("Removed oldest SSH connection: %s", oldestKey)
		}
	}
}

// cleanupIdleConnections 清理空闲连接
func (p *SSHClientPool) cleanupIdleConnections() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		p.mu.Lock()
		now := time.Now()
		for key, client := range p.clients {
			client.mu.Lock()
			if now.Sub(client.lastUsed) > p.idleTimeout {
				client.client.Close()
				delete(p.clients, key)
				p.logger.Infof("Cleaned up idle SSH connection: %s", key)
			}
			client.mu.Unlock()
		}
		p.mu.Unlock()
	}
}

// Close 关闭所有连接
func (p *SSHClientPool) Close() {
	p.mu.Lock()
	defer p.mu.Unlock()

	for key, client := range p.clients {
		client.mu.Lock()
		client.client.Close()
		client.mu.Unlock()
		delete(p.clients, key)
	}

	p.logger.Info("SSH client pool closed")
}

// Stats 获取连接池统计信息
func (p *SSHClientPool) Stats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return map[string]interface{}{
		"total_connections": len(p.clients),
		"max_pool_size":     p.maxPoolSize,
		"idle_timeout":      p.idleTimeout.String(),
	}
}
