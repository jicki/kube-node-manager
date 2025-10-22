package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/node"
	"strings"
)

// QuickCommandHandler handles quick operation commands
type QuickCommandHandler struct{}

// Handle executes the quick command
func (h *QuickCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Card: BuildQuickHelpCard(),
		}, nil
	}

	// 检查用户是否已绑定
	if ctx.UserMapping == nil || ctx.UserMapping.SystemUserID == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 没有权限操作\n\n请联系管理员。"),
		}, nil
	}

	switch ctx.Command.Action {
	case "status":
		return h.handleQuickStatus(ctx)
	case "nodes":
		return h.handleQuickNodes(ctx)
	case "health":
		return h.handleQuickHealth(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知快捷命令: %s。支持的命令: status, nodes, health", ctx.Command.Action),
		}, nil
	}
}

// handleQuickStatus shows current cluster status overview
func (h *QuickCommandHandler) handleQuickStatus(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /cluster set <集群名> 选择集群"),
		}, nil
	}

	// 获取节点列表
	nodes, err := ctx.Service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, ctx.UserMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点列表失败: %s", err.Error())),
		}, nil
	}

	nodeList, ok := nodes.([]k8s.NodeInfo)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("节点列表数据格式错误"),
		}, nil
	}

	// 统计节点状态
	totalNodes := len(nodeList)
	readyNodes := 0
	unschedulableNodes := 0
	notReadyNodes := 0

	for _, n := range nodeList {
		// 使用 strings.Contains 检查状态，因为状态可能是 "Ready,SchedulingDisabled"
		if strings.Contains(n.Status, "Ready") {
			readyNodes++
		} else {
			notReadyNodes++
		}
		// 检查节点是否可调度
		if !n.Schedulable {
			unschedulableNodes++
		}
	}

	return &CommandResponse{
		Card: BuildQuickStatusCard(clusterName, nil, totalNodes, readyNodes, notReadyNodes, unschedulableNodes),
	}, nil
}

// handleQuickNodes shows problematic nodes (NotReady/Unschedulable)
func (h *QuickCommandHandler) handleQuickNodes(ctx *CommandContext) (*CommandResponse, error) {
	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /cluster set <集群名> 选择集群"),
		}, nil
	}

	// 获取节点列表
	nodes, err := ctx.Service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, ctx.UserMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点列表失败: %s", err.Error())),
		}, nil
	}

	nodeList, ok := nodes.([]k8s.NodeInfo)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("节点列表数据格式错误"),
		}, nil
	}

	// 过滤问题节点（NotReady 或 禁止调度）
	var problematicNodes []k8s.NodeInfo
	for _, n := range nodeList {
		// NotReady 或 禁止调度的节点视为问题节点
		// 使用 strings.Contains 检查状态，因为状态可能是 "Ready,SchedulingDisabled" 这样的组合
		isReady := strings.Contains(n.Status, "Ready")
		if !isReady || !n.Schedulable {
			problematicNodes = append(problematicNodes, n)
		}
	}

	return &CommandResponse{
		Card: BuildQuickNodesCard(clusterName, problematicNodes),
	}, nil
}

// handleQuickHealth performs health check across all clusters
func (h *QuickCommandHandler) handleQuickHealth(ctx *CommandContext) (*CommandResponse, error) {
	// 简化实现：返回帮助信息提示用户使用其他命令
	return &CommandResponse{
		Card: BuildQuickHealthCard(nil),
	}, nil
}

// Description returns the command description
func (h *QuickCommandHandler) Description() string {
	return "快捷操作命令"
}
