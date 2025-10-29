package automation

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// AnsibleService Ansible 服务
type AnsibleService struct {
	db            *gorm.DB
	logger        *logger.Logger
	k8sSvc        *k8s.Service
	progressSvc   *progress.Service
	runner        *PlaybookRunner
	credentialMgr *CredentialManager
}

// NewAnsibleService 创建 Ansible 服务
func NewAnsibleService(
	db *gorm.DB,
	logger *logger.Logger,
	k8sSvc *k8s.Service,
	progressSvc *progress.Service,
	binaryPath string,
	tempDir string,
	timeout time.Duration,
	encryptionKey string,
) *AnsibleService {
	credentialMgr := NewCredentialManager(db, logger, encryptionKey)
	runner := NewPlaybookRunner(logger, binaryPath, tempDir, timeout, credentialMgr)

	return &AnsibleService{
		db:            db,
		logger:        logger,
		k8sSvc:        k8sSvc,
		progressSvc:   progressSvc,
		runner:        runner,
		credentialMgr: credentialMgr,
	}
}

// CreatePlaybook 创建 Playbook
func (as *AnsibleService) CreatePlaybook(playbook *model.AnsiblePlaybook) error {
	// 验证 Playbook 语法
	if err := as.runner.Validate(playbook.Content); err != nil {
		return fmt.Errorf("playbook validation failed: %w", err)
	}

	// 检查名称是否已存在
	var existing model.AnsiblePlaybook
	if err := as.db.Where("name = ?", playbook.Name).First(&existing).Error; err == nil {
		return fmt.Errorf("playbook with name '%s' already exists", playbook.Name)
	} else if err != gorm.ErrRecordNotFound {
		return err
	}

	// 创建 Playbook
	if err := as.db.Create(playbook).Error; err != nil {
		return fmt.Errorf("failed to create playbook: %w", err)
	}

	as.logger.Infof("Created playbook: %s (ID: %d)", playbook.Name, playbook.ID)
	return nil
}

// GetPlaybook 获取 Playbook
func (as *AnsibleService) GetPlaybook(id uint) (*model.AnsiblePlaybook, error) {
	var playbook model.AnsiblePlaybook
	if err := as.db.First(&playbook, id).Error; err != nil {
		return nil, err
	}
	return &playbook, nil
}

// ListPlaybooks 列出 Playbook
func (as *AnsibleService) ListPlaybooks(category string, isActive *bool) ([]model.AnsiblePlaybook, error) {
	var playbooks []model.AnsiblePlaybook
	query := as.db.Model(&model.AnsiblePlaybook{})

	if category != "" {
		query = query.Where("category = ?", category)
	}

	if isActive != nil {
		query = query.Where("is_active = ?", *isActive)
	}

	if err := query.Order("created_at DESC").Find(&playbooks).Error; err != nil {
		return nil, err
	}

	return playbooks, nil
}

// UpdatePlaybook 更新 Playbook
func (as *AnsibleService) UpdatePlaybook(id uint, updates *model.AnsiblePlaybook) error {
	var existing model.AnsiblePlaybook
	if err := as.db.First(&existing, id).Error; err != nil {
		return err
	}

	// 如果内容有更新，验证语法
	if updates.Content != "" && updates.Content != existing.Content {
		if err := as.runner.Validate(updates.Content); err != nil {
			return fmt.Errorf("playbook validation failed: %w", err)
		}
		// 版本号自动递增
		updates.Version = existing.Version + 1
	}

	// 不允许修改内置 Playbook 的某些字段
	if existing.IsBuiltin {
		updates.IsBuiltin = true
		// 内置 Playbook 可以被复制修改，但原始的不能删除
	}

	if err := as.db.Model(&existing).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update playbook: %w", err)
	}

	as.logger.Infof("Updated playbook: %s (ID: %d, Version: %d)", existing.Name, id, updates.Version)
	return nil
}

// DeletePlaybook 删除 Playbook
func (as *AnsibleService) DeletePlaybook(id uint) error {
	var playbook model.AnsiblePlaybook
	if err := as.db.First(&playbook, id).Error; err != nil {
		return err
	}

	// 不允许删除内置 Playbook
	if playbook.IsBuiltin {
		return fmt.Errorf("cannot delete builtin playbook")
	}

	if err := as.db.Delete(&playbook).Error; err != nil {
		return fmt.Errorf("failed to delete playbook: %w", err)
	}

	as.logger.Infof("Deleted playbook: %s (ID: %d)", playbook.Name, id)
	return nil
}

