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
			Text: "请指定操作。用法: /node <list|set|info|cordon|uncordon> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListClusters(ctx)
	case "set":
		return h.handleSetCluster(ctx)
	case "info":
		return h.handleNodeInfo(ctx)
	case "cordon":
		return h.handleCordon(ctx)
	case "uncordon":
		return h.handleUncordon(ctx)
	case "nodes":
		return h.handleListNodes(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, set, info, cordon, uncordon, nodes", ctx.Command.Action),
		}, nil
	}
}

// handleListClusters 显示所有集群列表
func (h *NodeCommandHandler) handleListClusters(ctx *CommandContext) (*CommandResponse, error) {
	// TODO: 调用 cluster service 获取实际的集群列表
	// 暂时返回示例数据
	clusters := []map[string]interface{}{
		{
			"name":   "default",
			"status": "健康",
			"nodes":  2,
		},
		{
			"name":   "test-k8s-cluster",
			"status": "健康",
			"nodes":  2,
		},
	}

	return &CommandResponse{
		Card: BuildClusterListCard(clusters),
	}, nil
}

// handleSetCluster 设置当前操作的集群
func (h *NodeCommandHandler) handleSetCluster(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /node set <集群名称>"),
		}, nil
	}

	clusterName := ctx.Command.Args[0]

	// TODO: 验证集群是否存在
	// 暂时直接设置

	// 设置用户当前集群
	if err := ctx.Service.SetCurrentCluster(ctx.UserMapping.FeishuUserID, clusterName); err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("设置集群失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 已切换到集群: %s\n\n现在可以直接使用以下命令:\n• /node nodes - 查看节点列表\n• /node info <节点名> - 查看节点详情\n• /node cordon <节点名> - 禁止调度\n• /node uncordon <节点名> - 恢复调度", clusterName)),
	}, nil
}

// handleListNodes handles the node nodes command (list nodes in current cluster)
func (h *NodeCommandHandler) handleListNodes(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取当前集群失败: %s", err.Error())),
		}, nil
	}

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /node list 查看集群列表\n然后使用 /node set <集群名> 选择集群"),
		}, nil
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

	return &CommandResponse{
		Card: BuildNodeListCard(nodes, clusterName),
	}, nil
}

// handleNodeInfo handles the node info command
func (h *NodeCommandHandler) handleNodeInfo(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取当前集群失败: %s", err.Error())),
		}, nil
	}

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /node list 查看集群列表\n然后使用 /node set <集群名> 选择集群"),
		}, nil
	}

	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /node info <节点名>"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]

	// TODO: Implement actual node info logic
	nodeInfo := map[string]interface{}{
		"name":              nodeName,
		"ready":             true,
		"unschedulable":     false,
		"internal_ip":       "192.168.1.100",
		"container_runtime": "containerd://1.6.0",
		"kernel_version":    "5.10.0",
		"os_image":          "Ubuntu 20.04",
		"cluster":           clusterName,
	}

	return &CommandResponse{
		Card: BuildNodeInfoCard(nodeInfo),
	}, nil
}

// handleCordon handles the node cordon command
func (h *NodeCommandHandler) handleCordon(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取当前集群失败: %s", err.Error())),
		}, nil
	}

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /node list 查看集群列表\n然后使用 /node set <集群名> 选择集群"),
		}, nil
	}

	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /node cordon <节点名> [原因]"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]
	reason := ""
	if len(ctx.Command.Args) > 1 {
		reason = ctx.Command.Args[1]
	}

	// TODO: Implement actual cordon logic by calling node service
	// Check user permissions
	// Execute cordon operation
	// Log audit

	_ = clusterName
	_ = reason

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点 %s 已成功设置为禁止调度\n\n集群: %s", nodeName, clusterName)),
	}, nil
}

// handleUncordon handles the node uncordon command
func (h *NodeCommandHandler) handleUncordon(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取当前集群失败: %s", err.Error())),
		}, nil
	}

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /node list 查看集群列表\n然后使用 /node set <集群名> 选择集群"),
		}, nil
	}

	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /node uncordon <节点名>"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]

	// TODO: Implement actual uncordon logic by calling node service
	_ = clusterName

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点 %s 已成功恢复调度\n\n集群: %s", nodeName, clusterName)),
	}, nil
}

// Description returns the command description
func (h *NodeCommandHandler) Description() string {
	return "节点管理命令"
}
