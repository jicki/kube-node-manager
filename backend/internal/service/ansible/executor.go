package ansible

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// TaskExecutor 任务执行器
type TaskExecutor struct {
	db              *gorm.DB
	logger          *logger.Logger
	runningTasks    map[uint]*RunningTask
	mu              sync.RWMutex
	maxConcurrent   int
	wsHub           interface{} // WebSocket Hub for log streaming
	inventorySvc    *InventoryService
	sshKeySvc       *SSHKeyService
	workDir         string // 工作目录
}

// RunningTask 正在执行的任务
type RunningTask struct {
	TaskID       uint
	Cmd          *exec.Cmd
	Cancel       context.CancelFunc
	StartTime    time.Time
	LogChannel   chan *model.AnsibleLog
	LogBuffer    *strings.Builder // 日志聚合缓冲区
	LogMutex     sync.Mutex       // 保护 LogBuffer
	LogSize      int64            // 当前日志大小
	MaxLogSize   int64            // 最大日志大小 (10MB)
	SSHKeyFile   string           // SSH 密钥临时文件路径
}

// NewTaskExecutor 创建任务执行器实例
func NewTaskExecutor(db *gorm.DB, logger *logger.Logger, inventorySvc *InventoryService, sshKeySvc *SSHKeyService, wsHub interface{}) *TaskExecutor {
	// 创建工作目录
	workDir := filepath.Join(os.TempDir(), "kube-node-manager-ansible")
	if err := os.MkdirAll(workDir, 0755); err != nil {
		logger.Errorf("Failed to create work directory: %v", err)
	}

	return &TaskExecutor{
		db:            db,
		logger:        logger,
		runningTasks:  make(map[uint]*RunningTask),
		maxConcurrent: 5, // 最多同时执行 5 个任务
		wsHub:         wsHub,
		inventorySvc:  inventorySvc,
		sshKeySvc:     sshKeySvc,
		workDir:       workDir,
	}
}

// ExecuteTask 执行任务
func (e *TaskExecutor) ExecuteTask(taskID uint) error {
	e.mu.Lock()
	// 检查并发数
	if len(e.runningTasks) >= e.maxConcurrent {
		e.mu.Unlock()
		return fmt.Errorf("maximum concurrent tasks limit reached (%d)", e.maxConcurrent)
	}

	// 检查任务是否已经在运行
	if _, exists := e.runningTasks[taskID]; exists {
		e.mu.Unlock()
		return fmt.Errorf("task is already running")
	}
	e.mu.Unlock()

	// 获取任务
	var task model.AnsibleTask
	if err := e.db.Preload("Inventory").First(&task, taskID).Error; err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查任务状态
	if task.Status != model.AnsibleTaskStatusPending {
		return fmt.Errorf("task is not in pending status")
	}

	// 创建上下文（带超时控制）
	var ctx context.Context
	var cancel context.CancelFunc
	
	if task.TimeoutSeconds > 0 {
		// 设置了超时时间
		ctx, cancel = context.WithTimeout(context.Background(), time.Duration(task.TimeoutSeconds)*time.Second)
		e.logger.Infof("Task %d: timeout set to %d seconds", taskID, task.TimeoutSeconds)
	} else {
		// 没有设置超时时间
		ctx, cancel = context.WithCancel(context.Background())
	}

	// 创建运行任务记录
	runningTask := &RunningTask{
		TaskID:      taskID,
		Cancel:      cancel,
		StartTime:   time.Now(),
		LogChannel:  make(chan *model.AnsibleLog, 100),
		LogBuffer:   &strings.Builder{},
		LogSize:     0,
		MaxLogSize:  10 * 1024 * 1024, // 10MB 日志大小限制
	}

	e.mu.Lock()
	e.runningTasks[taskID] = runningTask
	e.mu.Unlock()

	// 异步执行任务
	go e.executeTaskAsync(ctx, &task, runningTask)

	return nil
}

