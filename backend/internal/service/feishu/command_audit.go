package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/audit"
	"strconv"
)

// AuditCommandHandler handles audit-related commands
type AuditCommandHandler struct{}

// Handle executes the audit command
func (h *AuditCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Audit commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /audit logs [user] [limit]",
		}, nil
	}

	switch ctx.Command.Action {
	case "logs":
		return h.handleAuditLogs(ctx)
	default:
		return &CommandResponse{
			Text: "未知操作。支持的操作: logs",
		}, nil
	}
}

// handleAuditLogs handles the audit logs command
func (h *AuditCommandHandler) handleAuditLogs(ctx *CommandContext) (*CommandResponse, error) {
	username := ""
	limit := 10

	// Parse arguments
	if len(ctx.Command.Args) > 0 {
		username = ctx.Command.Args[0]
	}

	if len(ctx.Command.Args) > 1 {
		if l, err := strconv.Atoi(ctx.Command.Args[1]); err == nil && l > 0 && l <= 20 {
			limit = l
		}
	}

	// 调用审计服务获取真实数据
	if ctx.Service.auditService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("审计服务未配置"),
		}, nil
	}

	result, err := ctx.Service.auditService.List(audit.ListRequest{
		Page:     1,
		PageSize: limit,
		Username: username,
	})

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取审计日志失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取审计日志失败: %s", err.Error())),
		}, nil
	}

	// 类型断言
	listResp, ok := result.(*audit.ListResponse)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("数据格式错误"),
		}, nil
	}

	// 转换为卡片需要的格式
	var logs []map[string]interface{}
	for _, log := range listResp.Logs {
		username := "未知用户"
		if log.User.Username != "" {
			username = log.User.Username
		}

		action := string(log.Action)
		details := log.Details
		status := string(log.Status)
		createdAt := log.CreatedAt.Format("2006-01-02 15:04:05")

		logs = append(logs, map[string]interface{}{
			"username":   username,
			"action":     action,
			"details":    details,
			"status":     status,
			"created_at": createdAt,
		})
	}

	if len(logs) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("没有找到审计日志"),
		}, nil
	}

	return &CommandResponse{
		Card: BuildAuditLogsCard(logs),
	}, nil
}

// Description returns the command description
func (h *AuditCommandHandler) Description() string {
	return "审计日志命令"
}