// ExecutePlaybook 执行 Playbook
func (as *AnsibleService) ExecutePlaybook(ctx context.Context, config *PlaybookRunConfig, userID uint) (string, error) {
	// 获取 Playbook
	playbook, err := as.GetPlaybook(config.PlaybookID)
	if err != nil {
		return "", fmt.Errorf("failed to get playbook: %w", err)
	}

	if !playbook.IsActive {
		return "", fmt.Errorf("playbook is not active")
	}

	config.PlaybookContent = playbook.Content

	// 生成任务 ID
	taskID := fmt.Sprintf("ansible_playbook_%d_%d", playbook.ID, time.Now().UnixNano())

	// 创建执行记录
	execution := &model.AnsibleExecution{
		TaskID:       taskID,
		PlaybookID:   playbook.ID,
		PlaybookName: playbook.Name,
		ClusterName:  config.ClusterName,
		TargetNodes:  as.marshalJSON(config.TargetNodes),
		ExtraVars:    as.marshalJSON(config.ExtraVars),
		Tags:         strings.Join(config.Tags, ","),
		CheckMode:    config.CheckMode,
		Status:       "pending",
		UserID:       userID,
	}

	if err := as.db.Create(execution).Error; err != nil {
		return "", fmt.Errorf("failed to create execution record: %w", err)
	}

	// 异步执行
	go as.runPlaybookAsync(ctx, config, execution)

	return taskID, nil
}

// runPlaybookAsync 异步执行 Playbook
func (as *AnsibleService) runPlaybookAsync(ctx context.Context, config *PlaybookRunConfig, execution *model.AnsibleExecution) {
	startTime := time.Now()

	// 更新状态为 running
	as.db.Model(execution).Updates(map[string]interface{}{
		"status":     "running",
		"start_time": startTime,
	})

	// 创建进度任务
	if as.progressSvc != nil {
		as.progressSvc.CreateTask(execution.TaskID, "ansible_playbook", len(config.TargetNodes), execution.UserID)
	}

	// 进度回调
	current := 0
	progressCallback := func(event *PlaybookEvent) {
		if as.progressSvc != nil && event.Status != "" {
			current++
			message := fmt.Sprintf("[%s] %s: %s", event.Host, event.Task, event.Status)
			as.progressSvc.UpdateProgress(execution.TaskID, current, event.Host, execution.UserID)
			as.logger.Infof("Playbook progress: %s", message)
		}
	}

	// 执行 Playbook
	result, err := as.runner.Run(ctx, config, progressCallback)

	endTime := time.Now()
	execution.EndTime = &endTime
	execution.Duration = int(endTime.Sub(startTime).Seconds())

	if err != nil {
		execution.Status = "failed"
		execution.ErrorMessage = err.Error()
		as.logger.Errorf("Playbook execution failed: %v", err)
	} else {
		execution.Status = result.Status
		execution.Output = result.Output
		execution.SuccessCount = result.SuccessCount
		execution.FailedCount = result.FailedCount

		if result.ErrorMessage != "" {
			execution.ErrorMessage = result.ErrorMessage
		}
	}

	// 更新执行记录
	if err := as.db.Save(execution).Error; err != nil {
		as.logger.Errorf("Failed to save execution record: %v", err)
	}

	// 发送完成消息
	if as.progressSvc != nil {
		if execution.Status == "failed" || err != nil {
			as.progressSvc.ErrorTask(execution.TaskID, fmt.Errorf("%s", execution.ErrorMessage), execution.UserID)
		} else {
			as.progressSvc.CompleteTask(execution.TaskID, execution.UserID)
		}
	}

	as.logger.Infof("Playbook execution completed: TaskID=%s, Status=%s, Duration=%ds",
		execution.TaskID, execution.Status, execution.Duration)
}

// GetExecutionStatus 获取执行状态
func (as *AnsibleService) GetExecutionStatus(taskID string) (*model.AnsibleExecution, error) {
	var execution model.AnsibleExecution
	if err := as.db.Where("task_id = ?", taskID).Preload("Playbook").Preload("User").First(&execution).Error; err != nil {
		return nil, err
	}
	return &execution, nil
}

