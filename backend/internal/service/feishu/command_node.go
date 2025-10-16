package feishu

import (
	"fmt"
)

// NodeCommandHandler handles node-related commands
type NodeCommandHandler struct{}

// Handle executes the node command
func (h *NodeCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Node commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /node <list|info|cordon|uncordon> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListNodes(ctx)
	case "info":
		return h.handleNodeInfo(ctx)
	case "cordon":
		return h.handleCordon(ctx)
	case "uncordon":
		return h.handleUncordon(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, info, cordon, uncordon", ctx.Command.Action),
		}, nil
	}
}

// handleListNodes handles the node list command
func (h *NodeCommandHandler) handleListNodes(ctx *CommandContext) (*CommandResponse, error) {
	// Get cluster name (optional, defaults to all clusters or current)
	clusterName := ""
	if len(ctx.Command.Args) > 0 {
		clusterName = ctx.Command.Args[0]
	}

	// TODO: Implement actual node listing logic by calling node service
	// For now, return a placeholder
	nodes := []map[string]interface{}{
		{
			"name":          "node-1",
			"ready":         true,
			"unschedulable": false,
		},
		{
			"name":          "node-2",
			"ready":         true,
			"unschedulable": true,
		},
	}

	if clusterName == "" {
		clusterName = "default"
	}

	return &CommandResponse{
		Card: BuildNodeListCard(nodes, clusterName),
	}, nil
}

// handleNodeInfo handles the node info command
func (h *NodeCommandHandler) handleNodeInfo(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Text: "参数不足。用法: /node info <cluster> <node_name>",
		}, nil
	}

	clusterName := ctx.Command.Args[0]
	nodeName := ctx.Command.Args[1]

	// TODO: Implement actual node info logic
	nodeInfo := map[string]interface{}{
		"name":              nodeName,
		"ready":             true,
		"unschedulable":     false,
		"internal_ip":       "192.168.1.100",
		"container_runtime": "containerd://1.6.0",
		"kernel_version":    "5.10.0",
		"os_image":          "Ubuntu 20.04",
	}

	_ = clusterName // Use cluster name in actual implementation

	return &CommandResponse{
		Card: BuildNodeInfoCard(nodeInfo),
	}, nil
}

// handleCordon handles the node cordon command
func (h *NodeCommandHandler) handleCordon(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Text: "参数不足。用法: /node cordon <cluster> <node_name> [reason]",
		}, nil
	}

	clusterName := ctx.Command.Args[0]
	nodeName := ctx.Command.Args[1]
	reason := ""
	if len(ctx.Command.Args) > 2 {
		reason = ctx.Command.Args[2]
	}

	// TODO: Implement actual cordon logic by calling node service
	// Check user permissions
	// Execute cordon operation
	// Log audit

	_ = clusterName
	_ = reason

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("节点 %s 已成功设置为禁止调度", nodeName)),
	}, nil
}

// handleUncordon handles the node uncordon command
func (h *NodeCommandHandler) handleUncordon(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Text: "参数不足。用法: /node uncordon <cluster> <node_name>",
		}, nil
	}

	clusterName := ctx.Command.Args[0]
	nodeName := ctx.Command.Args[1]

	// TODO: Implement actual uncordon logic by calling node service
	_ = clusterName

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("节点 %s 已成功恢复调度", nodeName)),
	}, nil
}

// Description returns the command description
func (h *NodeCommandHandler) Description() string {
	return "节点管理命令"
}
