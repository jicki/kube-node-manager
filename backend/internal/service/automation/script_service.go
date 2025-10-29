package automation

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// ScriptService 脚本管理服务
type ScriptService struct {
	db            *gorm.DB
	logger        *logger.Logger
	progressSvc   *progress.Service
	sshSvc        *SSHService
	scriptDir     string
	maxScriptSize int64 // 最大脚本大小（字节）
}

// ScriptExecuteConfig 脚本执行配置
type ScriptExecuteConfig struct {
	ScriptID     uint
	ClusterName  string
	TargetNodes  []string
	Parameters   map[string]string
	CredentialID uint
	Timeout      time.Duration
	Concurrent   int
}

// ScriptValidator 脚本验证器
type ScriptValidator struct {
	language string
}

// NewScriptService 创建脚本服务
func NewScriptService(
	db *gorm.DB,
	logger *logger.Logger,
	progressSvc *progress.Service,
	sshSvc *SSHService,
	scriptDir string,
) *ScriptService {
	// 确保脚本目录存在
	if err := os.MkdirAll(scriptDir, 0755); err != nil {
		logger.Errorf("Failed to create script directory: %v", err)
	}

	return &ScriptService{
		db:            db,
		logger:        logger,
		progressSvc:   progressSvc,
		sshSvc:        sshSvc,
		scriptDir:     scriptDir,
		maxScriptSize: 1024 * 1024, // 1MB
	}
}

// CreateScript 创建脚本
func (s *ScriptService) CreateScript(script *model.Script) error {
	// 验证脚本内容
	if err := s.validateScript(script); err != nil {
		return fmt.Errorf("script validation failed: %w", err)
	}

	// 设置初始版本
	if script.Version == 0 {
		script.Version = 1
	}

	return s.db.Create(script).Error
}

// UpdateScript 更新脚本
func (s *ScriptService) UpdateScript(id uint, updates *model.Script) error {
	var script model.Script
	if err := s.db.First(&script, id).Error; err != nil {
		return err
	}

	// 如果是内置脚本，不允许修改
	if script.IsBuiltin {
		return fmt.Errorf("cannot modify builtin script")
	}

	// 如果内容发生变化，创建新版本
	if updates.Content != "" && updates.Content != script.Content {
		// 验证新内容
		validator := &ScriptValidator{language: script.Language}
		if err := validator.Validate(updates.Content); err != nil {
			return fmt.Errorf("script validation failed: %w", err)
		}

		// 增加版本号
		updates.Version = script.Version + 1
	}

	return s.db.Model(&script).Updates(updates).Error
}

// DeleteScript 删除脚本
func (s *ScriptService) DeleteScript(id uint) error {
	var script model.Script
	if err := s.db.First(&script, id).Error; err != nil {
		return err
	}

	// 内置脚本不能删除
	if script.IsBuiltin {
		return fmt.Errorf("cannot delete builtin script")
	}

	return s.db.Delete(&script).Error
}

// GetScript 获取脚本详情
func (s *ScriptService) GetScript(id uint) (*model.Script, error) {
	var script model.Script
	if err := s.db.First(&script, id).Error; err != nil {
		return nil, err
	}
	return &script, nil
}

