package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/label"
	"kube-node-manager/internal/service/node"
	"strings"
)

// LabelCommandHandler handles label-related commands
type LabelCommandHandler struct{}

// Handle executes the label command
func (h *LabelCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Label commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /label <list|add|remove|update> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListLabels(ctx)
	case "add":
		return h.handleAddLabel(ctx)
	case "remove":
		return h.handleRemoveLabel(ctx)
	case "update":
		return h.handleUpdateLabel(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, add, remove, update", ctx.Command.Action),
		}, nil
	}
}

// handleListLabels handles the label list command
func (h *LabelCommandHandler) handleListLabels(ctx *CommandContext) (*CommandResponse, error) {
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

	if len(ctx.Command.Args) < 1 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /label list <节点名>"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]

	// 调用节点服务获取节点详情
	if ctx.Service.nodeService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("节点服务未配置"),
		}, nil
	}

	result, err := ctx.Service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("获取节点列表失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点列表失败: %s", err.Error())),
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
			Card: BuildErrorCard(fmt.Sprintf("节点 `%s` 不存在\n\n集群: %s", nodeName, clusterName)),
		}, nil
	}

	// 构建标签卡片
	return &CommandResponse{
		Card: BuildLabelListCard(foundNode.Labels, nodeName, clusterName),
	}, nil
}

// handleAddLabel handles the label add command
func (h *LabelCommandHandler) handleAddLabel(ctx *CommandContext) (*CommandResponse, error) {
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

	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Card: BuildLabelHelpCard(),
		}, nil
	}

	nodeName := ctx.Command.Args[0]

	// 解析标签 key=value 或 key1=val1,key2=val2
	labels, err := h.parseLabels(ctx.Command.Args[1:])
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("标签格式错误: %s\n\n正确格式: key=value 或 key1=val1,key2=val2", err.Error())),
		}, nil
	}

	// 调用标签服务添加标签
	if ctx.Service.labelService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("标签服务未配置"),
		}, nil
	}

	err = ctx.Service.labelService.UpdateNodeLabels(label.UpdateLabelsRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		Labels:      labels,
		Operation:   "add",
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("添加标签失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("添加标签失败: %s", err.Error())),
		}, nil
	}

	// 构建标签显示字符串
	labelStrs := make([]string, 0, len(labels))
	for k, v := range labels {
		labelStrs = append(labelStrs, fmt.Sprintf("%s=%s", k, v))
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 标签添加成功\n\n节点: `%s`\n集群: %s\n标签: %s", nodeName, clusterName, strings.Join(labelStrs, ", "))),
	}, nil
}

// handleRemoveLabel handles the label remove command
func (h *LabelCommandHandler) handleRemoveLabel(ctx *CommandContext) (*CommandResponse, error) {
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

	if len(ctx.Command.Args) < 2 {
		return &CommandResponse{
			Card: BuildErrorCard("参数不足。用法: /label remove <节点名> <标签key>"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]
	// 支持删除多个标签，用逗号分隔
	labelKeys := strings.Split(ctx.Command.Args[1], ",")

	// 构建标签 map（只需要 key）
	labels := make(map[string]string)
	for _, key := range labelKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			labels[key] = "" // value 对于删除操作不重要
		}
	}

	if len(labels) == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("没有指定有效的标签 key"),
		}, nil
	}

	// 调用标签服务删除标签
	if ctx.Service.labelService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("标签服务未配置"),
		}, nil
	}

	err = ctx.Service.labelService.UpdateNodeLabels(label.UpdateLabelsRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		Labels:      labels,
		Operation:   "remove",
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("删除标签失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("删除标签失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 标签删除成功\n\n节点: `%s`\n集群: %s\n删除的标签: %s", nodeName, clusterName, strings.Join(labelKeys, ", "))),
	}, nil
}

// handleUpdateLabel handles the label update command
func (h *LabelCommandHandler) handleUpdateLabel(ctx *CommandContext) (*CommandResponse, error) {
	// update 命令等同于 add（覆盖现有值）
	return h.handleAddLabel(ctx)
}

// parseLabels parses label arguments in format: key=value or key1=val1,key2=val2
func (h *LabelCommandHandler) parseLabels(args []string) (map[string]string, error) {
	labels := make(map[string]string)

	// 合并所有参数
	combined := strings.Join(args, " ")

	// 按逗号分隔
	pairs := strings.Split(combined, ",")

	for _, pair := range pairs {
		pair = strings.TrimSpace(pair)
		if pair == "" {
			continue
		}

		// 按等号分隔
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid label format: %s (expected key=value)", pair)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("label key cannot be empty")
		}

		labels[key] = value
	}

	if len(labels) == 0 {
		return nil, fmt.Errorf("no valid labels provided")
	}

	return labels, nil
}

// Description returns the command description
func (h *LabelCommandHandler) Description() string {
	return "标签管理命令"
}
