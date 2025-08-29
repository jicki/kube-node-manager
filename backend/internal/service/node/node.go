package node

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"strings"
)

// Service 节点管理服务
type Service struct {
	logger   *logger.Logger
	k8sSvc   *k8s.Service
	auditSvc *audit.Service
}

// ListRequest 节点列表请求
type ListRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	Status      string `json:"status"`
	Role        string `json:"role"`
	LabelKey    string `json:"label_key"`
	LabelValue  string `json:"label_value"`
}

// GetRequest 获取节点详情请求
type GetRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name" binding:"required"`
}

// DrainRequest 驱逐节点请求
type DrainRequest struct {
	ClusterName string `json:"cluster_name" binding:"required"`
	NodeName    string `json:"node_name" binding:"required"`
	Force       bool   `json:"force"`
}

// CordonRequest 封锁节点请求
type CordonRequest struct {
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
		s.logger.Error("Failed to list nodes for cluster %s: %v", req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionView,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to list nodes for cluster %s", req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 应用过滤器
	filteredNodes := s.filterNodes(nodes, req)

	// 只在有特定过滤条件时记录审计日志，避免频繁记录普通列表查看
	if req.Status != "" || req.Role != "" || len(req.Labels) > 0 || req.Search != "" {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
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
		s.logger.Error("Failed to get node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
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

// Drain 驱逐节点
func (s *Service) Drain(req DrainRequest, userID uint) error {
	err := s.k8sSvc.DrainNode(req.ClusterName, req.NodeName)
	if err != nil {
		s.logger.Error("Failed to drain node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to drain node %s for cluster %s", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to drain node: %w", err)
	}

	s.logger.Info("Successfully drained node %s for cluster %s", req.NodeName, req.ClusterName)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Drained node %s for cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// Cordon 封锁节点（标记为不可调度）
func (s *Service) Cordon(req CordonRequest, userID uint) error {
	// 获取当前节点信息
	node, err := s.k8sSvc.GetNode(req.ClusterName, req.NodeName)
	if err != nil {
		return fmt.Errorf("failed to get node: %w", err)
	}

	// 如果节点已经是不可调度状态，直接返回
	if node.Status == "SchedulingDisabled" {
		return fmt.Errorf("node %s is already cordoned", req.NodeName)
	}

	// 执行驱逐操作（不删除pods）
	err = s.k8sSvc.DrainNode(req.ClusterName, req.NodeName)
	if err != nil {
		s.logger.Error("Failed to cordon node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceNode,
			Details:      fmt.Sprintf("Failed to cordon node %s for cluster %s", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to cordon node: %w", err)
	}

	s.logger.Info("Successfully cordoned node %s for cluster %s", req.NodeName, req.ClusterName)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Cordoned node %s for cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// Uncordon 取消封锁节点（标记为可调度）
func (s *Service) Uncordon(req CordonRequest, userID uint) error {
	err := s.k8sSvc.UncordonNode(req.ClusterName, req.NodeName)
	if err != nil {
		s.logger.Error("Failed to uncordon node %s for cluster %s: %v", req.NodeName, req.ClusterName, err)
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

	s.logger.Info("Successfully uncordoned node %s for cluster %s", req.NodeName, req.ClusterName)
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
		s.logger.Error("Failed to get node summary for cluster %s: %v", clusterName, err)
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

		filtered = append(filtered, node)
	}

	return filtered
}

// GetMetrics 获取节点指标（简化版本，实际环境中可能需要集成Prometheus等监控系统）
func (s *Service) GetMetrics(clusterName, nodeName string, userID uint) (*NodeMetrics, error) {
	node, err := s.k8sSvc.GetNode(clusterName, nodeName)
	if err != nil {
		s.logger.Error("Failed to get node metrics for %s: %v", nodeName, err)
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
	node, err := s.k8sSvc.GetNode(clusterName, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node for validation: %w", err)
	}

	switch operation {
	case "drain":
		// 检查是否是master节点
		for _, role := range node.Roles {
			if strings.ToLower(role) == "master" || strings.ToLower(role) == "control-plane" {
				return fmt.Errorf("cannot drain master/control-plane node: %s", nodeName)
			}
		}
	case "delete":
		// 通常不允许通过此API删除节点
		return fmt.Errorf("node deletion is not allowed through this API")
	}

	return nil
}
