package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/node"
	"strings"
)

// NodeCommandHandler handles node-related commands
type NodeCommandHandler struct{}

// Handle executes the node command
func (h *NodeCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Node commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /node <list|info|cordon|uncordon|batch> [参数...]",
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
	case "batch":
		return h.handleBatchOperation(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, info, cordon, uncordon, batch", ctx.Command.Action),
		}, nil
	}
}

// handleListNodes handles the node list command (list nodes in current cluster)
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
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /cluster list 查看集群列表\n然后使用 /cluster set <集群名> 选择集群"),
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

	// 获取搜索关键词和过滤参数
	var searchKeyword string
	if len(ctx.Command.Args) > 0 {
		searchKeyword = ctx.Command.Args[0]
	}

	// 转换为卡片需要的格式并应用过滤
	var nodes []map[string]interface{}
	for _, n := range nodeInfos {
		// 模糊搜索过滤
		if searchKeyword != "" {
			if !strings.Contains(strings.ToLower(n.Name), strings.ToLower(searchKeyword)) {
				continue
			}
		}

		nodeData := map[string]interface{}{
			"name":          n.Name,
			"ready":         n.Status == "Ready",
			"unschedulable": !n.Schedulable,
			"roles":         n.Roles, // 添加节点类型
		}

		// 优先使用 deeproute.cn/user-type 标签
		if userType, exists := n.Labels["deeproute.cn/user-type"]; exists {
			nodeData["user_type"] = userType
		}

		nodes = append(nodes, nodeData)
	}

	if len(nodes) == 0 {
		if searchKeyword != "" {
			return &CommandResponse{
				Card: BuildErrorCard(fmt.Sprintf("❌ 未找到匹配的节点\n\n搜索关键词: `%s`\n集群: %s", searchKeyword, clusterName)),
			}, nil
		}
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("集群 %s 中没有节点", clusterName)),
		}, nil
	}

	// 使用交互式按钮卡片
	return &CommandResponse{
		Card: BuildNodeListCardWithActions(nodes, clusterName),
	}, nil
}

// handleListClusters 显示所有集群列表（已废弃，由 /cluster list 替代）
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
		// 使用代码块格式避免节点名称被识别为超链接
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("节点 `%s` 不存在\n\n集群: %s", nodeName, clusterName)),
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

	// 添加资源信息
	capacity := map[string]interface{}{
		"cpu":    foundNode.Capacity.CPU,
		"memory": foundNode.Capacity.Memory,
		"pods":   foundNode.Capacity.Pods,
		"gpu":    foundNode.Capacity.GPU,
	}
	allocatable := map[string]interface{}{
		"cpu":    foundNode.Allocatable.CPU,
		"memory": foundNode.Allocatable.Memory,
		"pods":   foundNode.Allocatable.Pods,
		"gpu":    foundNode.Allocatable.GPU,
	}
	nodeInfo["capacity"] = capacity
	nodeInfo["allocatable"] = allocatable

	// 添加使用量信息（如果有）
	if foundNode.Usage != nil {
		nodeInfo["cpu_usage"] = foundNode.Usage.CPU
		nodeInfo["memory_usage"] = foundNode.Usage.Memory
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
		// 显示用法和常用原因选项
		return &CommandResponse{
			Card: BuildCordonHelpCard(),
		}, nil
	}

	nodeName := ctx.Command.Args[0]
	reason := ""
	if len(ctx.Command.Args) > 1 {
		// 合并剩余的参数作为原因
		reason = joinArgs(ctx.Command.Args[1:])
	}

	// 调用节点服务执行禁止调度
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	err = ctx.Service.nodeService.Cordon(node.CordonRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		Reason:      reason,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("禁止调度节点失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("禁止调度节点失败: %s", err.Error())),
		}, nil
	}

	reasonText := ""
	if reason != "" {
		reasonText = fmt.Sprintf("\n原因: %s", reason)
	}

	// 使用代码块格式避免节点名称被识别为超链接
	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功设置为禁止调度\n\n节点: `%s`\n集群: %s%s", nodeName, clusterName, reasonText)),
	}, nil
}

