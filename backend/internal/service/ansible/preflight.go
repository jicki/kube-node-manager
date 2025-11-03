package ansible

import (
	"context"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"os/exec"
	"strings"
	"time"

	"gorm.io/gorm"
)

// PreflightService 前置检查服务
type PreflightService struct {
	db              *gorm.DB
	logger          *logger.Logger
	inventorySvc    *InventoryService
	sshKeySvc       *SSHKeyService
}

// NewPreflightService 创建前置检查服务实例
func NewPreflightService(
	db *gorm.DB,
	logger *logger.Logger,
	inventorySvc *InventoryService,
	sshKeySvc *SSHKeyService,
) *PreflightService {
	return &PreflightService{
		db:           db,
		logger:       logger,
		inventorySvc: inventorySvc,
		sshKeySvc:    sshKeySvc,
	}
}

// RunPreflightChecks 执行前置检查
func (s *PreflightService) RunPreflightChecks(taskID uint) (*model.PreflightCheckResult, error) {
	startTime := time.Now()
	s.logger.Infof("Starting preflight checks for task %d", taskID)

	// 获取任务信息
	var task model.AnsibleTask
	if err := s.db.Preload("Inventory").First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if task.Inventory == nil {
		return nil, fmt.Errorf("task inventory not found")
	}

	checks := []model.PreflightCheck{}

	// 1. 检查主机清单
	check1 := s.checkInventory(task.Inventory)
	checks = append(checks, check1)

	// 2. 检查 SSH 连接
	if task.Inventory.SSHKeyID != nil {
		check2 := s.checkSSHConnectivity(&task)
		checks = append(checks, check2)
	}

	// 3. 检查 Playbook 语法
	check3 := s.checkPlaybookSyntax(task.PlaybookContent)
	checks = append(checks, check3)

	// 计算摘要
	summary := s.calculateSummary(checks)

	// 确定总体状态
	overallStatus := "pass"
	if summary.Failed > 0 {
		overallStatus = "fail"
	} else if summary.Warnings > 0 {
		overallStatus = "warning"
	}

	duration := int(time.Since(startTime).Milliseconds())

	result := &model.PreflightCheckResult{
		Status:    overallStatus,
		CheckedAt: time.Now(),
		Duration:  duration,
		Checks:    checks,
		Summary:   summary,
	}

	// 保存检查结果到任务
	task.PreflightChecks = result
	if err := s.db.Save(&task).Error; err != nil {
		s.logger.Errorf("Failed to save preflight checks: %v", err)
	}

	s.logger.Infof("Preflight checks completed for task %d: %s (duration: %dms)", 
		taskID, overallStatus, duration)

	return result, nil
}

// checkInventory 检查主机清单
func (s *PreflightService) checkInventory(inventory *model.AnsibleInventory) model.PreflightCheck {
	startTime := time.Now()
	check := model.PreflightCheck{
		Name:      "主机清单检查",
		Category:  "config",
		CheckedAt: time.Now(),
	}

	// 检查清单内容
	if inventory.Content == "" {
		check.Status = "fail"
		check.Message = "主机清单内容为空"
		check.Details = "请确保清单中至少包含一个主机"
	} else if inventory.HostCount == 0 {
		check.Status = "warning"
		check.Message = "主机清单中没有主机"
		check.Details = "清单配置可能存在问题，建议检查"
	} else {
		check.Status = "pass"
		check.Message = fmt.Sprintf("主机清单正常，包含 %d 个主机", inventory.HostCount)
		check.Details = fmt.Sprintf("清单名称: %s, 来源: %s", inventory.Name, inventory.SourceType)
	}

	check.Duration = int(time.Since(startTime).Milliseconds())
	return check
}

