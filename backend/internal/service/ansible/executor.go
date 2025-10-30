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
	SSHKeyFile   string // SSH 密钥临时文件路径
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

	// 创建上下文
	ctx, cancel := context.WithCancel(context.Background())

	// 创建运行任务记录
	runningTask := &RunningTask{
		TaskID:     taskID,
		Cancel:     cancel,
		StartTime:  time.Now(),
		LogChannel: make(chan *model.AnsibleLog, 100),
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

	// 解析执行结果
	success := err == nil
	var errorMsg string
	if err != nil {
		errorMsg = err.Error()
	}

	// 尝试解析统计信息（从日志中）
	e.parseTaskStats(task)

	// 标记任务完成
	task.MarkCompleted(success, errorMsg)
	if err := e.db.Save(task).Error; err != nil {
		e.logger.Errorf("Failed to save task completion: %v", err)
	}

	// 移除运行任务记录
	e.mu.Lock()
	delete(e.runningTasks, task.ID)
	e.mu.Unlock()

	if success {
		e.logger.Infof("Task %d completed successfully", task.ID)
	} else {
		e.logger.Errorf("Task %d failed: %v", task.ID, errorMsg)
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

	cmd := exec.CommandContext(ctx, "ansible-playbook", args...)
	cmd.Dir = e.workDir

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		"ANSIBLE_HOST_KEY_CHECKING=False",
		"ANSIBLE_STDOUT_CALLBACK=default",
	)

	// 记录完整的命令（用于调试）
	cmdString := "ansible-playbook " + strings.Join(args, " ")
	e.logger.Infof("Task %d: Executing command: %s", task.ID, cmdString)

	return cmd
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
	buffer := make([]*model.AnsibleLog, 0, 50) // 批量保存日志
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case log, ok := <-runningTask.LogChannel:
			if !ok {
				// 通道关闭，保存剩余日志
				if len(buffer) > 0 {
					e.saveLogs(buffer)
				}
				return
			}

			buffer = append(buffer, log)

			// 如果缓冲区满了，保存日志
			if len(buffer) >= 50 {
				e.saveLogs(buffer)
				buffer = buffer[:0]
			}

		case <-ticker.C:
			// 定期保存日志
			if len(buffer) > 0 {
				e.saveLogs(buffer)
				buffer = buffer[:0]
			}
		}
	}
}

// saveLogs 保存日志到数据库
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
	// 从最后的日志中查找统计信息
	var logs []model.AnsibleLog
	if err := e.db.Where("task_id = ?", task.ID).
		Order("line_number DESC").
		Limit(50).
		Find(&logs).Error; err != nil {
		e.logger.Warningf("Failed to get task logs for stats parsing: %v", err)
		return
	}

	// 查找 PLAY RECAP 部分
	var recapBuffer bytes.Buffer
	inRecap := false

	for i := len(logs) - 1; i >= 0; i-- {
		line := logs[i].Content
		
		if strings.Contains(line, "PLAY RECAP") {
			inRecap = true
			continue
		}

		if inRecap {
			recapBuffer.WriteString(line + "\n")
		}
	}

	recapText := recapBuffer.String()
	if recapText == "" {
		return
	}

	// 解析统计信息
	// 格式示例: hostname : ok=2 changed=1 unreachable=0 failed=0 skipped=0 rescued=0 ignored=0
	re := regexp.MustCompile(`(\w+)\s*:\s*ok=(\d+)\s+changed=(\d+)\s+unreachable=(\d+)\s+failed=(\d+)\s+skipped=(\d+)`)
	matches := re.FindAllStringSubmatch(recapText, -1)

	totalOk := 0
	totalFailed := 0
	totalSkipped := 0
	hostsTotal := len(matches)

	for _, match := range matches {
		if len(match) >= 7 {
			ok, _ := strconv.Atoi(match[2])
			failed, _ := strconv.Atoi(match[5])
			skipped, _ := strconv.Atoi(match[6])

			totalOk += ok
			totalFailed += failed
			totalSkipped += skipped
		}
	}

	// 更新任务统计
	task.UpdateStats(hostsTotal, totalOk, totalFailed, totalSkipped)
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

