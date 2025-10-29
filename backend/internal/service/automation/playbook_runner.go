package automation

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
)

// PlaybookRunner Playbook 执行器
type PlaybookRunner struct {
	logger        *logger.Logger
	binaryPath    string
	tempDir       string
	timeout       time.Duration
	credentialMgr *CredentialManager
}

// PlaybookRunConfig Playbook 执行配置
type PlaybookRunConfig struct {
	PlaybookID      uint
	PlaybookContent string
	ClusterName     string
	TargetNodes     []string
	ExtraVars       map[string]interface{}
	Tags            []string
	CheckMode       bool
	Verbose         int
	CredentialID    uint
}

// PlaybookResult Playbook 执行结果
type PlaybookResult struct {
	TaskID       string
	Status       string // running, completed, failed
	StartTime    time.Time
	EndTime      *time.Time
	Duration     int // 秒
	Output       string
	ErrorMessage string
	SuccessCount int
	FailedCount  int
	Stats        *PlaybookStats
}

// PlaybookStats Ansible 执行统计
type PlaybookStats struct {
	Ok          int `json:"ok"`
	Changed     int `json:"changed"`
	Unreachable int `json:"unreachable"`
	Failed      int `json:"failed"`
	Skipped     int `json:"skipped"`
	Rescued     int `json:"rescued"`
	Ignored     int `json:"ignored"`
}

// PlaybookProgressCallback 进度回调函数
type PlaybookProgressCallback func(event *PlaybookEvent)

// PlaybookEvent Playbook 执行事件
type PlaybookEvent struct {
	Type      string                 `json:"type"` // task_start, task_end, play_start, play_end
	Task      string                 `json:"task,omitempty"`
	Host      string                 `json:"host,omitempty"`
	Status    string                 `json:"status,omitempty"` // ok, failed, changed, skipped
	Message   string                 `json:"message,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data,omitempty"`
}

// NewPlaybookRunner 创建 Playbook 执行器
func NewPlaybookRunner(logger *logger.Logger, binaryPath, tempDir string, timeout time.Duration, credentialMgr *CredentialManager) *PlaybookRunner {
	return &PlaybookRunner{
		logger:        logger,
		binaryPath:    binaryPath,
		tempDir:       tempDir,
		timeout:       timeout,
		credentialMgr: credentialMgr,
	}
}

// Run 执行 Playbook
func (pr *PlaybookRunner) Run(ctx context.Context, config *PlaybookRunConfig, callback PlaybookProgressCallback) (*PlaybookResult, error) {
	result := &PlaybookResult{
		StartTime: time.Now(),
		Status:    "running",
		Output:    "",
	}

	// 创建临时工作目录
	workDir, err := os.MkdirTemp(pr.tempDir, "ansible-run-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(workDir)

	// 写入 Playbook 文件
	playbookFile := filepath.Join(workDir, "playbook.yml")
	if err := os.WriteFile(playbookFile, []byte(config.PlaybookContent), 0600); err != nil {
		return nil, fmt.Errorf("failed to write playbook file: %w", err)
	}

	// 创建 Inventory 文件
	inventoryFile := filepath.Join(workDir, "inventory.ini")
	if err := pr.createInventory(inventoryFile, config); err != nil {
		return nil, fmt.Errorf("failed to create inventory: %w", err)
	}

	// 获取 SSH 凭据
	var credential *model.SSHCredential
	if config.CredentialID > 0 {
		credential, err = pr.credentialMgr.GetCredential(config.CredentialID)
		if err != nil {
			return nil, fmt.Errorf("failed to get credential: %w", err)
		}
	}

	// 构建命令参数
	args := pr.buildCommandArgs(playbookFile, inventoryFile, config, credential, workDir)

	// 创建执行上下文（带超时）
	execCtx, cancel := context.WithTimeout(ctx, pr.timeout)
	defer cancel()

	// 执行命令
	cmd := exec.CommandContext(execCtx, pr.binaryPath, args...)
	cmd.Dir = workDir

	// 设置环境变量
	cmd.Env = append(os.Environ(),
		"ANSIBLE_FORCE_COLOR=false",
		"ANSIBLE_STDOUT_CALLBACK=json",
		"ANSIBLE_LOAD_CALLBACK_PLUGINS=1",
	)

	// 获取输出管道
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	// 启动命令
	pr.logger.Infof("Executing ansible-playbook: %s %v", pr.binaryPath, args)
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start ansible-playbook: %w", err)
	}

	// 读取输出
	outputChan := make(chan string, 100)
	errChan := make(chan error, 1)

	// 解析标准输出（JSON 格式）
	go pr.parseOutput(stdout, callback, outputChan)

	// 读取标准错误
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			pr.logger.Warning("Ansible stderr: " + line)
			outputChan <- line
		}
	}()

	// 等待命令完成
	go func() {
		errChan <- cmd.Wait()
	}()

	// 收集输出
	var outputBuilder strings.Builder

outputLoop:
	for {
		select {
		case line := <-outputChan:
			outputBuilder.WriteString(line + "\n")
		case err := <-errChan:
			// 命令完成
			endTime := time.Now()
			result.EndTime = &endTime
			result.Duration = int(endTime.Sub(result.StartTime).Seconds())
			result.Output = outputBuilder.String()

			if err != nil {
				result.Status = "failed"
				result.ErrorMessage = err.Error()
				return result, fmt.Errorf("ansible-playbook execution failed: %w", err)
			}

			result.Status = "completed"
			break outputLoop
		case <-ctx.Done():
			// 上下文取消
			if err := cmd.Process.Kill(); err != nil {
				pr.logger.Errorf("Failed to kill ansible-playbook process: %v", err)
			}
			result.Status = "cancelled"
			result.ErrorMessage = "execution cancelled"
			return result, ctx.Err()
		}
	}

	// 解析统计信息
	result.Stats = pr.parseStats(result.Output)
	if result.Stats != nil {
		result.SuccessCount = result.Stats.Ok + result.Stats.Changed
		result.FailedCount = result.Stats.Failed + result.Stats.Unreachable
	}

	return result, nil
}