// executeTaskAsync 异步执行任务
func (e *TaskExecutor) executeTaskAsync(ctx context.Context, task *model.AnsibleTask, runningTask *RunningTask) {
	// 标记任务开始
	task.MarkStarted()
	if err := e.db.Save(task).Error; err != nil {
		e.logger.Errorf("Failed to mark task as started: %v", err)
	}

	// 创建临时文件
	playbookFile, err := e.createPlaybookFile(task)
	if err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to create playbook file: %w", err))
		return
	}
	defer os.Remove(playbookFile)

	inventoryFile, err := e.createInventoryFile(task)
	if err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to create inventory file: %w", err))
		return
	}
	defer os.Remove(inventoryFile)

	// 创建 SSH 密钥文件（如果需要）
	sshKeyFile, err := e.createSSHKeyFile(task)
	if err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to create ssh key file: %w", err))
		return
	}
	if sshKeyFile != "" {
		runningTask.SSHKeyFile = sshKeyFile
		defer os.Remove(sshKeyFile)
	}

	// 构建命令
	cmd := e.buildAnsibleCommand(ctx, playbookFile, inventoryFile, sshKeyFile, task)
	runningTask.Cmd = cmd

	// 启动日志收集
	go e.collectLogs(runningTask)

	// 捕获输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to create stdout pipe: %w", err))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to create stderr pipe: %w", err))
		return
	}

	// 启动命令
	if err := cmd.Start(); err != nil {
		e.handleTaskError(task, runningTask, fmt.Errorf("failed to start ansible command: %w", err))
		return
	}

	// 读取输出
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		e.readOutput(stdout, runningTask, model.AnsibleLogTypeStdout)
	}()

	go func() {
		defer wg.Done()
		e.readOutput(stderr, runningTask, model.AnsibleLogTypeStderr)
	}()

	wg.Wait()

	// 等待命令完成
	err = cmd.Wait()

	// 关闭日志通道
	close(runningTask.LogChannel)

	// 检查是否超时
	isTimedOut := false
	if ctx.Err() == context.DeadlineExceeded {
		isTimedOut = true
		e.logger.Warnf("Task %d exceeded timeout limit (%d seconds)", task.ID, task.TimeoutSeconds)
	}

	// 解析执行结果
	success := err == nil
	var errorMsg string
	if err != nil {
		if isTimedOut {
			errorMsg = fmt.Sprintf("任务执行超时（超过 %d 秒）", task.TimeoutSeconds)
		} else {
			errorMsg = err.Error()
		}
	}

	// 先保存完整日志到任务（必须在 parseTaskStats 之前）
	runningTask.LogMutex.Lock()
	task.FullLog = runningTask.LogBuffer.String()
	task.LogSize = runningTask.LogSize
	runningTask.LogMutex.Unlock()

	// 再解析统计信息（从 task.FullLog 中）
	e.parseTaskStats(task)

	// 标记是否超时
	task.IsTimedOut = isTimedOut

	// 标记任务完成
	task.MarkCompleted(success, errorMsg)
	if err := e.db.Save(task).Error; err != nil {
		e.logger.Errorf("Failed to save task completion: %v", err)
	}

	e.logger.Infof("Task %d completed, log size: %d bytes (%d KB)", 
		task.ID, task.LogSize, task.LogSize/1024)

	// 移除运行任务记录
	e.mu.Lock()
	delete(e.runningTasks, task.ID)
	e.mu.Unlock()

	if success {
		e.logger.Infof("Task %d completed successfully", task.ID)
	} else {
		e.logger.Errorf("Task %d failed: %v", task.ID, errorMsg)
		
		// 检查是否需要重试
		e.checkAndRetryTask(task)
	}
}

// checkAndRetryTask 检查并重试失败的任务
func (e *TaskExecutor) checkAndRetryTask(task *model.AnsibleTask) {
	// 检查是否配置了重试策略
	if task.RetryPolicy == nil || !task.RetryPolicy.RetryOnError {
		return
	}

	// 检查是否达到最大重试次数
	if task.RetryCount >= task.MaxRetries {
		e.logger.Infof("Task %d reached maximum retry count (%d), no more retries", task.ID, task.MaxRetries)
		return
	}

	// 如果设置了 MaxRetries，更新任务的 MaxRetries 字段
	if task.MaxRetries == 0 && task.RetryPolicy.MaxRetries > 0 {
		task.MaxRetries = task.RetryPolicy.MaxRetries
	}

	// 计算重试间隔
	retryInterval := time.Duration(task.RetryPolicy.RetryInterval) * time.Second
	if retryInterval == 0 {
		retryInterval = 30 * time.Second // 默认30秒
	}

	e.logger.Infof("Task %d will be retried after %v (retry %d/%d)", 
		task.ID, retryInterval, task.RetryCount+1, task.MaxRetries)

	// 延迟后重试
	time.AfterFunc(retryInterval, func() {
		e.retryTask(task.ID)
	})
}