// ListScripts 列出脚本
func (s *ScriptService) ListScripts(scriptType string, category string, limit int, offset int) ([]model.Script, int64, error) {
	var scripts []model.Script
	var total int64

	query := s.db.Model(&model.Script{})

	if scriptType != "" {
		query = query.Where("type = ?", scriptType)
	}

	if category != "" {
		query = query.Where("category = ?", category)
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
		Find(&scripts).Error; err != nil {
		return nil, 0, err
	}

	return scripts, total, nil
}

// ExecuteScript 执行脚本
func (s *ScriptService) ExecuteScript(ctx context.Context, config *ScriptExecuteConfig, userID uint) (string, error) {
	// 获取脚本
	script, err := s.GetScript(config.ScriptID)
	if err != nil {
		return "", fmt.Errorf("failed to get script: %w", err)
	}

	// 生成任务 ID
	taskID := fmt.Sprintf("script_exec_%d_%d", userID, time.Now().UnixNano())

	// 创建执行记录
	execution := &model.ScriptExecution{
		TaskID:      taskID,
		ScriptID:    script.ID,
		ScriptName:  script.Name,
		ClusterName: config.ClusterName,
		TargetNodes: marshalJSON(config.TargetNodes),
		Parameters:  marshalJSON(config.Parameters),
		Status:      "pending",
		UserID:      userID,
	}

	if err := s.db.Create(execution).Error; err != nil {
		return "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// 异步执行
	go s.executeScriptAsync(ctx, config, script, execution)

	return taskID, nil
}

// executeScriptAsync 异步执行脚本
func (s *ScriptService) executeScriptAsync(ctx context.Context, config *ScriptExecuteConfig, script *model.Script, execution *model.ScriptExecution) {
	startTime := time.Now()

	// 更新状态为 running
	s.db.Model(execution).Updates(map[string]interface{}{
		"status":     "running",
		"start_time": startTime,
	})

	// 创建进度任务
	if s.progressSvc != nil {
		s.progressSvc.CreateTask(execution.TaskID, "script_execution", len(config.TargetNodes), execution.UserID)
	}

	// 注入参数
	scriptContent, err := s.injectParameters(script.Content, config.Parameters)
	if err != nil {
		s.failExecution(execution, fmt.Errorf("parameter injection failed: %w", err))
		return
	}

	// 远程批量执行脚本
	results := s.executeRemotely(ctx, config, scriptContent, script.Language)

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
	execution.EndTime = &endTime
	execution.Duration = int(endTime.Sub(startTime).Seconds())
	execution.Output = marshalJSON(results)
	execution.SuccessCount = successCount
	execution.FailedCount = failedCount

	if failedCount == 0 {
		execution.Status = "completed"
	} else if successCount == 0 {
		execution.Status = "failed"
	} else {
		execution.Status = "partial"
	}

	if err := s.db.Save(execution).Error; err != nil {
		s.logger.Errorf("Failed to save execution record: %v", err)
	}

	// 发送完成消息
	if s.progressSvc != nil {
		if execution.Status == "failed" {
			s.progressSvc.ErrorTask(execution.TaskID, fmt.Errorf("script execution failed"), execution.UserID)
		} else {
			s.progressSvc.CompleteTask(execution.TaskID, execution.UserID)
		}
	}

	s.logger.Infof("Script execution completed: TaskID=%s, Status=%s, Success=%d, Failed=%d",
		execution.TaskID, execution.Status, successCount, failedCount)
}

// executeLocally 本地执行脚本
func (s *ScriptService) executeLocally(ctx context.Context, content string, scriptType string, timeout time.Duration) *SSHExecuteResult {
	startTime := time.Now()

	result := &SSHExecuteResult{
		Host:    "localhost",
		Command: fmt.Sprintf("[%s script]", scriptType),
	}

	// 创建临时脚本文件
	tmpFile, err := s.createTempScriptFile(content, scriptType)
	if err != nil {
		result.Error = fmt.Sprintf("failed to create temp file: %v", err)
		return result
	}
	defer os.Remove(tmpFile)

	// 构建执行命令
	var cmd *exec.Cmd
	switch scriptType {
	case "shell":
		cmd = exec.CommandContext(ctx, "/bin/bash", tmpFile)
	case "python":
		cmd = exec.CommandContext(ctx, "python3", tmpFile)
	default:
		result.Error = fmt.Sprintf("unsupported script type: %s", scriptType)
		return result
	}

	// 执行命令（带超时）
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(startTime)
	result.Stdout = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.Error = err.Error()
		}
	} else {
		result.ExitCode = 0
	}

	return result
}