// buildCommandArgs 构建 ansible-playbook 命令参数
func (pr *PlaybookRunner) buildCommandArgs(playbookFile, inventoryFile string, config *PlaybookRunConfig, credential *model.SSHCredential, workDir string) []string {
	args := []string{
		"-i", inventoryFile,
	}

	// 添加详细度
	if config.Verbose > 0 {
		verbosity := strings.Repeat("v", config.Verbose)
		args = append(args, "-"+verbosity)
	}

	// 检查模式
	if config.CheckMode {
		args = append(args, "--check")
	}

	// 标签过滤
	if len(config.Tags) > 0 {
		args = append(args, "--tags", strings.Join(config.Tags, ","))
	}

	// 额外变量
	if len(config.ExtraVars) > 0 {
		extraVarsJSON, _ := json.Marshal(config.ExtraVars)
		args = append(args, "--extra-vars", string(extraVarsJSON))
	}

	// SSH 配置
	if credential != nil {
		args = append(args, "--user", credential.Username)

		if credential.AuthType == "privatekey" && credential.PrivateKey != "" {
			// 写入私钥文件
			keyFile := filepath.Join(workDir, "ssh_key")
			if err := os.WriteFile(keyFile, []byte(credential.PrivateKey), 0600); err == nil {
				args = append(args, "--private-key", keyFile)
			}
		} else if credential.AuthType == "password" && credential.Password != "" {
			// 使用 sshpass（需要系统安装）
			// 注意：生产环境建议使用私钥认证
			args = append(args, "--extra-vars", fmt.Sprintf("ansible_password=%s", credential.Password))
		}

		// SSH 端口
		if credential.Port != 0 && credential.Port != 22 {
			args = append(args, "--extra-vars", fmt.Sprintf("ansible_port=%d", credential.Port))
		}
	}

	// Playbook 文件
	args = append(args, playbookFile)

	return args
}

// createInventory 创建 Inventory 文件
func (pr *PlaybookRunner) createInventory(inventoryFile string, config *PlaybookRunConfig) error {
	var content strings.Builder

	content.WriteString("[targets]\n")
	for _, node := range config.TargetNodes {
		content.WriteString(fmt.Sprintf("%s\n", node))
	}

	return os.WriteFile(inventoryFile, []byte(content.String()), 0600)
}

// parseOutput 解析 Ansible 输出（JSON callback 格式）
func (pr *PlaybookRunner) parseOutput(reader io.Reader, callback PlaybookProgressCallback, outputChan chan<- string) {
	scanner := bufio.NewScanner(reader)

	for scanner.Scan() {
		line := scanner.Text()
		outputChan <- line

		if callback == nil {
			continue
		}

		// 尝试解析为 JSON
		var data map[string]interface{}
		if err := json.Unmarshal([]byte(line), &data); err != nil {
			// 不是 JSON，跳过
			continue
		}

		// 构建事件
		event := &PlaybookEvent{
			Timestamp: time.Now(),
			Data:      data,
		}

		// 解析事件类型
		if eventType, ok := data["event"].(string); ok {
			event.Type = eventType
		}

		if task, ok := data["task"].(string); ok {
			event.Task = task
		}

		if host, ok := data["host"].(string); ok {
			event.Host = host
		}

		if status, ok := data["status"].(string); ok {
			event.Status = status
		}

		// 调用回调
		callback(event)
	}
}

// parseStats 从输出中解析统计信息
func (pr *PlaybookRunner) parseStats(output string) *PlaybookStats {
	// 查找 PLAY RECAP 部分
	lines := strings.Split(output, "\n")
	inRecap := false
	stats := &PlaybookStats{}

	for _, line := range lines {
		if strings.Contains(line, "PLAY RECAP") {
			inRecap = true
			continue
		}

		if inRecap && strings.Contains(line, ":") {
			// 解析统计行，例如：
			// node1 : ok=2    changed=1    unreachable=0    failed=0    skipped=0    rescued=0    ignored=0
			parts := strings.Split(line, ":")
			if len(parts) < 2 {
				continue
			}

			statsPart := parts[1]
			fields := strings.Fields(statsPart)

			for _, field := range fields {
				kv := strings.Split(field, "=")
				if len(kv) != 2 {
					continue
				}

				var value int
				fmt.Sscanf(kv[1], "%d", &value)

				switch kv[0] {
				case "ok":
					stats.Ok += value
				case "changed":
					stats.Changed += value
				case "unreachable":
					stats.Unreachable += value
				case "failed":
					stats.Failed += value
				case "skipped":
					stats.Skipped += value
				case "rescued":
					stats.Rescued += value
				case "ignored":
					stats.Ignored += value
				}
			}
		}
	}

	return stats
}

// Validate 验证 Playbook 语法
func (pr *PlaybookRunner) Validate(playbookContent string) error {
	// 创建临时文件
	tmpFile, err := os.CreateTemp(pr.tempDir, "playbook-*.yml")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write([]byte(playbookContent)); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write playbook: %w", err)
	}
	tmpFile.Close()

	// 执行语法检查
	cmd := exec.Command(pr.binaryPath, "--syntax-check", tmpFile.Name())
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("syntax check failed: %s", string(output))
	}

	return nil
}
