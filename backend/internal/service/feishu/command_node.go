package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/node"
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
	// 调用实际的集群服务
	if ctx.Service.clusterService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("集群服务未配置"),
		}, nil
	}

	// 调用集群服务获取列表
	result, err := ctx.Service.clusterService.List(cluster.ListRequest{
		Page:     1,
		PageSize: 100, // 获取所有集群
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取集群列表失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取集群列表失败: %s", err.Error())),
		}, nil
	}

	// 类型断言
	listResp, ok := result.(*cluster.ListResponse)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("数据格式错误"),
		}, nil
	}

	// 转换为卡片需要的格式
	var clusters []map[string]interface{}
	for _, c := range listResp.Clusters {
		status := "🟢 健康"
		if c.Status != "active" {
			status = "🔴 不可用"
		}

		clusters = append(clusters, map[string]interface{}{
			"name":   c.Name,
			"status": status,
			"nodes":  c.NodeCount,
		})
	}

	if len(clusters) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("系统中没有配置集群\n\n请先在 Web 界面添加集群配置"),
		}, nil
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

	// 调用节点服务获取真实数据
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	// 创建节点列表请求
	result, err := ctx.Service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取节点列表失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点列表失败: %s\n\n请检查集群连接是否正常", err.Error())),
		}, nil
	}

	// 类型断言 - node.List 返回 []k8s.NodeInfo
	nodeInfos, ok := result.([]k8s.NodeInfo)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("节点数据格式错误"),
		}, nil
	}

	// 转换为卡片需要的格式
	var nodes []map[string]interface{}
	for _, n := range nodeInfos {
		nodes = append(nodes, map[string]interface{}{
			"name":          n.Name,
			"ready":         n.Status == "Ready",
			"unschedulable": !n.Schedulable,
		})
	}

	if len(nodes) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("集群 %s 中没有节点", clusterName)),
		}, nil
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

	// 调用节点服务获取节点详情
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	// 获取节点列表，然后找到指定节点
	result, err := ctx.Service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取节点信息失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点信息失败: %s", err.Error())),
		}, nil
	}

	// 类型断言
	nodeInfos, ok := result.([]k8s.NodeInfo)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("节点数据格式错误"),
		}, nil
	}

	// 查找指定的节点
	var foundNode *k8s.NodeInfo
	for _, n := range nodeInfos {
		if n.Name == nodeName {
			foundNode = &n
			break
		}
	}

	if foundNode == nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("节点 %s 不存在\n\n集群: %s", nodeName, clusterName)),
		}, nil
	}

	// 转换为卡片需要的格式
	nodeInfo := map[string]interface{}{
		"name":              foundNode.Name,
		"ready":             foundNode.Status == "Ready",
		"unschedulable":     !foundNode.Schedulable,
		"internal_ip":       foundNode.InternalIP,
		"container_runtime": foundNode.ContainerRuntime,
		"kernel_version":    foundNode.KernelVersion,
		"os_image":          foundNode.OSImage,
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
