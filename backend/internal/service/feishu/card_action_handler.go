package feishu

import (
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/node"
)

// CardActionHandler handles card button click actions
type CardActionHandler struct {
	service *Service
}

// NewCardActionHandler creates a new card action handler
func NewCardActionHandler(service *Service) *CardActionHandler {
	return &CardActionHandler{
		service: service,
	}
}

// HandleCardAction processes card button actions
func (h *CardActionHandler) HandleCardAction(actionValue string, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	// Parse action value (JSON format)
	var action map[string]interface{}
	if err := json.Unmarshal([]byte(actionValue), &action); err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("解析操作失败: %s", err.Error())),
		}, nil
	}

	actionType, ok := action["action"].(string)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("无效的操作类型"),
		}, nil
	}

	// 检查用户是否已绑定
	if userMapping == nil || userMapping.SystemUserID == 0 {
		return &CommandResponse{
			Card: BuildErrorCard("❌ 没有权限操作\n\n请联系管理员。"),
		}, nil
	}

	// Route to specific action handler
	switch actionType {
	case "node_info":
		return h.handleNodeInfo(action, userMapping)
	case "node_cordon":
		return h.handleNodeCordon(action, userMapping)
	case "node_uncordon":
		return h.handleNodeUncordon(action, userMapping)
	case "node_refresh":
		return h.handleNodeRefresh(action, userMapping)
	case "cluster_switch":
		return h.handleClusterSwitch(action, userMapping)
	case "cluster_status":
		return h.handleClusterStatus(action, userMapping)
	case "page_prev", "page_next":
		return h.handlePageNavigation(action, userMapping)
	case "confirm_action":
		return h.handleConfirmAction(action, userMapping)
	case "cancel_action":
		return h.handleCancelAction()
	default:
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("未知操作: %s", actionType)),
		}, nil
	}
}

// handleNodeInfo handles node info button click
func (h *CardActionHandler) handleNodeInfo(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	nodeName, _ := action["node"].(string)
	clusterName, _ := action["cluster"].(string)

	if nodeName == "" || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少节点或集群信息"),
		}, nil
	}

	// Get node list to find the specified node
	result, err := h.service.nodeService.List(node.ListRequest{
		ClusterName: clusterName,
	}, userMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("获取节点信息失败: %s", err.Error())),
		}, nil
	}

	// Type assertion
	nodeInfos, ok := result.([]k8s.NodeInfo)
	if !ok {
		return &CommandResponse{
			Card: BuildErrorCard("节点数据格式错误"),
		}, nil
	}

	// Find the specified node
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

	// Convert to card format
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

	// Add resource information
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

	// Add usage information (if available)
	if foundNode.Usage != nil {
		nodeInfo["cpu_usage"] = foundNode.Usage.CPU
		nodeInfo["memory_usage"] = foundNode.Usage.Memory
	}

	return &CommandResponse{
		Card: BuildNodeInfoCard(nodeInfo),
	}, nil
}

// handleNodeCordon handles node cordon button click
func (h *CardActionHandler) handleNodeCordon(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	nodeName, _ := action["node"].(string)
	clusterName, _ := action["cluster"].(string)

	if nodeName == "" || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少节点或集群信息"),
		}, nil
	}

	// Execute cordon
	reason := fmt.Sprintf("系统维护 by %s", userMapping.Username)
	err := h.service.nodeService.Cordon(node.CordonRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
		Reason:      reason,
	}, userMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("禁止调度失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功禁止调度\n\n节点: `%s`\n集群: %s", nodeName, clusterName)),
	}, nil
}

// handleNodeUncordon handles node uncordon button click
func (h *CardActionHandler) handleNodeUncordon(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	nodeName, _ := action["node"].(string)
	clusterName, _ := action["cluster"].(string)

	if nodeName == "" || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少节点或集群信息"),
		}, nil
	}

	// Execute uncordon
	err := h.service.nodeService.Uncordon(node.CordonRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
	}, userMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("恢复调度失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 节点已成功恢复调度\n\n节点: `%s`\n集群: %s", nodeName, clusterName)),
	}, nil
}

// handleNodeRefresh handles node refresh button click
func (h *CardActionHandler) handleNodeRefresh(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	nodeName, _ := action["node"].(string)
	clusterName, _ := action["cluster"].(string)

	if nodeName == "" || clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少节点或集群信息"),
		}, nil
	}

	// Get fresh node info
	_, err := h.service.nodeService.Get(node.GetRequest{
		ClusterName: clusterName,
		NodeName:    nodeName,
	}, userMapping.SystemUserID)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("刷新节点信息失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Text: fmt.Sprintf("✅ 节点 %s 信息已刷新", nodeName),
	}, nil
}

// handleClusterSwitch handles cluster switch button click
func (h *CardActionHandler) handleClusterSwitch(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	clusterName, _ := action["cluster"].(string)

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少集群信息"),
		}, nil
	}

	// Switch cluster
	err := h.service.SetCurrentCluster(userMapping.FeishuUserID, clusterName)
	if err != nil {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("切换集群失败: %s", err.Error())),
		}, nil
	}

	return &CommandResponse{
		Card: BuildSuccessCard(fmt.Sprintf("✅ 已切换到集群: `%s`\n\n您可以使用 /node list 查看节点列表", clusterName)),
	}, nil
}

// handleClusterStatus handles cluster status button click
func (h *CardActionHandler) handleClusterStatus(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	clusterName, _ := action["cluster"].(string)

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少集群信息"),
		}, nil
	}

	// Get cluster status (simplified - actual implementation would use cluster service)
	return &CommandResponse{
		Text: fmt.Sprintf("查看集群 %s 的状态", clusterName),
	}, nil
}

// handlePageNavigation handles pagination button clicks
func (h *CardActionHandler) handlePageNavigation(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	page, _ := action["page"].(float64)
	clusterName, _ := action["cluster"].(string)

	if clusterName == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少集群信息"),
		}, nil
	}

	// Get node list and build paginated card
	// This is simplified; actual implementation would fetch nodes and paginate
	return &CommandResponse{
		Text: fmt.Sprintf("显示集群 %s 的第 %.0f 页", clusterName, page),
	}, nil
}

// handleConfirmAction handles confirmed dangerous actions
func (h *CardActionHandler) handleConfirmAction(action map[string]interface{}, userMapping *model.FeishuUserMapping) (*CommandResponse, error) {
	command, _ := action["command"].(string)

	if command == "" {
		return &CommandResponse{
			Card: BuildErrorCard("缺少确认命令"),
		}, nil
	}

	// Parse and execute the confirmed command
	cmd := ParseCommand(command)

	// Execute command
	ctx := &CommandContext{
		Command:     cmd,
		Service:     h.service,
		UserMapping: userMapping,
	}

	handler, exists := h.service.commandRouter.handlers[cmd.Name]
	if !exists {
		return &CommandResponse{
			Card: BuildErrorCard(fmt.Sprintf("未知命令: %s", cmd.Name)),
		}, nil
	}

	return handler.Handle(ctx)
}

// handleCancelAction handles cancel button click
func (h *CardActionHandler) handleCancelAction() (*CommandResponse, error) {
	return &CommandResponse{
		Card: BuildSuccessCard("❌ 操作已取消"),
	}, nil
}