// retryTask 重试任务
func (e *TaskExecutor) retryTask(taskID uint) {
	// 获取任务
	var task model.AnsibleTask
	if err := e.db.Preload("Inventory").First(&task, taskID).Error; err != nil {
		e.logger.Errorf("Failed to get task %d for retry: %v", taskID, err)
		return
	}

	// 更新重试次数
	task.RetryCount++
	task.Status = model.AnsibleTaskStatusPending
	task.StartedAt = nil
	task.FinishedAt = nil
	task.ErrorMsg = ""
	task.FullLog = ""
	task.LogSize = 0

	if err := e.db.Save(&task).Error; err != nil {
		e.logger.Errorf("Failed to update task %d for retry: %v", taskID, err)
		return
	}

	e.logger.Infof("Retrying task %d (attempt %d/%d)", taskID, task.RetryCount, task.MaxRetries)

	// 执行任务
	if err := e.ExecuteTask(taskID); err != nil {
		e.logger.Errorf("Failed to execute retry for task %d: %v", taskID, err)
		
		// 更新任务状态为失败
		task.Status = model.AnsibleTaskStatusFailed
		task.ErrorMsg = err.Error()
		if err := e.db.Save(&task).Error; err != nil {
			e.logger.Errorf("Failed to update task status: %v", err)
		}
	}
}

// CancelTask 取消任务
func (e *TaskExecutor) CancelTask(taskID uint) error {
	e.mu.Lock()
	runningTask, exists := e.runningTasks[taskID]
	e.mu.Unlock()

	if !exists {
		return fmt.Errorf("task is not running")
	}

	// 取消上下文
	runningTask.Cancel()

	// 如果命令进程存在，杀死它
	if runningTask.Cmd != nil && runningTask.Cmd.Process != nil {
		if err := runningTask.Cmd.Process.Kill(); err != nil {
			e.logger.Warningf("Failed to kill task %d process: %v", taskID, err)
		}
	}

	// 更新任务状态
	var task model.AnsibleTask
	if err := e.db.First(&task, taskID).Error; err == nil {
		task.MarkCancelled()
		if err := e.db.Save(&task).Error; err != nil {
			e.logger.Errorf("Failed to save task cancellation: %v", err)
		}
	}

	e.logger.Infof("Task %d cancelled", taskID)
	return nil
}

// ContinueBatchExecution 继续批次执行
func (e *TaskExecutor) ContinueBatchExecution(taskID uint) error {
	// 获取任务
	var task model.AnsibleTask
	if err := e.db.First(&task, taskID).Error; err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// 检查批次配置
	if !task.IsBatchEnabled() {
		return fmt.Errorf("batch execution not enabled")
	}

	// 检查是否还有剩余批次
	if task.CurrentBatch >= task.TotalBatches {
		// 所有批次已完成
		task.MarkBatchCompleted()
		task.Status = model.AnsibleTaskStatusSuccess
		finishedAt := time.Now()
		task.FinishedAt = &finishedAt
		duration := int(time.Since(*task.StartedAt).Seconds())
		task.Duration = duration
		
		if err := e.db.Save(&task).Error; err != nil {
			e.logger.Errorf("Failed to update task: %v", err)
		}
		
		e.logger.Infof("Task %d: All batches completed", taskID)
		return nil
	}

	e.logger.Infof("Task %d: Continuing to batch %d/%d", taskID, task.CurrentBatch, task.TotalBatches)
	
	// 注意：实际的批次执行逻辑在 ExecuteTask 中处理
	// 这里只是记录日志，实际执行会在 ansible-playbook 命令中通过 serial 参数控制
	
	return nil
}

// IsTaskRunning 检查任务是否正在运行
func (e *TaskExecutor) IsTaskRunning(taskID uint) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	_, exists := e.runningTasks[taskID]
	return exists
}

// GetRunningTasksCount 获取正在运行的任务数量
func (e *TaskExecutor) GetRunningTasksCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.runningTasks)
}