// executeRemotely 远程批量执行脚本
func (s *ScriptService) executeRemotely(ctx context.Context, config *ScriptExecuteConfig, content string, language string) []*SSHExecuteResult {
	results := make([]*SSHExecuteResult, len(config.TargetNodes))
	var wg sync.WaitGroup

	// 限制并发数
	semaphore := make(chan struct{}, config.Concurrent)

	// 生成 taskID（在执行记录中应该已经有了）
	taskID := fmt.Sprintf("script_exec_%d", time.Now().UnixNano())

	for i, node := range config.TargetNodes {
		wg.Add(1)
		go func(index int, nodeName string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 在远程节点创建临时脚本文件并执行
			result := s.executeOnRemoteNode(ctx, nodeName, content, language, config)
			results[index] = result

			// 更新进度
			if s.progressSvc != nil {
				s.progressSvc.UpdateProgress(taskID, index+1, nodeName, 0)
			}
		}(i, node)
	}

	wg.Wait()
	return results
}

// executeOnRemoteNode 在远程节点执行脚本
func (s *ScriptService) executeOnRemoteNode(ctx context.Context, node string, content string, language string, config *ScriptExecuteConfig) *SSHExecuteResult {
	// 构建远程执行命令
	var command string
	switch language {
	case "shell", "bash":
		// 直接通过 bash -c 执行
		command = fmt.Sprintf("bash -c %s", shellQuote(content))
	case "python":
		// 通过 python3 -c 执行
		command = fmt.Sprintf("python3 -c %s", shellQuote(content))
	default:
		return &SSHExecuteResult{
			Host:  node,
			Error: fmt.Sprintf("unsupported script language: %s", language),
		}
	}

	// 通过 SSH 服务执行
	credential, err := s.sshSvc.credentialMgr.GetCredential(config.CredentialID)
	if err != nil {
		return &SSHExecuteResult{
			Host:  node,
			Error: fmt.Sprintf("failed to get credential: %v", err),
		}
	}

	client, err := s.sshSvc.clientPool.GetClient(node, credential)
	if err != nil {
		return &SSHExecuteResult{
			Host:  node,
			Error: fmt.Sprintf("failed to connect: %v", err),
		}
	}

	result, err := client.Execute(command, config.Timeout)
	if err != nil && result.Error == "" {
		result.Error = err.Error()
	}

	return result
}

// injectParameters 注入参数到脚本
func (s *ScriptService) injectParameters(content string, params map[string]string) (string, error) {
	result := content

	for key, value := range params {
		// 替换 ${PARAM_NAME} 格式的占位符
		placeholder := fmt.Sprintf("${%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}

	// 检查是否还有未替换的占位符
	re := regexp.MustCompile(`\$\{[A-Z_]+\}`)
	if matches := re.FindAllString(result, -1); len(matches) > 0 {
		return "", fmt.Errorf("missing parameters: %v", matches)
	}

	return result, nil
}

// validateScript 验证脚本
func (s *ScriptService) validateScript(script *model.Script) error {
	// 检查脚本大小
	if int64(len(script.Content)) > s.maxScriptSize {
		return fmt.Errorf("script size exceeds maximum allowed size of %d bytes", s.maxScriptSize)
	}

	// 验证脚本语法
	validator := &ScriptValidator{language: script.Language}
	return validator.Validate(script.Content)
}

// Validate 验证脚本语法
func (v *ScriptValidator) Validate(content string) error {
	switch v.language {
	case "shell", "bash":
		return v.validateShellScript(content)
	case "python":
		return v.validatePythonScript(content)
	default:
		return fmt.Errorf("unsupported script language: %s", v.language)
	}
}

// validateShellScript 验证 Shell 脚本
func (v *ScriptValidator) validateShellScript(content string) error {
	// 基本语法检查：使用 bash -n 进行语法验证
	cmd := exec.Command("bash", "-n")
	cmd.Stdin = strings.NewReader(content)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("shell syntax error: %s", string(output))
	}
	return nil
}

// validatePythonScript 验证 Python 脚本
func (v *ScriptValidator) validatePythonScript(content string) error {
	// 基本语法检查：使用 python3 -m py_compile
	cmd := exec.Command("python3", "-m", "py_compile", "-")
	cmd.Stdin = strings.NewReader(content)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("python syntax error: %s", string(output))
	}
	return nil
}

