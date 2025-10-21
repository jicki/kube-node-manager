package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/cluster"
)

// ClusterCommandHandler handles cluster-related commands
type ClusterCommandHandler struct{}

// Handle executes the cluster command
func (h *ClusterCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Cluster commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /cluster <list|set|status> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListClusters(ctx)
	case "set":
		return h.handleSetCluster(ctx)
	case "status":
		return h.handleClusterStatus(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, set, status", ctx.Command.Action),
		}, nil
	}
}

// handleListClusters handles the cluster list command
func (h *ClusterCommandHandler) handleListClusters(ctx *CommandContext) (*CommandResponse, error) {
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

	// 获取当前选择的集群
	currentCluster, _ := ctx.Service.GetCurrentCluster(ctx.UserMapping.FeishuUserID)

	// 转换为卡片需要的格式
	var clusters []map[string]interface{}
	for _, c := range listResp.Clusters {
		status := "Healthy"
		if c.Status != "active" {
			status = "Unavailable"
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

	// 使用交互式按钮卡片
	return &CommandResponse{
		Card: BuildClusterListCardWithActions(clusters, currentCluster),
	}, nil
}

// handleSetCluster 设置当前操作的集群
func (h *ClusterCommandHandler) handleSetCluster(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /cluster set <集群名称>"),
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
		Card: BuildSuccessCard(fmt.Sprintf("✅ 已切换到集群: %s\n\n现在可以直接使用以下命令:\n• /node list - 查看节点列表\n• /node info <节点名> - 查看节点详情\n• /node cordon <节点名> <禁止调度说明> - 禁止调度\n• /node uncordon <节点名> - 恢复调度", clusterName)),
	}, nil
}

// handleClusterStatus handles the cluster status command
func (h *ClusterCommandHandler) handleClusterStatus(ctx *CommandContext) (*CommandResponse, error) {
	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /cluster status <cluster_name>"),
		}, nil
	}

	clusterName := ctx.Command.Args[0]

	// 调用实际的集群服务
	if ctx.Service.clusterService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("集群服务未配置"),
		}, nil
	}

	// 获取集群列表以找到指定集群
	result, err := ctx.Service.clusterService.List(cluster.ListRequest{
		Page:     1,
		PageSize: 100,
		Name:     clusterName,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取集群信息失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取集群信息失败: %s", err.Error())),
		}, nil
	}

	// 类型断言
	listResp, ok := result.(*cluster.ListResponse)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("数据格式错误"),
		}, nil
	}

	if len(listResp.Clusters) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("未找到集群: %s", clusterName)),
		}, nil
	}

	c := listResp.Clusters[0]

	// 构建状态卡片
	statusIcon := "🟢"
	statusText := "正常"
	if c.Status != "active" {
		statusIcon = "🔴"
		statusText = "不可用"
	}

	// 默认假设所有节点都是健康的，如果状态不正常则显示0
	healthyNodes := c.NodeCount
	unhealthyNodes := 0
	if c.Status != "active" {
		healthyNodes = 0
		unhealthyNodes = c.NodeCount
	}

	return &CommandResponse{
		Card: BuildClusterStatusCard(c.Name, statusIcon, statusText, c.NodeCount, healthyNodes, unhealthyNodes),
	}, nil
}

// Description returns the command description
func (h *ClusterCommandHandler) Description() string {
	return "集群管理命令"
}
