package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// SSHService SSH 命令执行服务
type SSHService struct {
	db             *gorm.DB
	logger         *logger.Logger
	progressSvc    *progress.Service
	clientPool     *SSHClientPool
	credentialMgr  *CredentialManager
	commandChecker *CommandSecurityChecker
}

// SSHExecuteConfig SSH 执行配置
type SSHExecuteConfig struct {
	ClusterName  string
	TargetNodes  []string
	Command      string
	CredentialID uint
	Timeout      time.Duration
	Concurrent   int
}

// SSHExecutionRecord SSH 执行记录
type SSHExecutionRecord struct {
	ID           uint       `gorm:"primarykey" json:"id"`
	TaskID       string     `gorm:"type:varchar(100);uniqueIndex" json:"task_id"`
	ClusterName  string     `gorm:"type:varchar(100);index" json:"cluster_name"`
	TargetNodes  string     `gorm:"type:text" json:"target_nodes"` // JSON 数组
	Command      string     `gorm:"type:text" json:"command"`
	Status       string     `gorm:"type:varchar(20);index" json:"status"`
	StartTime    *time.Time `json:"start_time"`
	EndTime      *time.Time `json:"end_time"`
	Duration     int        `json:"duration"`                 // 秒
	Results      string     `gorm:"type:text" json:"results"` // JSON 格式的结果
	SuccessCount int        `json:"success_count"`
	FailedCount  int        `json:"failed_count"`
	UserID       uint       `gorm:"index" json:"user_id"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// CommandSecurityChecker 命令安全检查器
type CommandSecurityChecker struct {
	dangerousCommands []string
	dangerousPatterns []*regexp.Regexp
}

// NewSSHService 创建 SSH 服务
func NewSSHService(
	db *gorm.DB,
	logger *logger.Logger,
	progressSvc *progress.Service,
	credentialMgr *CredentialManager,
	maxPoolSize int,
	idleTimeout time.Duration,
) *SSHService {
	clientPool := NewSSHClientPool(logger, credentialMgr, maxPoolSize, idleTimeout)

	return &SSHService{
		db:             db,
		logger:         logger,
		progressSvc:    progressSvc,
		clientPool:     clientPool,
		credentialMgr:  credentialMgr,
		commandChecker: NewCommandSecurityChecker(),
	}
}

// NewCommandSecurityChecker 创建命令安全检查器
func NewCommandSecurityChecker() *CommandSecurityChecker {
	// 危险命令列表
	dangerousCommands := []string{
		"rm -rf /",
		"rm -rf /*",
		"mkfs",
		"dd if=/dev/zero",
		":(){ :|:& };:", // Fork bomb
		"mv / /dev/null",
		"> /dev/sda",
		"wget", // 可能下载恶意脚本
		"curl", // 可能下载恶意脚本
	}

	// 危险模式
	dangerousPatterns := []*regexp.Regexp{
		regexp.MustCompile(`rm\s+-rf\s+/`),
		regexp.MustCompile(`rm\s+-rf\s+\*`),
		regexp.MustCompile(`>\s*/dev/sd[a-z]`),
		regexp.MustCompile(`dd\s+if=/dev/zero`),
		regexp.MustCompile(`mkfs`),
		regexp.MustCompile(`fdisk`),
		regexp.MustCompile(`:\(\)\{`), // Fork bomb
	}

	return &CommandSecurityChecker{
		dangerousCommands: dangerousCommands,
		dangerousPatterns: dangerousPatterns,
	}
}

// CheckCommand 检查命令安全性
func (c *CommandSecurityChecker) CheckCommand(command string) error {
	cmdLower := strings.ToLower(strings.TrimSpace(command))

	// 检查危险命令
	for _, dangerous := range c.dangerousCommands {
		if strings.Contains(cmdLower, dangerous) {
			return fmt.Errorf("dangerous command detected: %s", dangerous)
		}
	}

	// 检查危险模式
	for _, pattern := range c.dangerousPatterns {
		if pattern.MatchString(cmdLower) {
			return fmt.Errorf("dangerous command pattern detected: %s", pattern.String())
		}
	}

	return nil
}

// ExecuteCommand 执行 SSH 命令
func (s *SSHService) ExecuteCommand(ctx context.Context, config *SSHExecuteConfig, userID uint) (string, error) {
	// 安全检查
	if err := s.commandChecker.CheckCommand(config.Command); err != nil {
		return "", fmt.Errorf("command security check failed: %w", err)
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("ssh_exec_%d_%d", userID, time.Now().UnixNano())

	// 创建执行记录
	record := &SSHExecutionRecord{
		TaskID:      taskID,
		ClusterName: config.ClusterName,
		TargetNodes: marshalJSON(config.TargetNodes),
		Command:     config.Command,
		Status:      "pending",
		UserID:      userID,
	}

	if err := s.db.Create(record).Error; err != nil {
		return "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// 异步执行
	go s.executeCommandAsync(ctx, config, record)

	return taskID, nil
}

// executeCommandAsync 异步执行命令
func (s *SSHService) executeCommandAsync(ctx context.Context, config *SSHExecuteConfig, record *SSHExecutionRecord) {
	startTime := time.Now()

	// 更新状态为 running
	s.db.Model(record).Updates(map[string]interface{}{
		"status":     "running",
		"start_time": startTime,
	})

	// 创建进度任务
	if s.progressSvc != nil {
		s.progressSvc.CreateTask(record.TaskID, "ssh_command", len(config.TargetNodes), record.UserID)
	}

	// 获取凭据
	credential, err := s.credentialMgr.GetCredential(config.CredentialID)
	if err != nil {
		s.failExecution(record, fmt.Errorf("failed to get credential: %w", err))
		return
	}

	// 并发执行
	results := s.executeConcurrent(ctx, config, credential)

	// 统计结果
	successCount := 0
	failedCount := 0
	for _, result := range results {
		if result.ExitCode == 0 && result.Error == "" {
			successCount++
		} else {
			failedCount++
		}
	}

	// 保存结果
	endTime := time.Now()
	record.EndTime = &endTime
	record.Duration = int(endTime.Sub(startTime).Seconds())
	record.Results = marshalJSON(results)
	record.SuccessCount = successCount
	record.FailedCount = failedCount

	if failedCount == 0 {
		record.Status = "completed"
	} else if successCount == 0 {
		record.Status = "failed"
	} else {
		record.Status = "partial"
	}

	if err := s.db.Save(record).Error; err != nil {
		s.logger.Errorf("Failed to save execution record: %v", err)
	}

	// 发送完成消息
	if s.progressSvc != nil {
		if record.Status == "failed" {
			s.progressSvc.ErrorTask(record.TaskID, fmt.Errorf("command execution failed"), record.UserID)
		} else {
			s.progressSvc.CompleteTask(record.TaskID, record.UserID)
		}
	}

	s.logger.Infof("SSH command execution completed: TaskID=%s, Status=%s, Success=%d, Failed=%d",
		record.TaskID, record.Status, successCount, failedCount)
}

// executeConcurrent 并发执行命令
func (s *SSHService) executeConcurrent(ctx context.Context, config *SSHExecuteConfig, credential *model.SSHCredential) []*SSHExecuteResult {
	results := make([]*SSHExecuteResult, len(config.TargetNodes))
	var wg sync.WaitGroup

	// 限制并发数
	semaphore := make(chan struct{}, config.Concurrent)

	for i, node := range config.TargetNodes {
		wg.Add(1)
		go func(index int, nodeName string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 执行命令
			result := s.executeOnNode(ctx, nodeName, config.Command, credential, config.Timeout)
			results[index] = result

			// 更新进度
			if s.progressSvc != nil {
				taskID := fmt.Sprintf("ssh_exec_%d", time.Now().UnixNano())
				s.progressSvc.UpdateProgress(taskID, index+1, nodeName, 0) // userID 应该从 record 传递
			}
		}(i, node)
	}

	wg.Wait()
	return results
}

// executeOnNode 在单个节点上执行命令
func (s *SSHService) executeOnNode(ctx context.Context, node, command string, credential *model.SSHCredential, timeout time.Duration) *SSHExecuteResult {
	// 获取 SSH 客户端
	client, err := s.clientPool.GetClient(node, credential)
	if err != nil {
		return &SSHExecuteResult{
			Host:    node,
			Command: command,
			Error:   fmt.Sprintf("failed to connect: %v", err),
		}
	}

	// 执行命令
	result, err := client.Execute(command, timeout)
	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}

	return result
}

// failExecution 标记执行失败
func (s *SSHService) failExecution(record *SSHExecutionRecord, err error) {
	endTime := time.Now()
	record.Status = "failed"
	record.EndTime = &endTime

	if record.StartTime != nil {
		record.Duration = int(endTime.Sub(*record.StartTime).Seconds())
	}

	// 保存错误结果
	errorResult := map[string]string{
		"error": err.Error(),
	}
	record.Results = marshalJSON(errorResult)

	s.db.Save(record)

	// 发送错误消息
	if s.progressSvc != nil {
		s.progressSvc.ErrorTask(record.TaskID, err, record.UserID)
	}

	s.logger.Errorf("SSH execution failed: TaskID=%s, Error=%v", record.TaskID, err)
}

// GetExecutionStatus 获取执行状态
func (s *SSHService) GetExecutionStatus(taskID string) (*SSHExecutionRecord, error) {
	var record SSHExecutionRecord
	if err := s.db.Where("task_id = ?", taskID).First(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// ListExecutions 列出执行历史
func (s *SSHService) ListExecutions(clusterName string, status string, limit int, offset int) ([]SSHExecutionRecord, int64, error) {
	var records []SSHExecutionRecord
	var total int64

	query := s.db.Model(&SSHExecutionRecord{})

	if clusterName != "" {
		query = query.Where("cluster_name = ?", clusterName)
	}

	if status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&records).Error; err != nil {
		return nil, 0, err
	}

	return records, total, nil
}

// Close 关闭服务
func (s *SSHService) Close() {
	if s.clientPool != nil {
		s.clientPool.Close()
	}
}

// TableName 指定表名
func (SSHExecutionRecord) TableName() string {
	return "ssh_executions"
}

// marshalJSON 序列化为 JSON
func marshalJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