// createPlaybookFile 创建 playbook 临时文件
func (e *TaskExecutor) createPlaybookFile(task *model.AnsibleTask) (string, error) {
	filename := filepath.Join(e.workDir, fmt.Sprintf("playbook-%d-%d.yml", task.ID, time.Now().Unix()))
	
	if err := os.WriteFile(filename, []byte(task.PlaybookContent), 0644); err != nil {
		return "", err
	}

	e.logger.Infof("Created playbook file for task %d: %s", task.ID, filename)
	return filename, nil
}

// createInventoryFile 创建 inventory 临时文件
func (e *TaskExecutor) createInventoryFile(task *model.AnsibleTask) (string, error) {
	if task.InventoryID == nil {
		return "", fmt.Errorf("task has no inventory")
	}

	inventory, err := e.inventorySvc.GetInventory(*task.InventoryID)
	if err != nil {
		return "", err
	}

	filename := filepath.Join(e.workDir, fmt.Sprintf("inventory-%d-%d.ini", task.ID, time.Now().Unix()))
	
	if err := os.WriteFile(filename, []byte(inventory.Content), 0644); err != nil {
		return "", err
	}

	e.logger.Infof("Created inventory file for task %d: %s", task.ID, filename)
	return filename, nil
}

// createSSHKeyFile 创建 SSH 密钥临时文件
func (e *TaskExecutor) createSSHKeyFile(task *model.AnsibleTask) (string, error) {
	// 获取清单信息
	if task.InventoryID == nil {
		e.logger.Warningf("Task %d has no inventory ID, skipping SSH key file creation", task.ID)
		return "", nil
	}

	inventory, err := e.inventorySvc.GetInventory(*task.InventoryID)
	if err != nil {
		e.logger.Errorf("Failed to get inventory %d for task %d: %v", *task.InventoryID, task.ID, err)
		return "", err
	}

	// 如果清单没有关联 SSH 密钥，返回空字符串
	if inventory.SSHKeyID == nil {
		e.logger.Warningf("Task %d: Inventory %d (%s) has no SSH key associated, Ansible will use default authentication", 
			task.ID, inventory.ID, inventory.Name)
		return "", nil
	}

	e.logger.Infof("Task %d: Using SSH key ID %d for inventory %d (%s)", 
		task.ID, *inventory.SSHKeyID, inventory.ID, inventory.Name)

	// 获取解密后的 SSH 密钥
	sshKey, err := e.sshKeySvc.GetDecryptedByID(*inventory.SSHKeyID)
	if err != nil {
		e.logger.Errorf("Failed to get SSH key %d for task %d: %v", *inventory.SSHKeyID, task.ID, err)
		return "", fmt.Errorf("failed to get ssh key: %w", err)
	}

	e.logger.Infof("Task %d: Retrieved SSH key %d (%s) - Type: %s, Username: %s", 
		task.ID, sshKey.ID, sshKey.Name, sshKey.Type, sshKey.Username)

	// 如果是密码认证，不需要创建密钥文件
	if sshKey.Type == model.SSHKeyTypePassword {
		e.logger.Infof("Task %d: SSH key is password type, will use password authentication", task.ID)
		// Ansible 使用密码认证需要安装 sshpass
		// 密码将通过环境变量传递（在 buildAnsibleCommand 中处理）
		return "", nil
	}

	// 创建临时密钥文件
	filename := filepath.Join(e.workDir, fmt.Sprintf("ssh-key-%d-%d.pem", task.ID, time.Now().Unix()))
	
	if err := os.WriteFile(filename, []byte(sshKey.PrivateKey), 0600); err != nil {
		e.logger.Errorf("Failed to write SSH key file for task %d: %v", task.ID, err)
		return "", fmt.Errorf("failed to write ssh key file: %w", err)
	}

	e.logger.Infof("Task %d: Created SSH key file: %s (size: %d bytes)", task.ID, filename, len(sshKey.PrivateKey))
	return filename, nil
}