// createTempScriptFile 创建临时脚本文件
func (s *ScriptService) createTempScriptFile(content string, language string) (string, error) {
	var ext string
	switch language {
	case "shell", "bash":
		ext = ".sh"
	case "python":
		ext = ".py"
	default:
		ext = ".txt"
	}

	tmpFile, err := os.CreateTemp(s.scriptDir, fmt.Sprintf("script_*%s", ext))
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if _, err := tmpFile.WriteString(content); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	// 设置可执行权限
	if err := os.Chmod(tmpFile.Name(), 0755); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// failExecution 标记执行失败
func (s *ScriptService) failExecution(execution *model.ScriptExecution, err error) {
	endTime := time.Now()
	execution.Status = "failed"
	execution.EndTime = &endTime

	if execution.StartTime != nil {
		execution.Duration = int(endTime.Sub(*execution.StartTime).Seconds())
	}

	execution.ErrorMessage = err.Error()
	execution.Output = ""

	s.db.Save(execution)

	if s.progressSvc != nil {
		s.progressSvc.ErrorTask(execution.TaskID, err, execution.UserID)
	}

	s.logger.Errorf("Script execution failed: TaskID=%s, Error=%v", execution.TaskID, err)
}

// GetExecutionStatus 获取执行状态
func (s *ScriptService) GetExecutionStatus(taskID string) (*model.ScriptExecution, error) {
	var execution model.ScriptExecution
	if err := s.db.Where("task_id = ?", taskID).First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListExecutions 列出执行历史
func (s *ScriptService) ListExecutions(scriptID uint, status string, limit int, offset int) ([]model.ScriptExecution, int64, error) {
	var executions []model.ScriptExecution
	var total int64

	query := s.db.Model(&model.ScriptExecution{})

	if scriptID > 0 {
		query = query.Where("script_id = ?", scriptID)
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
		Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// InitializeBuiltinScripts 初始化内置脚本
func (s *ScriptService) InitializeBuiltinScripts() error {
	builtinScripts := []model.Script{
		{
			Name:        "系统信息收集",
			Language:    "shell",
			Category:    "diagnosis",
			Description: "收集系统基本信息，包括CPU、内存、磁盘、网络等",
			Content:     getSystemInfoScript(),
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			Name:        "磁盘清理",
			Language:    "shell",
			Category:    "maintenance",
			Description: "清理系统临时文件、日志文件等",
			Content:     getDiskCleanupScript(),
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			Name:        "日志收集",
			Language:    "shell",
			Category:    "diagnosis",
			Description: "收集系统和应用日志",
			Content:     getLogCollectionScript(),
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
		{
			Name:        "性能诊断",
			Language:    "shell",
			Category:    "diagnosis",
			Description: "诊断系统性能问题，包括CPU、内存、IO等",
			Content:     getPerformanceDiagnosisScript(),
			IsBuiltin:   true,
			IsActive:    true,
			Version:     1,
		},
	}

	for _, script := range builtinScripts {
		// 检查是否已存在
		var existing model.Script
		result := s.db.Where("name = ? AND is_builtin = ?", script.Name, true).First(&existing)

		if result.Error == gorm.ErrRecordNotFound {
			// 不存在，创建新记录
			if err := s.db.Create(&script).Error; err != nil {
				s.logger.Errorf("Failed to create builtin script %s: %v", script.Name, err)
				continue
			}
			s.logger.Infof("Created builtin script: %s", script.Name)
		} else if result.Error == nil {
			// 已存在，检查是否需要更新内容
			if existing.Content != script.Content {
				updates := map[string]interface{}{
					"content":     script.Content,
					"description": script.Description,
					"version":     existing.Version + 1,
				}
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					s.logger.Errorf("Failed to update builtin script %s: %v", script.Name, err)
				} else {
					s.logger.Infof("Updated builtin script: %s", script.Name)
				}
			}
		}
	}

	return nil
}

// shellQuote Shell 命令参数转义
func shellQuote(s string) string {
	return fmt.Sprintf("'%s'", strings.ReplaceAll(s, "'", "'\\''"))
}

// 内置脚本内容

func getSystemInfoScript() string {
	return `#!/bin/bash
# 系统信息收集脚本

echo "=== System Information ==="
echo "Hostname: $(hostname)"
echo "OS: $(cat /etc/os-release | grep PRETTY_NAME | cut -d= -f2)"
echo "Kernel: $(uname -r)"
echo "Uptime: $(uptime -p)"
echo ""

echo "=== CPU Information ==="
lscpu | grep -E "Model name|Socket|Core|Thread"
echo "CPU Usage: $(top -bn1 | grep "Cpu(s)" | awk '{print $2}')%"
echo ""

echo "=== Memory Information ==="
free -h
echo ""

echo "=== Disk Information ==="
df -h
echo ""

echo "=== Network Information ==="
ip -brief addr show
echo ""

echo "=== Docker Information ==="
if command -v docker &> /dev/null; then
    docker version --format '{{.Server.Version}}'
    docker ps --format "table {{.Names}}\t{{.Status}}"
fi
`
}

func getDiskCleanupScript() string {
	return `#!/bin/bash
# 磁盘清理脚本（安全版本）

echo "=== Disk Cleanup Started ==="
echo "Initial disk usage:"
df -h /

# 清理系统临时文件
echo "Cleaning /tmp..."
find /tmp -type f -atime +7 -delete 2>/dev/null || true

# 清理旧日志
echo "Cleaning old logs..."
find /var/log -name "*.log" -type f -mtime +30 -delete 2>/dev/null || true
find /var/log -name "*.gz" -type f -mtime +30 -delete 2>/dev/null || true

# 清理 Docker 资源
if command -v docker &> /dev/null; then
    echo "Cleaning Docker resources..."
    docker system prune -f --volumes
fi

echo "Final disk usage:"
df -h /
echo "=== Disk Cleanup Completed ==="
`
}

func getLogCollectionScript() string {
	return `#!/bin/bash
# 日志收集脚本

COLLECTION_DIR="/tmp/log_collection_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$COLLECTION_DIR"

echo "=== Log Collection Started ==="
echo "Collection directory: $COLLECTION_DIR"

# 收集系统日志
echo "Collecting system logs..."
journalctl --since "24 hours ago" > "$COLLECTION_DIR/system.log" 2>/dev/null || true

# 收集 Kubernetes 日志
if command -v kubectl &> /dev/null; then
    echo "Collecting Kubernetes logs..."
    kubectl get pods --all-namespaces > "$COLLECTION_DIR/k8s_pods.txt" 2>/dev/null || true
fi

# 收集 Docker 日志
if command -v docker &> /dev/null; then
    echo "Collecting Docker logs..."
    docker ps -a > "$COLLECTION_DIR/docker_containers.txt" 2>/dev/null || true
fi

# 打包日志
echo "Creating archive..."
tar -czf "${COLLECTION_DIR}.tar.gz" -C "$(dirname $COLLECTION_DIR)" "$(basename $COLLECTION_DIR)"
rm -rf "$COLLECTION_DIR"

echo "Log collection completed: ${COLLECTION_DIR}.tar.gz"
ls -lh "${COLLECTION_DIR}.tar.gz"
`
}

func getPerformanceDiagnosisScript() string {
	return `#!/bin/bash
# 性能诊断脚本

echo "=== Performance Diagnosis Started ==="
echo "Timestamp: $(date)"
echo ""

# CPU 性能
echo "=== CPU Performance ==="
top -bn1 | head -20
echo ""

# 内存性能
echo "=== Memory Performance ==="
free -h
echo ""
echo "Top Memory Consumers:"
ps aux --sort=-%mem | head -10
echo ""

# 磁盘 IO
echo "=== Disk IO ==="
iostat -x 1 5 2>/dev/null || echo "iostat not available"
echo ""

# 网络性能
echo "=== Network Performance ==="
netstat -i 2>/dev/null || ss -i
echo ""

# 进程信息
echo "=== Top Processes ==="
ps aux --sort=-%cpu | head -10
echo ""

echo "=== Performance Diagnosis Completed ==="
`
}