// joinArgs 合并参数数组为字符串
func joinArgs(args []string) string {
	result := ""
	for i, arg := range args {
		if i > 0 {
			result += " "
		}
		result += arg
	}
	return result
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

	// 调用节点服务执行恢复调度
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	err = ctx.Service.nodeService.Uncordon(node.CordonRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("恢复调度节点失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("恢复调度节点失败: %s", err.Error())),
		}, nil
	}

	// 使用代码块格式避免节点名称被识别为超链接
	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功恢复调度\n\n节点: `%s`\n集群: %s", nodeName, clusterName)),
	}, nil
}

// handleBatchOperation handles batch operations on multiple nodes
func (h *NodeCommandHandler) handleBatchOperation(ctx *CommandContext) (*CommandResponse, error) {
	// 批量操作格式: /node batch <operation> <node1,node2,node3> [args...]
	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Card: BuildBatchHelpCard(),
		}, nil
	}

	operation := ctx.Command.Args[0]
	nodeList := ctx.Command.Args[1]

	// 解析节点列表（逗号分隔）
	nodeNames := parseNodeList(nodeList)
	if len(nodeNames) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("节点列表为空\n\n格式: node1,node2,node3"),
		}, nil
	}

	// 获取用户当前选择的集群
	clusterName, err := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)
	if err != nil || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 尚未选择集群\n\n请先使用 /cluster set <集群名> 选择集群"),
		}, nil
	}

	switch operation {
	case "cordon":
		return h.handleBatchCordon(ctx, clusterName, nodeNames)
	case "uncordon":
		return h.handleBatchUncordon(ctx, clusterName, nodeNames)
	default:
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("未知批量操作: %s\n\n支持的操作: cordon, uncordon", operation)),
		}, nil
	}
}

// handleBatchCordon handles batch cordon operation
func (h *NodeCommandHandler) handleBatchCordon(ctx *CommandContext, clusterName string, nodeNames []string) (*CommandResponse, error) {
	// 获取原因（如果有）
	reason := ""
	if len(ctx.Command.Args) > 2 {
		reason = joinArgs(ctx.Command.Args[2:])
	}

	// 验证节点服务
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	// 执行批量操作
	results := make(map[string]string) // nodeName -> "success" or error message
	successCount := 0
	failureCount := 0

	for _, nodeName := range nodeNames {
		err := ctx.Service.nodeService.Cordon(node.CordonRequest{
			ClusterName: clusterName,
			NodeName:    nodeName,
			Reason:      reason,
		}, ctx.UserMapping.SystemUserID)

		if err != nil {
			results[nodeName] = err.Error()
			failureCount++
			ctx.Service.logger.Error(fmt.Sprintf("批量禁止调度失败 - 节点: %s, 错误: %v", nodeName, err))
		} else {
			results[nodeName] = "success"
			successCount++
		}
	}

	// 构建结果卡片
	return &CommandResponse{
		Card: BuildBatchOperationResultCard("禁止调度", clusterName, nodeNames, results, successCount, failureCount, reason),
	}, nil
}

// handleBatchUncordon handles batch uncordon operation
func (h *NodeCommandHandler) handleBatchUncordon(ctx *CommandContext, clusterName string, nodeNames []string) (*CommandResponse, error) {
	// 验证节点服务
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	// 执行批量操作
	results := make(map[string]string)
	successCount := 0
	failureCount := 0

	for _, nodeName := range nodeNames {
		err := ctx.Service.nodeService.Uncordon(node.CordonRequest{
			ClusterName: clusterName,
			NodeName:    nodeName,
		}, ctx.UserMapping.SystemUserID)

		if err != nil {
			results[nodeName] = err.Error()
			failureCount++
			ctx.Service.logger.Error(fmt.Sprintf("批量恢复调度失败 - 节点: %s, 错误: %v", nodeName, err))
		} else {
			results[nodeName] = "success"
			successCount++
		}
	}

	// 构建结果卡片
	return &CommandResponse{
		Card: BuildBatchOperationResultCard("恢复调度", clusterName, nodeNames, results, successCount, failureCount, ""),
	}, nil
}

// parseNodeList parses comma-separated node list
func parseNodeList(nodeList string) []string {
	nodes := strings.Split(nodeList, ",")
	var result []string
	for _, node := range nodes {
		node = strings.TrimSpace(node)
		if node != "" {
			result = append(result, node)
		}
	}
	return result
}

// Description returns the command description
func (h *NodeCommandHandler) Description() string {
	return "节点管理命令"
}