// buildAnsibleCommand 构建 ansible-playbook 命令
func (e *TaskExecutor) buildAnsibleCommand(ctx context.Context, playbookFile, inventoryFile, sshKeyFile string, task *model.AnsibleTask) *exec.Cmd {
	args := []string{
		"-i", inventoryFile,
		playbookFile,
		"-v", // verbose mode
	}

	// 如果启用了 Dry Run 模式，添加 --check 参数
	if task.DryRun {
		args = append(args, "--check")
		e.logger.Infof("Task %d: Running in Dry Run mode (--check), no changes will be made", task.ID)
	}
	
	// 如果启用了分批执行，计算并添加 --limit 参数
	if task.IsBatchEnabled() && task.CurrentBatch > 0 {
		// 构建批次限制（基础实现，使用 ansible 的 batch 参数）
		batchSize := e.calculateBatchSize(task)
		if batchSize > 0 {
			// 使用 ansible-playbook 的 --limit 参数结合 serial 实现分批
			e.logger.Infof("Task %d: Batch execution - batch %d/%d, size: %d", 
				task.ID, task.CurrentBatch, task.TotalBatches, batchSize)
		}
	}

	// 如果有 SSH 密钥文件，添加 --private-key 参数
	if sshKeyFile != "" {
		args = append(args, "--private-key", sshKeyFile)
		e.logger.Infof("Task %d: Ansible will use SSH key file: %s", task.ID, sshKeyFile)
	} else {
		e.logger.Warningf("Task %d: No SSH key file provided, Ansible will use default authentication", task.ID)
	}

	// 添加额外变量
	if len(task.ExtraVars) > 0 {
		extraVarsJSON, _ := json.Marshal(task.ExtraVars)
		args = append(args, "--extra-vars", string(extraVarsJSON))
	}
	
	// 如果启用了分批执行，添加 serial 参数到 extra-vars
	if task.IsBatchEnabled() {
		batchSize := e.calculateBatchSize(task)
		if batchSize > 0 {
			// 通过 extra-vars 传递 serial 参数
			serialVar := map[string]interface{}{
				"ansible_serial": batchSize,
			}
			serialJSON, _ := json.Marshal(serialVar)
			args = append(args, "--extra-vars", string(serialJSON))
			e.logger.Infof("Task %d: Setting ansible_serial to %d for batch execution", task.ID, batchSize)
		}
	}

	cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
	cmd.Dir = e.workDir

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"ANSIBLE_STDOUT_CALLBACK=default",
		"ANSIBLE_REMOTE_TMP=/tmp/.ansible-${USER}/tmp", // 使用 /tmp 避免 home 目录权限问题
	)

	// 记录完整的命令（用于调试）
	cmdString := "ansible-playbook " + strings.Join(args, " ")
	e.logger.Infof("Task %d: Executing command: %s", task.ID, cmdString)

	return cmd
}

// calculateBatchSize 计算批次大小
func (e *TaskExecutor) calculateBatchSize(task *model.AnsibleTask) int {
	if task.BatchConfig == nil || !task.BatchConfig.Enabled {
		return 0
	}
	
	// 优先使用固定数量
	if task.BatchConfig.BatchSize > 0 {
		return task.BatchConfig.BatchSize
	}
	
	// 使用百分比计算
	if task.BatchConfig.BatchPercent > 0 && task.HostsTotal > 0 {
		size := (task.HostsTotal * task.BatchConfig.BatchPercent) / 100
		if size < 1 {
			size = 1 // 至少执行1台主机
		}
		return size
	}
	
	return 0
}

// readOutput 读取命令输出
func (e *TaskExecutor) readOutput(reader io.Reader, runningTask *RunningTask, logType model.AnsibleLogType) {
	scanner := bufio.NewScanner(reader)
	lineNumber := 1

	for scanner.Scan() {
		line := scanner.Text()
		
		// 创建日志记录
		log := &model.AnsibleLog{
			TaskID:     runningTask.TaskID,
			LogType:    logType,
			Content:    line,
			LineNumber: lineNumber,
			CreatedAt:  time.Now(),
		}

		// 发送到日志通道
		select {
		case runningTask.LogChannel <- log:
		default:
			e.logger.Warningf("Log channel full for task %d, dropping log line", runningTask.TaskID)
		}

		lineNumber++
	}

	if err := scanner.Err(); err != nil {
		e.logger.Errorf("Error reading output for task %d: %v", runningTask.TaskID, err)
	}
}