// ListExecutions 列出执行历史
func (as *AnsibleService) ListExecutions(clusterName string, status string, limit int, offset int) ([]model.AnsibleExecution, int64, error) {
	var executions []model.AnsibleExecution
	var total int64

	query := as.db.Model(&model.AnsibleExecution{})

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
		Preload("Playbook").
		Preload("User").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&executions).Error; err != nil {
		return nil, 0, err
	}

	return executions, total, nil
}

// CancelExecution 取消执行
func (as *AnsibleService) CancelExecution(taskID string) error {
	var execution model.AnsibleExecution
	if err := as.db.Where("task_id = ?", taskID).First(&execution).Error; err != nil {
		return err
	}

	if execution.Status != "running" && execution.Status != "pending" {
		return fmt.Errorf("execution is not running or pending")
	}

	// 更新状态
	execution.Status = "cancelled"
	if err := as.db.Save(&execution).Error; err != nil {
		return err
	}

	as.logger.Infof("Cancelled playbook execution: TaskID=%s", taskID)
	return nil
}

// InitializeBuiltinPlaybooks 初始化内置 Playbook
func (as *AnsibleService) InitializeBuiltinPlaybooks() error {
	builtinPlaybooks := []model.AnsiblePlaybook{
		{
			Name:        "System Upgrade",
			Description: "升级系统软件包到最新版本",
			Content: `---
- name: System Upgrade
  hosts: all
  become: yes
  tasks:
    - name: Update apt cache
      apt:
        update_cache: yes
      when: ansible_os_family == "Debian"

    - name: Upgrade all packages (Debian/Ubuntu)
      apt:
        upgrade: dist
      when: ansible_os_family == "Debian"

    - name: Upgrade all packages (RedHat/CentOS)
      yum:
        name: "*"
        state: latest
      when: ansible_os_family == "RedHat"`,
			Category:  "system",
			Variables: "{}",
			IsBuiltin: true,
			IsActive:  true,
		},
		{
			Name:        "Docker Restart",
			Description: "重启 Docker 服务",
			Content: `---
- name: Docker Restart
  hosts: all
  become: yes
  tasks:
    - name: Restart Docker service
      systemd:
        name: docker
        state: restarted
        daemon_reload: yes

    - name: Wait for Docker to be ready
      wait_for:
        port: 2375
        timeout: 30
      ignore_errors: yes`,
			Category:  "docker",
			Variables: "{}",
			IsBuiltin: true,
			IsActive:  true,
		},
		{
			Name:        "Kernel Upgrade",
			Description: "升级 Linux 内核",
			Content: `---
- name: Kernel Upgrade
  hosts: all
  become: yes
  tasks:
    - name: Update kernel packages (Debian/Ubuntu)
      apt:
        name: linux-image-generic
        state: latest
      when: ansible_os_family == "Debian"

    - name: Update kernel packages (RedHat/CentOS)
      yum:
        name: kernel
        state: latest
      when: ansible_os_family == "RedHat"`,
			Category:  "kernel",
			Variables: "{}",
			IsBuiltin: true,
			IsActive:  true,
		},
		{
			Name:        "Security Patches",
			Description: "安装安全补丁",
			Content: `---
- name: Security Patches
  hosts: all
  become: yes
  tasks:
    - name: Install security updates (Debian/Ubuntu)
      apt:
        upgrade: safe
        update_cache: yes
      when: ansible_os_family == "Debian"

    - name: Install security updates (RedHat/CentOS)
      yum:
        name: "*"
        state: latest
        security: yes
      when: ansible_os_family == "RedHat"`,
			Category:  "security",
			Variables: "{}",
			IsBuiltin: true,
			IsActive:  true,
		},
	}

	for _, playbook := range builtinPlaybooks {
		var existing model.AnsiblePlaybook
		if err := as.db.Where("name = ? AND is_builtin = ?", playbook.Name, true).First(&existing).Error; err == gorm.ErrRecordNotFound {
			if err := as.db.Create(&playbook).Error; err != nil {
				as.logger.Errorf("Failed to create builtin playbook '%s': %v", playbook.Name, err)
				continue
			}
			as.logger.Infof("Created builtin playbook: %s", playbook.Name)
		}
	}

	return nil
}

// marshalJSON 辅助函数：序列化为 JSON
func (as *AnsibleService) marshalJSON(v interface{}) string {
	data, _ := json.Marshal(v)
	return string(data)
}
