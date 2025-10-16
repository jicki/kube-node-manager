package feishu

import (
	"fmt"
)

// ClusterCommandHandler handles cluster-related commands
type ClusterCommandHandler struct{}

// Handle executes the cluster command
func (h *ClusterCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Cluster commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /cluster <list|status> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListClusters(ctx)
	case "status":
		return h.handleClusterStatus(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, status", ctx.Command.Action),
		}, nil
	}
}

// handleListClusters handles the cluster list command
func (h *ClusterCommandHandler) handleListClusters(ctx *CommandContext) (*CommandResponse, error) {
	// TODO: Implement actual cluster listing logic by calling cluster service
	clusters := []map[string]interface{}{
		{
			"name":       "production",
			"status":     "active",
			"node_count": 10,
		},
		{
			"name":       "staging",
			"status":     "active",
			"node_count": 5,
		},
	}

	return &CommandResponse{
		Card: BuildClusterListCard(clusters),
	}, nil
}

// handleClusterStatus handles the cluster status command
func (h *ClusterCommandHandler) handleClusterStatus(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Text: "参数不足。用法: /cluster status <cluster_name>",
		}, nil
	}

	clusterName := ctx.Command.Args[0]

	// TODO: Implement actual cluster status logic
	statusText := fmt.Sprintf(`**集群**: %s
**状态**: 🟢 正常
**节点数**: 10
**健康节点**: 10
**不健康节点**: 0`, clusterName)

	return &CommandResponse{
		Text: statusText,
	}, nil
}

// Description returns the command description
func (h *ClusterCommandHandler) Description() string {
	return "集群管理命令"
}