// collectLogs 收集并保存日志
func (e *TaskExecutor) collectLogs(runningTask *RunningTask) {
	importantLogs := make([]*model.AnsibleLog, 0, 10) // 只保存重要日志到数据库
	
	for {
		select {
		case log, ok := <-runningTask.LogChannel:
			if !ok {
				// 通道关闭，保存重要日志
				if len(importantLogs) > 0 {
					e.saveImportantLogs(importantLogs)
				}
				return
			}

			// 写入日志聚合缓冲区
			runningTask.LogMutex.Lock()
			if runningTask.LogSize < runningTask.MaxLogSize {
				logLine := fmt.Sprintf("[%s] %s\n", log.LogType, log.Content)
				runningTask.LogBuffer.WriteString(logLine)
				runningTask.LogSize += int64(len(logLine))
			} else if runningTask.LogSize == runningTask.MaxLogSize {
				// 达到限制，记录一次警告
				warningLine := "[SYSTEM] Log size limit reached (10MB), truncating further logs\n"
				runningTask.LogBuffer.WriteString(warningLine)
				runningTask.LogSize++ // 避免重复写入警告
			}
			runningTask.LogMutex.Unlock()

			// 推送到 WebSocket（仍然实时推送）
			e.pushLogToWebSocket(log)

			// 只保留重要的日志（错误、RECAP）到数据库
			if e.isImportantLog(log) {
				importantLogs = append(importantLogs, log)
				if len(importantLogs) >= 10 {
					e.saveImportantLogs(importantLogs)
					importantLogs = importantLogs[:0]
				}
			}
		}
	}
}

// isImportantLog 判断是否是重要日志（需要保存到数据库）
func (e *TaskExecutor) isImportantLog(log *model.AnsibleLog) bool {
	if log.LogType == model.AnsibleLogTypeStderr {
		return true // 所有错误输出都保留
	}
	
	content := strings.ToLower(log.Content)
	// 保留包含特定关键字的日志
	keywords := []string{"fatal", "error", "failed", "unreachable", "recap", "play recap"}
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			return true
		}
	}
	
	return false
}

// saveImportantLogs 保存重要日志到数据库
func (e *TaskExecutor) saveImportantLogs(logs []*model.AnsibleLog) {
	if len(logs) == 0 {
		return
	}

	if err := e.db.Create(&logs).Error; err != nil {
		e.logger.Errorf("Failed to save important logs: %v", err)
	}
}

// pushLogToWebSocket 推送单条日志到 WebSocket
func (e *TaskExecutor) pushLogToWebSocket(log *model.AnsibleLog) {
	if e.wsHub == nil {
		return
	}

	// 使用类型断言获取 WebSocket Hub 的方法
	type WSHub interface {
		BroadcastToTask(taskID uint, message interface{})
	}

	if hub, ok := e.wsHub.(WSHub); ok {
		hub.BroadcastToTask(log.TaskID, map[string]interface{}{
			"type":    "log",
			"task_id": log.TaskID,
			"log":     log,
		})
	}
}

// saveLogs 保存日志到数据库（保留兼容性）
func (e *TaskExecutor) saveLogs(logs []*model.AnsibleLog) {
	if len(logs) == 0 {
		return
	}

	if err := e.db.Create(&logs).Error; err != nil {
		e.logger.Errorf("Failed to save logs: %v", err)
	}

	// 推送日志到 WebSocket（如果有）
	e.pushLogsToWebSocket(logs)
}

// pushLogsToWebSocket 推送日志到 WebSocket
func (e *TaskExecutor) pushLogsToWebSocket(logs []*model.AnsibleLog) {
	if e.wsHub == nil {
		return
	}

	// 使用类型断言获取 WebSocket Hub 的方法
	type WSHub interface {
		BroadcastToTask(taskID uint, message interface{})
	}

	if hub, ok := e.wsHub.(WSHub); ok {
		for _, log := range logs {
			hub.BroadcastToTask(log.TaskID, log)
		}
	}
}

