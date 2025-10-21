package feishu

import (
	"fmt"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/node"
	"kube-node-manager/internal/service/taint"
	"strings"
	"time"
)

// TaintCommandHandler handles taint-related commands
type TaintCommandHandler struct{}

// Handle executes the taint command
func (h *TaintCommandHandler) Handle(ctx *CommandContext) (*CommandResponse, error) {
	// Taint commands require action
	if ctx.Command.Action == "" {
		return &CommandResponse{
			Text: "请指定操作。用法: /taint <list|add|remove> [参数...]",
		}, nil
	}

	switch ctx.Command.Action {
	case "list":
		return h.handleListTaints(ctx)
	case "add":
		return h.handleAddTaint(ctx)
	case "remove":
		return h.handleRemoveTaint(ctx)
	default:
		return &CommandResponse{
			Text: fmt.Sprintf("未知操作: %s。支持的操作: list, add, remove", ctx.Command.Action),
		}, nil
	}
}

// handleListTaints handles the taint list command
func (h *TaintCommandHandler) handleListTaints(ctx *CommandContext) (*CommandResponse, error) {
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
			Card: BuildErrorCard("参数不足。用法: /taint list <节点名>"),
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

	// 构建污点卡片
	return &CommandResponse{
		Card: BuildTaintListCard(foundNode.Taints, nodeName, clusterName),
	}, nil
}

// handleAddTaint handles the taint add command
func (h *TaintCommandHandler) handleAddTaint(ctx *CommandContext) (*CommandResponse, error) {
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
			Card: BuildTaintHelpCard(),
		}, nil
	}

	nodeName := ctx.Command.Args[0]

	// 解析污点 key=value:effect
	taints, err := h.parseTaints(ctx.Command.Args[1:])
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("污点格式错误: %s\n\n正确格式: key=value:effect\nEffect 可选: NoSchedule, PreferNoSchedule, NoExecute", err.Error())),
		}, nil
	}

	// 检查是否有危险操作（NoExecute）需要确认
	hasNoExecute := false
	for _, t := range taints {
		if t.Effect == "NoExecute" {
			hasNoExecute = true
			break
		}
	}

	if hasNoExecute {
		return &CommandResponse{
			Card: BuildTaintNoExecuteWarningCard(nodeName, taints),
		}, nil
	}

	// 调用污点服务添加污点
	if ctx.Service.taintService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("污点服务未配置"),
		}, nil
	}

	err = ctx.Service.taintService.UpdateNodeTaints(taint.UpdateTaintsRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		Taints:      taints,
		Operation:   "add",
	}, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("添加污点失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("添加污点失败: %s", err.Error())),
		}, nil
	}

	// 构建污点显示字符串
	taintStrs := make([]string, 0, len(taints))
	for _, t := range taints {
		taintStrs = append(taintStrs, fmt.Sprintf("%s=%s:%s", t.Key, t.Value, t.Effect))
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 污点添加成功\n\n节点: `%s`\n集群: %s\n污点: %s", nodeName, clusterName, strings.Join(taintStrs, ", "))),
	}, nil
}

// handleRemoveTaint handles the taint remove command
func (h *TaintCommandHandler) handleRemoveTaint(ctx *CommandContext) (*CommandResponse, error) {
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
			Card: BuildErrorCard("参数不足。用法: /taint remove <节点名> <污点key>"),
		}, nil
	}

	nodeName := ctx.Command.Args[0]
	taintKey := ctx.Command.Args[1]

	// 调用污点服务删除污点
	if ctx.Service.taintService == nil {
		return &CommandResponse{
			Card: BuildErrorCard("污点服务未配置"),
		}, nil
	}

	err = ctx.Service.taintService.RemoveTaint(clusterName, nodeName, taintKey, ctx.UserMapping.SystemUserID)

	if err != nil {
		ctx.Service.logger.Error(fmt.Sprintf("删除污点失败: %v", err))
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("删除污点失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 污点删除成功\n\n节点: `%s`\n集群: %s\n删除的污点: %s", nodeName, clusterName, taintKey)),
	}, nil
}

// parseTaints parses taint arguments in format: key=value:effect
func (h *TaintCommandHandler) parseTaints(args []string) ([]k8s.TaintInfo, error) {
	var taints []k8s.TaintInfo

	// 合并所有参数
	combined := strings.Join(args, " ")

	// 按逗号分隔
	taintStrs := strings.Split(combined, ",")

	validEffects := map[string]bool{
		"NoSchedule":       true,
		"PreferNoSchedule": true,
		"NoExecute":        true,
	}

	for _, taintStr := range taintStrs {
		taintStr = strings.TrimSpace(taintStr)
		if taintStr == "" {
			continue
		}

		// 按冒号分隔 key=value:effect
		parts := strings.Split(taintStr, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid taint format: %s (expected key=value:effect)", taintStr)
		}

		// 解析 key=value
		kvPair := strings.SplitN(parts[0], "=", 2)
		if len(kvPair) != 2 {
			return nil, fmt.Errorf("invalid taint format: %s (expected key=value:effect)", taintStr)
		}

		key := strings.TrimSpace(kvPair[0])
		value := strings.TrimSpace(kvPair[1])
		effect := strings.TrimSpace(parts[1])

		if key == "" {
			return nil, fmt.Errorf("taint key cannot be empty")
		}

		if !validEffects[effect] {
			return nil, fmt.Errorf("invalid effect: %s (must be NoSchedule, PreferNoSchedule, or NoExecute)", effect)
		}

		now := time.Now()
		taints = append(taints, k8s.TaintInfo{
			Key:       key,
			Value:     value,
			Effect:    effect,
			TimeAdded: &now,
		})
	}

	if len(taints) == 0 {
		return nil, fmt.Errorf("no valid taints provided")
	}

	return taints, nil
}

// Description returns the command description
func (h *TaintCommandHandler) Description() string {
	return "污点管理命令"
}