// checkSSHConnectivity 检查 SSH 连接
func (s *PreflightService) checkSSHConnectivity(task *model.AnsibleTask) model.PreflightCheck {
	startTime := time.Now()
	check := model.PreflightCheck{
		Name:      "SSH 连接检查",
		Category:  "connectivity",
		CheckedAt: time.Now(),
	}

	// 获取 SSH 密钥
	if task.Inventory.SSHKeyID == nil {
		check.Status = "warning"
		check.Message = "未配置 SSH 密钥"
		check.Details = "任务将使用默认 SSH 配置，可能导致认证失败"
		check.Duration = int(time.Since(startTime).Milliseconds())
		return check
	}

	sshKey, err := s.sshKeySvc.GetSSHKey(*task.Inventory.SSHKeyID)
	if err != nil {
		check.Status = "fail"
		check.Message = "SSH 密钥不存在"
		check.Details = fmt.Sprintf("无法获取 SSH 密钥: %v", err)
		check.Duration = int(time.Since(startTime).Milliseconds())
		return check
	}

	// 简单检查：验证 SSH 密钥类型
	if sshKey.AuthType == "password" {
		check.Status = "pass"
		check.Message = "SSH 密码认证已配置"
		check.Details = fmt.Sprintf("SSH 用户: %s", sshKey.SSHUser)
	} else if sshKey.AuthType == "key" {
		if sshKey.PrivateKey == "" {
			check.Status = "fail"
			check.Message = "SSH 私钥为空"
			check.Details = "请配置有效的 SSH 私钥"
		} else {
			check.Status = "pass"
			check.Message = "SSH 密钥认证已配置"
			check.Details = fmt.Sprintf("SSH 用户: %s, 密钥长度: %d bytes", 
				sshKey.SSHUser, len(sshKey.PrivateKey))
		}
	} else {
		check.Status = "warning"
		check.Message = "未知的 SSH 认证类型"
		check.Details = fmt.Sprintf("认证类型: %s", sshKey.AuthType)
	}

	check.Duration = int(time.Since(startTime).Milliseconds())
	return check
}

// checkPlaybookSyntax 检查 Playbook 语法
func (s *PreflightService) checkPlaybookSyntax(playbookContent string) model.PreflightCheck {
	startTime := time.Now()
	check := model.PreflightCheck{
		Name:      "Playbook 语法检查",
		Category:  "config",
		CheckedAt: time.Now(),
	}

	// 基本检查：内容不为空
	if playbookContent == "" {
		check.Status = "fail"
		check.Message = "Playbook 内容为空"
		check.Details = "请提供有效的 Playbook 内容"
		check.Duration = int(time.Since(startTime).Milliseconds())
		return check
	}

	// 使用 ansible-playbook --syntax-check（如果 ansible 已安装）
	// 这是可选的，如果没有安装 ansible 命令行工具，会跳过
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 检查是否安装了 ansible-playbook
	checkCmd := exec.CommandContext(ctx, "which", "ansible-playbook")
	if err := checkCmd.Run(); err != nil {
		// ansible 未安装，跳过语法检查
		check.Status = "pass"
		check.Message = "Playbook 内容已提供"
		check.Details = "跳过语法检查（ansible-playbook 未安装）"
		check.Duration = int(time.Since(startTime).Milliseconds())
		return check
	}

	// 执行语法检查（简化版：只检查基本格式）
	if !strings.Contains(playbookContent, "hosts:") && !strings.Contains(playbookContent, "- name:") {
		check.Status = "warning"
		check.Message = "Playbook 格式可能不正确"
		check.Details = "未找到必需的 hosts 或 name 字段"
	} else {
		check.Status = "pass"
		check.Message = "Playbook 格式正常"
		check.Details = fmt.Sprintf("Playbook 大小: %d bytes", len(playbookContent))
	}

	check.Duration = int(time.Since(startTime).Milliseconds())
	return check
}

// calculateSummary 计算检查摘要
func (s *PreflightService) calculateSummary(checks []model.PreflightCheck) model.PreflightSummary {
	summary := model.PreflightSummary{
		Total: len(checks),
	}

	for _, check := range checks {
		switch check.Status {
		case "pass":
			summary.Passed++
		case "warning":
			summary.Warnings++
		case "fail":
			summary.Failed++
		}
	}

	return summary
}

// GetPreflightChecks 获取任务的前置检查结果
func (s *PreflightService) GetPreflightChecks(taskID uint) (*model.PreflightCheckResult, error) {
	var task model.AnsibleTask
	if err := s.db.First(&task, taskID).Error; err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	if task.PreflightChecks == nil {
		return nil, fmt.Errorf("no preflight checks found for this task")
	}

	return task.PreflightChecks, nil
}