// parseTaskStats 解析任务统计信息
func (e *TaskExecutor) parseTaskStats(task *model.AnsibleTask) {
	// 优先从完整日志中解析统计信息
	logContent := task.FullLog
	
	// 如果 FullLog 为空，则尝试从数据库查询日志（向后兼容）
	if logContent == "" {
		var logs []model.AnsibleLog
		if err := e.db.Where("task_id = ?", task.ID).
			Order("line_number DESC").
			Limit(50).
			Find(&logs).Error; err != nil {
			e.logger.Warningf("Failed to get task logs for stats parsing: %v", err)
			return
		}

		// 构建日志内容
		var logBuffer bytes.Buffer
		for i := len(logs) - 1; i >= 0; i-- {
			logBuffer.WriteString(logs[i].Content + "\n")
		}
		logContent = logBuffer.String()
	}

	if logContent == "" {
		e.logger.Warningf("Task %d: No log content available for stats parsing", task.ID)
		return
	}

	// 查找 PLAY RECAP 部分
	lines := strings.Split(logContent, "\n")
	var recapBuffer bytes.Buffer
	inRecap := false

	for _, line := range lines {
		if strings.Contains(line, "PLAY RECAP") {
			inRecap = true
			continue
		}

		if inRecap && strings.TrimSpace(line) != "" {
			// 检查是否还在 RECAP 部分（通常 RECAP 后面会有空行或新的部分）
			if strings.HasPrefix(line, "TASK") || strings.HasPrefix(line, "PLAY") {
				break
			}
			recapBuffer.WriteString(line + "\n")
		}
	}

	recapText := recapBuffer.String()
	if recapText == "" {
		e.logger.Warningf("Task %d: No RECAP section found in logs", task.ID)
		return
	}

	// 解析统计信息
	// 格式示例: hostname : ok=2 changed=1 unreachable=0 failed=0 skipped=0 rescued=0 ignored=0
	re := regexp.MustCompile(`(\S+)\s*:\s*ok=(\d+)\s+changed=(\d+)\s+unreachable=(\d+)\s+failed=(\d+)\s+skipped=(\d+)`)
	matches := re.FindAllStringSubmatch(recapText, -1)

	if len(matches) == 0 {
		e.logger.Warningf("Task %d: No host stats found in RECAP section", task.ID)
		return
	}

	totalOk := 0
	totalFailed := 0
	totalSkipped := 0
	totalUnreachable := 0
	hostsTotal := len(matches)

	for _, match := range matches {
		if len(match) >= 7 {
			ok, _ := strconv.Atoi(match[2])
			unreachable, _ := strconv.Atoi(match[4])
			failed, _ := strconv.Atoi(match[5])
			skipped, _ := strconv.Atoi(match[6])

			totalOk += ok
			totalFailed += failed
			totalSkipped += skipped
			totalUnreachable += unreachable
		}
	}

	// 成功的主机数 = 总主机数 - 失败主机数 - 不可达主机数
	hostsOk := hostsTotal
	hostsFailed := 0
	
	// 如果有任何 failed 或 unreachable 的任务，该主机被视为失败
	for _, match := range matches {
		if len(match) >= 7 {
			unreachable, _ := strconv.Atoi(match[4])
			failed, _ := strconv.Atoi(match[5])
			
			if unreachable > 0 || failed > 0 {
				hostsFailed++
				hostsOk--
			}
		}
	}

	e.logger.Infof("Task %d stats parsed: total=%d, ok=%d, failed=%d, skipped=%d", 
		task.ID, hostsTotal, hostsOk, hostsFailed, totalSkipped)

	// 更新任务统计
	task.UpdateStats(hostsTotal, hostsOk, hostsFailed, totalSkipped)
}

// handleTaskError 处理任务错误
func (e *TaskExecutor) handleTaskError(task *model.AnsibleTask, runningTask *RunningTask, err error) {
	e.logger.Errorf("Task %d error: %v", task.ID, err)
	
	task.MarkCompleted(false, err.Error())
	if err := e.db.Save(task).Error; err != nil {
		e.logger.Errorf("Failed to save task error: %v", err)
	}

	e.mu.Lock()
	delete(e.runningTasks, task.ID)
	e.mu.Unlock()
}

// Cleanup 清理工作目录中的临时文件
func (e *TaskExecutor) Cleanup() {
	// 清理超过 24 小时的临时文件
	files, err := os.ReadDir(e.workDir)
	if err != nil {
		e.logger.Errorf("Failed to read work directory: %v", err)
		return
	}

	cutoff := time.Now().Add(-24 * time.Hour)
	removed := 0

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filePath := filepath.Join(e.workDir, file.Name())
			if err := os.Remove(filePath); err != nil {
				e.logger.Warningf("Failed to remove old file %s: %v", filePath, err)
			} else {
				removed++
			}
		}
	}

	if removed > 0 {
		e.logger.Infof("Cleaned up %d old temporary files", removed)
	}
}

