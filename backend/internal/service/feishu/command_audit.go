package feishu

import (
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

	// TODO: Implement actual audit log querying logic
	logs := []map[string]interface{}{
		{
			"username":   "admin",
			"action":     "cordon",
			"details":    "禁止调度节点 node-1",
			"status":     "success",
			"created_at": "2024-01-01 12:00:00",
		},
		{
			"username":   "operator",
			"action":     "uncordon",
			"details":    "恢复调度节点 node-2",
			"status":     "success",
			"created_at": "2024-01-01 11:30:00",
		},
	}

	_ = username
	_ = limit

	return &CommandResponse{
		Card: BuildAuditLogsCard(logs),
	}, nil
}

// Description returns the command description
func (h *AuditCommandHandler) Description() string {
	return "审计日志命令"
}
