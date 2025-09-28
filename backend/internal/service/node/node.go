package node

import (
	"context"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"
	"strings"
	"time"
)

// Service 节点管理服务
type Service struct {
	logger      *logger.Logger
	k8sSvc      *k8s.Service
	auditSvc    *audit.Service
	progressSvc *progress.Service
}

// ListRequest 节点列表请求
type ListRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	Status      string `json:"status"`
	Role        string `json:"role"`
	LabelKey    string `json:"label_key"`
	LabelValue  string `json:"label_value"`
	TaintKey    string `json:"taint_key"`
	TaintValue  string `json:"taint_value"`
	TaintEffect string `json:"taint_effect"`
}

// GetRequest 获取节点详情请求
type GetRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name" binding:"required"`
}

// CordonRequest 禁止调度节点请求
type CordonRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name"` // 从URL路径参数获取，不需要binding验证
	Reason      string `json:"reason"`    // 禁止调度的原因说明
}

// BatchNodeRequest 批量节点操作请求
type BatchNodeRequest struct {
	ClusterName string   `json:"cluster_name"`
	Nodes       []string `json:"nodes" binding:"required"`
	Reason      string   `json:"reason"` // 批量操作的原因说明
}

// DrainRequest 节点驱逐请求
type DrainRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name"` // 从URL路径参数获取，不需要binding验证
	Reason      string `json:"reason"`    // 驱逐的原因说明
}

// CordonInfoRequest 获取禁止调度信息请求
type CordonInfoRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name" binding:"required"`
}

// NodeMetrics 节点指标
type NodeMetrics struct {
	NodeName    string              `json:"node_name"`
	CPUUsage    string              `json:"cpu_usage"`
	MemoryUsage string              `json:"memory_usage"`
	PodCount    int                 `json:"pod_count"`
	PodCapacity int                 `json:"pod_capacity"`
	Conditions  []k8s.NodeCondition `json:"conditions"`
}

// NodeSummary 节点摘要
type NodeSummary struct {
	Total       int `json:"total"`
	Ready       int `json:"ready"`
	NotReady    int `json:"not_ready"`
	Schedulable int `json:"schedulable"`
	Masters     int `json:"masters"`
	Workers     int `json:"workers"`
}

// CordonHistoryResponse 禁止调度历史响应
type CordonHistoryResponse struct {
	NodeName     string    `json:"node_name"`
	Reason       string    `json:"reason"`
	OperatorName string    `json:"operator_name"`
	OperatorID   uint      `json:"operator_id"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewService 创建新的节点管理服务实例
func NewService(logger *logger.Logger, k8sSvc *k8s.Service, auditSvc *audit.Service) *Service {
	return &Service{
		logger:   logger,
		k8sSvc:   k8sSvc,
		auditSvc: auditSvc,
	}
}

// List 获取节点列表
func (s *Service) List(req ListRequest, userID uint) ([]k8s.NodeInfo, error) {
	nodes, err := s.k8sSvc.ListNodes(req.ClusterName)
	if err != nil {
		s.logger.Errorf("Failed to list nodes for cluster %s: %v", req.ClusterName, err)
		// 尝试获取集群ID以正确记录审计日志
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			Action:       model.ActionView,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to list nodes for cluster %s: %s", req.ClusterName, err.Error()),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		// 保持原始错误信息以便前端显示
		return nil, err
	}

	// 应用过滤器
	filteredNodes := s.filterNodes(nodes, req)

	// 检查并同步被禁止调度的节点的annotations信息到审计日志
	// 同步方法内部会检查是否有 deeproute.cn/kube-node-mgr annotation，没有的话会跳过
	go func() {
		var cordonedNodes []string
		for _, node := range filteredNodes {
			if !node.Schedulable { // 节点被禁止调度
				cordonedNodes = append(cordonedNodes, node.Name)
			}
		}

		if len(cordonedNodes) > 0 {
			if err := s.BatchSyncCordonAnnotationsToAudit(req.ClusterName, cordonedNodes); err != nil {
				s.logger.Warningf("Failed to sync cordon annotations for nodes in cluster %s: %v", req.ClusterName, err)
			}
		}
	}()

	// 只在有特定过滤条件时记录审计日志，避免频繁记录普通列表查看
	if req.Status != "" || req.Role != "" || req.LabelKey != "" || req.TaintKey != "" {
		// 获取集群ID以正确记录审计日志
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			Action:       model.ActionView,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Filtered nodes for cluster %s with conditions", req.ClusterName),
			Status:       model.AuditStatusSuccess,
		})
	}

	return filteredNodes, nil
}

// Get 获取单个节点详情
func (s *Service) Get(req GetRequest, userID uint) (*k8s.NodeInfo, error) {
	node, err := s.k8sSvc.GetNode(req.ClusterName, req.NodeName)
	if err != nil {
		s.logger.Errorf("Failed to get node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     req.NodeName,
			Action:       model.ActionView,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to get node %s for cluster %s", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// 如果节点被禁止调度，检查并同步annotations信息到审计日志
	// 同步方法内部会检查是否有 deeproute.cn/kube-node-mgr annotation，没有的话会跳过
	if !node.Schedulable {
		go func() {
			if err := s.SyncCordonAnnotationsToAudit(req.ClusterName, req.NodeName); err != nil {
				s.logger.Warningf("Failed to sync cordon annotations for node %s: %v", req.NodeName, err)
			}
		}()
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     req.NodeName,
		Action:       model.ActionView,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Viewed node %s for cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return node, nil
}

// Cordon 禁止调度节点（标记为不可调度）
func (s *Service) Cordon(req CordonRequest, userID uint) error {
	// 执行禁止调度操作（仅设置不可调度，不删除pods），并添加原因注释
	err := s.k8sSvc.CordonNodeWithReason(req.ClusterName, req.NodeName, req.Reason)
	if err != nil {
		s.logger.Errorf("Failed to cordon node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
		reasonMsg := ""
		if req.Reason != "" {
			reasonMsg = fmt.Sprintf(" (原因: %s)", req.Reason)
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to cordon node %s for cluster %s%s", req.NodeName, req.ClusterName, reasonMsg),
			Reason:       req.Reason,
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	s.logger.Infof("Successfully cordoned node %s for cluster %s", req.NodeName, req.ClusterName)
	reasonMsg := ""
	if req.Reason != "" {
		reasonMsg = fmt.Sprintf(" (原因: %s)", req.Reason)
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Cordoned node %s for cluster %s%s", req.NodeName, req.ClusterName, reasonMsg),
		Reason:       req.Reason,
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// Uncordon 解除调度节点（标记为可调度）
func (s *Service) Uncordon(req CordonRequest, userID uint) error {
	err := s.k8sSvc.UncordonNode(req.ClusterName, req.NodeName)
	if err != nil {
		s.logger.Errorf("Failed to uncordon node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to uncordon node %s for cluster %s", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to uncordon node: %w", err)
	}

	s.logger.Infof("Successfully uncordoned node %s for cluster %s", req.NodeName, req.ClusterName)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Uncordoned node %s for cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// GetSummary 获取节点摘要信息
func (s *Service) GetSummary(clusterName string, userID uint) (*NodeSummary, error) {
	nodes, err := s.k8sSvc.ListNodes(clusterName)
	if err != nil {
		s.logger.Errorf("Failed to get node summary for cluster %s: %v", clusterName, err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	summary := &NodeSummary{}
	summary.Total = len(nodes)

	for _, node := range nodes {
		// 统计状态
		if node.Status == "Ready" {
			summary.Ready++
		} else {
			summary.NotReady++
		}

		// 统计可调度状态
		if node.Status != "SchedulingDisabled" {
			summary.Schedulable++
		}

		// 统计角色
		for _, role := range node.Roles {
			switch strings.ToLower(role) {
			case "master", "control-plane":
				summary.Masters++
			case "worker", "":
				summary.Workers++
			}
		}
	}

	// 如果没有明确的master节点，但有worker节点，调整计数
	if summary.Masters == 0 && summary.Workers > 0 {
		// 可能所有节点都是worker节点
	} else if summary.Masters > 0 && summary.Workers == 0 {
		// 可能是单节点集群
		summary.Workers = summary.Total - summary.Masters
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionView,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Viewed node summary for cluster %s", clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return summary, nil
}

// filterNodes 过滤节点
func (s *Service) filterNodes(nodes []k8s.NodeInfo, req ListRequest) []k8s.NodeInfo {
	var filtered []k8s.NodeInfo

	for _, node := range nodes {
		// 状态过滤
		if req.Status != "" && !strings.EqualFold(node.Status, req.Status) {
			continue
		}

		// 角色过滤
		if req.Role != "" {
			hasRole := false
			for _, role := range node.Roles {
				if strings.EqualFold(role, req.Role) {
					hasRole = true
					break
				}
			}
			if !hasRole {
				continue
			}
		}

		// 标签过滤
		if req.LabelKey != "" {
			if req.LabelValue != "" {
				// 精确匹配标签键值对
				if value, exists := node.Labels[req.LabelKey]; !exists || value != req.LabelValue {
					continue
				}
			} else {
				// 只检查标签键是否存在
				if _, exists := node.Labels[req.LabelKey]; !exists {
					continue
				}
			}
		}

		// 污点过滤
		if req.TaintKey != "" {
			found := false
			for _, taint := range node.Taints {
				// 检查污点键匹配
				if taint.Key == req.TaintKey {
					// 如果指定了污点值，进行值匹配
					if req.TaintValue != "" && taint.Value != req.TaintValue {
						continue
					}
					// 如果指定了污点效果，进行效果匹配
					if req.TaintEffect != "" && taint.Effect != req.TaintEffect {
						continue
					}
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		filtered = append(filtered, node)
	}

	return filtered
}

// GetMetrics 获取节点指标（简化版本，实际环境中可能需要集成Prometheus等监控系统）
func (s *Service) GetMetrics(clusterName, nodeName string, userID uint) (*NodeMetrics, error) {
	node, err := s.k8sSvc.GetNode(clusterName, nodeName)
	if err != nil {
		s.logger.Errorf("Failed to get node metrics for %s: %v", nodeName, err)
		return nil, fmt.Errorf("failed to get node: %w", err)
	}

	// 计算资源使用率（这里是简化版本，实际应该从监控系统获取）
	metrics := &NodeMetrics{
		NodeName:    node.Name,
		CPUUsage:    "N/A", // 需要从监控系统获取
		MemoryUsage: "N/A", // 需要从监控系统获取
		PodCount:    0,     // 需要统计实际运行的Pod数量
		Conditions:  node.Conditions,
	}

	// 从节点信息中提取Pod容量
	if node.Capacity.Pods != "" {
		// 这里应该解析Pod容量，但为了简化直接设置为0
		metrics.PodCapacity = 0
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     nodeName,
		Action:       model.ActionView,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Viewed metrics for node %s in cluster %s", nodeName, clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return metrics, nil
}

// GetNodesByLabels 根据标签获取节点
func (s *Service) GetNodesByLabels(clusterName string, labels map[string]string, userID uint) ([]k8s.NodeInfo, error) {
	nodes, err := s.k8sSvc.ListNodes(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var matchingNodes []k8s.NodeInfo
	for _, node := range nodes {
		matches := true
		for key, value := range labels {
			if nodeValue, exists := node.Labels[key]; !exists || nodeValue != value {
				matches = false
				break
			}
		}
		if matches {
			matchingNodes = append(matchingNodes, node)
		}
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionView,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Searched nodes by labels in cluster %s", clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return matchingNodes, nil
}

// ValidateNodeOperation 验证节点操作权限
func (s *Service) ValidateNodeOperation(clusterName, nodeName string, operation string) error {
	// 获取节点信息进行验证
	_, err := s.k8sSvc.GetNode(clusterName, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node for validation: %w", err)
	}

	switch operation {
	case "delete":
		// 通常不允许通过此API删除节点
		return fmt.Errorf("node deletion is not allowed through this API")
	}

	return nil
}

// BatchCordon 批量禁止调度节点
func (s *Service) BatchCordon(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]string)
	successful := make([]string, 0)

	for _, nodeName := range req.Nodes {
		cordonReq := CordonRequest{
			ClusterName: req.ClusterName,
			NodeName:    nodeName,
			Reason:      req.Reason,
		}

		if err := s.Cordon(cordonReq, userID); err != nil {
			errors[nodeName] = err.Error()
			s.logger.Errorf("Failed to cordon node %s: %v", nodeName, err)
		} else {
			successful = append(successful, nodeName)
		}
	}

	results["successful"] = successful
	results["errors"] = errors
	results["total"] = len(req.Nodes)
	results["success_count"] = len(successful)
	results["error_count"] = len(errors)

	// 记录审计日志
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Batch cordon %d nodes in cluster %s: %d successful, %d failed", len(req.Nodes), req.ClusterName, len(successful), len(errors)),
		Status:       model.AuditStatusSuccess,
	})

	return results, nil
}

// BatchUncordon 批量解除调度节点
func (s *Service) BatchUncordon(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]string)
	successful := make([]string, 0)

	for _, nodeName := range req.Nodes {
		uncordonReq := CordonRequest{
			ClusterName: req.ClusterName,
			NodeName:    nodeName,
			Reason:      req.Reason,
		}

		if err := s.Uncordon(uncordonReq, userID); err != nil {
			errors[nodeName] = err.Error()
			s.logger.Errorf("Failed to uncordon node %s: %v", nodeName, err)
		} else {
			successful = append(successful, nodeName)
		}
	}

	results["successful"] = successful
	results["errors"] = errors
	results["total"] = len(req.Nodes)
	results["success_count"] = len(successful)
	results["error_count"] = len(errors)

	// 记录审计日志
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Batch uncordon %d nodes in cluster %s: %d successful, %d failed", len(req.Nodes), req.ClusterName, len(successful), len(errors)),
		Status:       model.AuditStatusSuccess,
	})

	return results, nil
}

// Drain 驱逐节点
func (s *Service) Drain(req DrainRequest, userID uint) error {
	s.logger.Infof("User %d initiating drain operation on node %s in cluster %s", userID, req.NodeName, req.ClusterName)

	// 获取集群ID以正确记录审计日志
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}

	// 调用k8s服务进行节点驱逐
	if err := s.k8sSvc.DrainNode(req.ClusterName, req.NodeName, req.Reason); err != nil {
		s.logger.Errorf("Failed to drain node %s in cluster %s: %v", req.NodeName, req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to drain node %s in cluster %s", req.NodeName, req.ClusterName),
			Reason:       req.Reason,
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to drain node: %w", err)
	}

	// 记录禁止调度的审计日志（这样就不需要依赖后续的同步过程）
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Cordoned node %s in cluster %s", req.NodeName, req.ClusterName),
		Reason:       req.Reason,
		Status:       model.AuditStatusSuccess,
	})

	// 记录驱逐操作的审计日志
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Drained node %s in cluster %s", req.NodeName, req.ClusterName),
		Reason:       req.Reason,
		Status:       model.AuditStatusSuccess,
	})

	s.logger.Infof("Successfully drained node %s in cluster %s", req.NodeName, req.ClusterName)
	return nil
}

// BatchDrain 批量驱逐节点
func (s *Service) BatchDrain(req BatchNodeRequest, userID uint) (map[string]interface{}, error) {
	results := make(map[string]interface{})
	errors := make(map[string]string)
	successful := make([]string, 0)

	s.logger.Infof("User %d initiating batch drain operation on %d nodes in cluster %s", userID, len(req.Nodes), req.ClusterName)

	for _, nodeName := range req.Nodes {
		drainReq := DrainRequest{
			ClusterName: req.ClusterName,
			NodeName:    nodeName,
			Reason:      req.Reason,
		}

		if err := s.Drain(drainReq, userID); err != nil {
			errors[nodeName] = err.Error()
			s.logger.Errorf("Failed to drain node %s: %v", nodeName, err)
		} else {
			successful = append(successful, nodeName)
		}
	}

	results["successful"] = successful
	results["errors"] = errors
	results["total"] = len(req.Nodes)
	results["success_count"] = len(successful)
	results["error_count"] = len(errors)

	// 记录审计日志
	status := model.AuditStatusSuccess
	if len(errors) == len(req.Nodes) {
		status = model.AuditStatusFailed
	}

	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Batch drain %d nodes in cluster %s: %d successful, %d failed", len(req.Nodes), req.ClusterName, len(successful), len(errors)),
		Reason:       req.Reason,
		Status:       status,
	})

	return results, nil
}

// GetNodeCordonInfo 获取节点的禁止调度信息（从annotations）
func (s *Service) GetNodeCordonInfo(clusterName, nodeName string) (map[string]interface{}, error) {
	return s.k8sSvc.GetNodeCordonInfo(clusterName, nodeName)
}

// SetProgressService 设置进度推送服务
func (s *Service) SetProgressService(progressSvc *progress.Service) {
	s.progressSvc = progressSvc
}

// CordonProcessor 禁止调度处理器
type CordonProcessor struct {
	svc         *Service
	clusterName string
	reason      string
	userID      uint
}

func (p *CordonProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	req := CordonRequest{
		ClusterName: p.clusterName,
		NodeName:    nodeName,
		Reason:      p.reason,
	}
	return p.svc.Cordon(req, p.userID)
}

// UncordonProcessor 解除调度处理器
type UncordonProcessor struct {
	svc         *Service
	clusterName string
	reason      string
	userID      uint
}

func (p *UncordonProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	req := CordonRequest{
		ClusterName: p.clusterName,
		NodeName:    nodeName,
		Reason:      p.reason,
	}
	return p.svc.Uncordon(req, p.userID)
}

// DrainProcessor 驱逐处理器
type DrainProcessor struct {
	svc         *Service
	clusterName string
	reason      string
	userID      uint
}

func (p *DrainProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	req := DrainRequest{
		ClusterName: p.clusterName,
		NodeName:    nodeName,
		Reason:      p.reason,
	}
	return p.svc.Drain(req, p.userID)
}

// BatchCordonWithProgress 批量禁止调度节点（带进度）
func (s *Service) BatchCordonWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	if s.progressSvc == nil {
		return fmt.Errorf("progress service not set")
	}

	processor := &CordonProcessor{
		svc:         s,
		clusterName: req.ClusterName,
		reason:      req.Reason,
		userID:      userID,
	}

	ctx := context.Background()
	return s.progressSvc.ProcessBatchWithProgress(
		ctx,
		taskID,
		"batch_cordon",
		req.Nodes,
		userID,
		5, // 并发数
		processor,
	)
}

// BatchUncordonWithProgress 批量解除调度节点（带进度）
func (s *Service) BatchUncordonWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	if s.progressSvc == nil {
		return fmt.Errorf("progress service not set")
	}

	processor := &UncordonProcessor{
		svc:         s,
		clusterName: req.ClusterName,
		reason:      req.Reason,
		userID:      userID,
	}

	ctx := context.Background()
	return s.progressSvc.ProcessBatchWithProgress(
		ctx,
		taskID,
		"batch_uncordon",
		req.Nodes,
		userID,
		5, // 并发数
		processor,
	)
}

// BatchDrainWithProgress 批量驱逐节点（带进度）
func (s *Service) BatchDrainWithProgress(req BatchNodeRequest, userID uint, taskID string) error {
	if s.progressSvc == nil {
		return fmt.Errorf("progress service not set")
	}

	processor := &DrainProcessor{
		svc:         s,
		clusterName: req.ClusterName,
		reason:      req.Reason,
		userID:      userID,
	}

	ctx := context.Background()
	return s.progressSvc.ProcessBatchWithProgress(
		ctx,
		taskID,
		"batch_drain",
		req.Nodes,
		userID,
		3, // 驱逐操作较重，减少并发数
		processor,
	)
}

// GetCordonHistory 获取节点的禁止调度历史
func (s *Service) GetCordonHistory(nodeName, clusterName string, userID uint) (*CordonHistoryResponse, error) {
	// 获取正确的集群ID
	clusterID, err := s.getClusterIDByName(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get cluster ID for %s: %v, using ID 0", clusterName, err)
		clusterID = 0 // 作为备用方案
	}

	log, err := s.auditSvc.GetLatestCordonRecord(nodeName, clusterID)
	if err != nil {
		// 如果没有找到记录，返回空的历史
		if err.Error() == "record not found" || strings.Contains(err.Error(), "not found") {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cordon history: %w", err)
	}

	// 检查是否是禁止调度操作（包含"Cordoned"关键词且不包含"Uncordoned"）
	if !strings.Contains(log.Details, "Cordoned") || strings.Contains(log.Details, "Uncordoned") {
		// 如果最新记录是解除调度，返回空历史
		return nil, nil
	}

	response := &CordonHistoryResponse{
		NodeName:     log.NodeName,
		Reason:       log.Reason,
		OperatorName: log.User.Username,
		OperatorID:   log.UserID,
		Timestamp:    log.CreatedAt,
	}

	return response, nil
}

// GetBatchCordonHistory 批量获取节点的禁止调度历史
func (s *Service) GetBatchCordonHistory(nodeNames []string, clusterName string) (map[string]*CordonHistoryResponse, error) {
	if len(nodeNames) == 0 {
		return make(map[string]*CordonHistoryResponse), nil
	}

	// 获取正确的集群ID
	clusterID, err := s.getClusterIDByName(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get cluster ID for %s: %v, using ID 0", clusterName, err)
		clusterID = 0 // 作为备用方案
	}

	logs, err := s.auditSvc.GetLatestCordonRecords(nodeNames, clusterID)
	if err != nil {
		return nil, fmt.Errorf("failed to get batch cordon history: %w", err)
	}

	result := make(map[string]*CordonHistoryResponse)
	for nodeName, log := range logs {
		// 检查是否是禁止调度操作且不是解除调度操作
		if log != nil && strings.Contains(log.Details, "Cordoned") && !strings.Contains(log.Details, "Uncordoned") {
			result[nodeName] = &CordonHistoryResponse{
				NodeName:     log.NodeName,
				Reason:       log.Reason,
				OperatorName: log.User.Username,
				OperatorID:   log.UserID,
				Timestamp:    log.CreatedAt,
			}
		}
	}

	return result, nil
}

// getClusterIDByName 根据集群名称获取集群ID
func (s *Service) getClusterIDByName(clusterName string) (uint, error) {
	return s.auditSvc.GetClusterIDByName(clusterName)
}

// SyncCordonAnnotationsToAudit 同步kubectl-plugin的禁止调度annotations到审计日志
func (s *Service) SyncCordonAnnotationsToAudit(clusterName, nodeName string) error {
	// 获取节点的禁止调度信息
	cordonInfo, err := s.k8sSvc.GetNodeCordonInfo(clusterName, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node cordon info: %w", err)
	}

	// 检查节点是否被禁止调度
	cordoned, exists := cordonInfo["cordoned"]
	if !exists || !cordoned.(bool) {
		return nil // 节点未被禁止调度，无需同步
	}

	// 关键检查：只有存在 deeproute.cn/kube-node-mgr annotation 时才同步
	// 这意味着节点要么是通过 kubectl-plugin 禁止调度，要么是手动添加了我们的 annotation
	if _, hasReason := cordonInfo["reason"]; !hasReason {
		s.logger.Infof("Skipping sync for node %s: no deeproute.cn/kube-node-mgr annotation found (pure kubectl cordon)", nodeName)
		return nil // 纯粹的 kubectl cordon 操作，无需同步
	}

	// 获取集群ID
	clusterID, err := s.getClusterIDByName(clusterName)
	if err != nil {
		s.logger.Warningf("Failed to get cluster ID for %s: %v", clusterName, err)
		return fmt.Errorf("failed to get cluster ID: %w", err)
	}

	// 获取admin用户ID
	adminUserID, err := s.auditSvc.GetAdminUserID()
	if err != nil {
		s.logger.Warningf("Failed to get admin user ID: %v, using default ID 1", err)
		adminUserID = 1 // 默认使用ID为1的用户
	}

	// 检查是否已经有同步的审计记录 - 使用正确的集群ID
	existingLog, _ := s.auditSvc.GetLatestCordonRecord(nodeName, clusterID)

	// 从annotations或taints获取时间戳
	var cordonTime time.Time
	var timestampSource string
	if timestampStr, exists := cordonInfo["timestamp"]; exists {
		if parsedTime, err := time.Parse(time.RFC3339, timestampStr.(string)); err == nil {
			cordonTime = parsedTime
			if source, hasSource := cordonInfo["timestamp_source"]; hasSource {
				timestampSource = source.(string)
			}
		}
	}

	// 获取原因
	reason := ""
	if reasonStr, exists := cordonInfo["reason"]; exists {
		reason = reasonStr.(string)
	}

	// 决定是否需要同步的逻辑
	shouldSync := false
	syncReason := ""

	if existingLog == nil {
		// 情况1: 没有现有记录，直接同步
		shouldSync = true
		syncReason = "no existing record"
	} else {
		// 检查现有记录是否是通过Web界面操作的（非admin用户或包含Cordoned关键词的记录）
		isWebUIOperation := existingLog.UserID != adminUserID ||
			(strings.Contains(existingLog.Details, "Cordoned") &&
				!strings.Contains(existingLog.Details, "kubectl-plugin"))

		if isWebUIOperation {
			s.logger.Infof("Skipping sync for node %s: existing record from Web UI (User ID: %d)",
				nodeName, existingLog.UserID)
			return nil // 跳过同步，保留Web界面的正确用户记录
		}
	}

	if existingLog != nil && !cordonTime.IsZero() {
		// 情况2: 有时间戳（来自kubectl-plugin或K8s taint）
		if existingLog.CreatedAt.Before(cordonTime) {
			shouldSync = true
			if timestampSource == "kubectl_plugin" {
				syncReason = "newer kubectl-plugin timestamp"
			} else {
				syncReason = "newer kubernetes taint timestamp (manual cordon)"
			}
		} else if timestampSource == "kubernetes_taint" {
			// 即使时间相近，如果是从taint获取的时间戳且现有记录不是手动操作记录，也要同步
			isExistingManual := strings.Contains(existingLog.Details, "manual operation")
			if !isExistingManual {
				shouldSync = true
				syncReason = "manual cordon operation detected via taint timestamp"
			}
		}
	} else if existingLog != nil {
		// 情况3: existingLog存在但没有时间戳信息
		// 检查现有记录的类型来决定是否同步
		isExistingFromSync := strings.Contains(existingLog.Details, "synced from kubectl-plugin") ||
			strings.Contains(existingLog.Details, "manual operation")

		if !isExistingFromSync {
			// 现有记录来自web界面等，记录这次手动操作
			shouldSync = true
			syncReason = "manual operation after web operation (no timestamp available)"
		} else {
			// 检查reason是否不同
			if reason != existingLog.Reason {
				shouldSync = true
				syncReason = "different reason from existing record (no timestamp available)"
			} else {
				// 相同信息，不重复同步
				s.logger.Infof("Skipping sync for node %s: similar operation already recorded", nodeName)
			}
		}
	}

	if shouldSync {
		s.logger.Infof("Syncing cordon annotation for node %s: %s", nodeName, syncReason)

		// 创建审计日志记录
		// 根据时间戳来源和annotations确定详情描述
		var details string
		if timestampSource == "kubectl_plugin" {
			// 来自kubectl-plugin的annotations
			details = fmt.Sprintf("Cordoned node %s for cluster %s (synced from kubectl-plugin)", nodeName, clusterName)
		} else if timestampSource == "kubernetes_taint" {
			// 来自kubernetes taint的时间戳，但有我们的annotation，表示手动添加了annotation
			details = fmt.Sprintf("Cordoned node %s for cluster %s (kubectl cordon + manual annotation, synced via taint timestamp)", nodeName, clusterName)
		} else {
			// 没有时间戳信息的手动操作
			details = fmt.Sprintf("Cordoned node %s for cluster %s (manual operation, synced as admin)", nodeName, clusterName)
		}

		logReq := audit.LogRequest{
			UserID:       adminUserID,
			ClusterID:    &clusterID, // 设置正确的集群ID
			NodeName:     nodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      details,
			Reason:       reason,
			Status:       model.AuditStatusSuccess,
		}

		// 如果有时间戳，则使用自定义时间
		if !cordonTime.IsZero() {
			err = s.auditSvc.LogWithCustomTime(logReq, cordonTime)
		} else {
			err = s.auditSvc.LogWithError(logReq)
		}

		if err != nil {
			s.logger.Errorf("Failed to sync cordon annotation to audit log for node %s: %v", nodeName, err)
			return fmt.Errorf("failed to sync to audit log: %w", err)
		}

		s.logger.Infof("Successfully synced kubectl-plugin cordon annotation to audit log for node %s", nodeName)
	}

	return nil
}

// BatchSyncCordonAnnotationsToAudit 批量同步kubectl-plugin的禁止调度annotations到审计日志
func (s *Service) BatchSyncCordonAnnotationsToAudit(clusterName string, nodeNames []string) error {
	errors := make([]string, 0)

	for _, nodeName := range nodeNames {
		if err := s.SyncCordonAnnotationsToAudit(clusterName, nodeName); err != nil {
			errors = append(errors, fmt.Sprintf("Node %s: %s", nodeName, err.Error()))
			s.logger.Errorf("Failed to sync annotations for node %s: %v", nodeName, err)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("failed to sync some nodes: %s", strings.Join(errors, "; "))
	}

	return nil
}
