package taint

import (
	"context"
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/pkg/logger"
	"strings"
	"time"

	"gorm.io/gorm"
)

// Service 污点管理服务
type Service struct {
	db          *gorm.DB
	logger      *logger.Logger
	auditSvc    *audit.Service
	k8sSvc      *k8s.Service
	progressSvc *progress.Service
}

// UpdateTaintsRequest 更新节点污点请求
type UpdateTaintsRequest struct {
	ClusterName string          `json:"cluster_name" binding:"required"`
	NodeName    string          `json:"node_name" binding:"required"`
	Taints      []k8s.TaintInfo `json:"taints" binding:"required"`
	Operation   string          `json:"operation"` // add, remove, replace
}

// BatchUpdateRequest 批量更新污点请求
type BatchUpdateRequest struct {
	ClusterName string          `json:"cluster_name" binding:"required"`
	NodeNames   []string        `json:"node_names" binding:"required"`
	Taints      []k8s.TaintInfo `json:"taints" binding:"required"`
	Operation   string          `json:"operation"` // add, remove, replace
}

// CopyTaintsRequest 复制污点请求
type CopyTaintsRequest struct {
	ClusterName    string `json:"cluster_name" binding:"required"`
	SourceNodeName string `json:"source_node_name" binding:"required"`
	TargetNodeName string `json:"target_node_name" binding:"required"`
}

// BatchCopyTaintsRequest 批量复制污点请求
type BatchCopyTaintsRequest struct {
	ClusterName     string   `json:"cluster_name" binding:"required"`
	SourceNodeName  string   `json:"source_node_name" binding:"required"`
	TargetNodeNames []string `json:"target_node_names" binding:"required"`
}

// TemplateCreateRequest 创建污点模板请求
type TemplateCreateRequest struct {
	Name        string          `json:"name" binding:"required"`
	Description string          `json:"description"`
	Taints      []k8s.TaintInfo `json:"taints" binding:"required"`
}

// TemplateUpdateRequest 更新污点模板请求
type TemplateUpdateRequest struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Taints      []k8s.TaintInfo `json:"taints"`
}

// TemplateListRequest 模板列表请求
type TemplateListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Name     string `json:"name"`
	Search   string `json:"search"` // 搜索关键词
	Effect   string `json:"effect"` // 按效果筛选
}

// TemplateListResponse 模板列表响应
type TemplateListResponse struct {
	Templates []TemplateInfo `json:"templates"`
	Total     int64          `json:"total"`
	Page      int            `json:"page"`
	PageSize  int            `json:"page_size"`
}

// TemplateInfo 模板信息
type TemplateInfo struct {
	ID          uint            `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Taints      []k8s.TaintInfo `json:"taints"`
	CreatedBy   uint            `json:"created_by"`
	CreatedAt   string          `json:"created_at"`
	UpdatedAt   string          `json:"updated_at"`
	Creator     model.User      `json:"creator"`
}

// ApplyTemplateRequest 应用模板请求
type ApplyTemplateRequest struct {
	ClusterName string          `json:"cluster_name" binding:"required"`
	NodeNames   []string        `json:"node_names" binding:"required"`
	TemplateID  uint            `json:"template_id" binding:"required"`
	Operation   string          `json:"operation"`        // add, replace
	Taints      []k8s.TaintInfo `json:"taints,omitempty"` // 前端选择的污点值
}

// TaintUsage 污点使用情况
type TaintUsage struct {
	Key       string   `json:"key"`
	Values    []string `json:"values"`
	Effects   []string `json:"effects"`
	NodeCount int      `json:"node_count"`
	Nodes     []string `json:"nodes,omitempty"`
}

// TaintOperation 污点操作类型
const (
	TaintOperationAdd     = "add"
	TaintOperationRemove  = "remove"
	TaintOperationReplace = "replace"
)

// TaintEffect 污点效果
const (
	TaintEffectNoSchedule       = "NoSchedule"
	TaintEffectPreferNoSchedule = "PreferNoSchedule"
	TaintEffectNoExecute        = "NoExecute"
)

// NewService 创建新的污点管理服务实例
func NewService(db *gorm.DB, logger *logger.Logger, auditSvc *audit.Service, k8sSvc *k8s.Service) *Service {
	return &Service{
		db:       db,
		logger:   logger,
		auditSvc: auditSvc,
		k8sSvc:   k8sSvc,
	}
}

// SetProgressService 设置进度推送服务
func (s *Service) SetProgressService(progressSvc *progress.Service) {
	s.progressSvc = progressSvc
}

// getClusterIDByName 根据集群名称获取集群ID
func (s *Service) getClusterIDByName(clusterName string) (uint, error) {
	return s.auditSvc.GetClusterIDByName(clusterName)
}

// UpdateNodeTaints 更新单个节点污点
func (s *Service) UpdateNodeTaints(req UpdateTaintsRequest, userID uint) error {
	// 验证污点信息
	if err := s.validateTaints(req.Taints, req.Operation); err != nil {
		return fmt.Errorf("invalid taints: %w", err)
	}

	// 获取当前节点信息，强制刷新缓存确保获取最新的污点
	currentNode, err := s.k8sSvc.GetNodeWithCache(req.ClusterName, req.NodeName, true)
	if err != nil {
		s.logger.Errorf("Failed to get node %s in cluster %s: %v", req.NodeName, req.ClusterName, err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to update taints for node %s in cluster %s: node not found or inaccessible", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to get node %s in cluster %s: %w", req.NodeName, req.ClusterName, err)
	}

	// 准备更新的污点
	var updatedTaints []k8s.TaintInfo

	switch strings.ToLower(req.Operation) {
	case TaintOperationAdd, "":
		// 添加污点，保留现有污点
		updatedTaints = append(updatedTaints, currentNode.Taints...)
		for _, newTaint := range req.Taints {
			// 检查是否已存在相同键的污点，如果存在则更新
			found := false
			for i, existingTaint := range updatedTaints {
				if existingTaint.Key == newTaint.Key {
					updatedTaints[i] = newTaint
					found = true
					break
				}
			}
			if !found {
				updatedTaints = append(updatedTaints, newTaint)
			}
		}
	case TaintOperationRemove:
		// 删除指定的污点
		for _, existingTaint := range currentNode.Taints {
			shouldRemove := false
			for _, removeTaint := range req.Taints {
				if existingTaint.Key == removeTaint.Key {
					shouldRemove = true
					break
				}
			}
			if !shouldRemove {
				updatedTaints = append(updatedTaints, existingTaint)
			}
		}
	case TaintOperationReplace:
		// 替换所有污点
		updatedTaints = req.Taints
	default:
		return fmt.Errorf("invalid operation: %s", req.Operation)
	}

	// 设置时间戳
	now := time.Now()
	for i := range updatedTaints {
		if updatedTaints[i].TimeAdded == nil {
			updatedTaints[i].TimeAdded = &now
		}
	}

	// 更新节点污点
	updateReq := k8s.TaintUpdateRequest{
		NodeName: req.NodeName,
		Taints:   updatedTaints,
	}

	if err := s.k8sSvc.UpdateNodeTaints(req.ClusterName, updateReq); err != nil {
		s.logger.Errorf("Failed to update node taints: %v", err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to update taints for node %s", req.NodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to update node taints: %w", err)
	}

	s.logger.Infof("Successfully updated taints for node %s", req.NodeName)
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Updated taints for node %s in cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// TaintProcessor 实现 BatchProcessor 接口
type TaintProcessor struct {
	svc    *Service
	req    BatchUpdateRequest
	userID uint
}

func (p *TaintProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	p.svc.logger.Infof("[ProcessNode] Starting for node %s (index %d)", nodeName, index)
	
	updateReq := UpdateTaintsRequest{
		ClusterName: p.req.ClusterName,
		NodeName:    nodeName,
		Taints:      p.req.Taints,
		Operation:   p.req.Operation,
	}

	p.svc.logger.Infof("[ProcessNode] Calling UpdateNodeTaints for node %s", nodeName)
	err := p.svc.UpdateNodeTaints(updateReq, p.userID)
	
	if err != nil {
		p.svc.logger.Errorf("[ProcessNode] Failed for node %s: %v", nodeName, err)
	} else {
		p.svc.logger.Infof("[ProcessNode] Completed successfully for node %s", nodeName)
	}
	
	return err
}

// BatchUpdateTaints 批量更新节点污点 (带进度推送)
func (s *Service) BatchUpdateTaints(req BatchUpdateRequest, userID uint) error {
	return s.BatchUpdateTaintsWithProgress(req, userID, "")
}

// BatchUpdateTaintsWithProgress 批量更新节点污点 (带进度推送)
func (s *Service) BatchUpdateTaintsWithProgress(req BatchUpdateRequest, userID uint, taskID string) error {
	s.logger.Infof("Starting batch taint update for %d nodes in cluster %s", len(req.NodeNames), req.ClusterName)

	// 注意：使用 Informer + WebSocket 实时同步后，无需手动清除缓存
	// Informer 会自动检测到节点变化并通过 WebSocket 推送给前端

	// 如果提供了taskID，则使用进度推送
	if taskID != "" && s.progressSvc != nil {
		processor := &TaintProcessor{
			svc:    s,
			req:    req,
			userID: userID,
		}

		// 使用进度推送的并发处理
		maxConcurrency := 5 // 限制并发数避免过载
		if err := s.progressSvc.ProcessBatchWithProgress(
			context.Background(),
			taskID,
			"batch_taint",
			req.NodeNames,
			userID,
			maxConcurrency,
			processor,
		); err != nil {
			var clusterID *uint
			if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
				clusterID = &cID
			}
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				ClusterID:    clusterID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceTaint,
				Details:      fmt.Sprintf("Batch update taints failed for %d nodes", len(req.NodeNames)),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     err.Error(),
			})
			return err
		}
	} else {
		// 传统的顺序处理方式（向后兼容）
		var errors []string
		successCount := 0
		for _, nodeName := range req.NodeNames {
			updateReq := UpdateTaintsRequest{
				ClusterName: req.ClusterName,
				NodeName:    nodeName,
				Taints:      req.Taints,
				Operation:   req.Operation,
			}

			if err := s.UpdateNodeTaints(updateReq, userID); err != nil {
				errorMsg := fmt.Sprintf("Node %s: %v", nodeName, err)
				errors = append(errors, errorMsg)
				s.logger.Errorf("Failed to update taints for node %s: %v", nodeName, err)
			} else {
				successCount++
			}
		}

		if len(errors) > 0 {
			combinedError := strings.Join(errors, "; ")
			var clusterID *uint
			if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
				clusterID = &cID
			}
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				ClusterID:    clusterID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceTaint,
				Details:      fmt.Sprintf("Batch update taints failed for %d nodes", len(errors)),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     combinedError,
			})
			return fmt.Errorf("batch update failed for some nodes: %s", combinedError)
		}
	}

	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Batch updated taints for %d nodes in cluster %s", len(req.NodeNames), req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// CreateTemplate 创建污点模板
func (s *Service) CreateTemplate(req TemplateCreateRequest, userID uint) (*TemplateInfo, error) {
	// 验证污点信息
	if err := s.validateTaints(req.Taints, "add"); err != nil {
		return nil, fmt.Errorf("invalid taints: %w", err)
	}

	// 检查模板名称是否已存在（包括未删除的记录）
	var existingTemplate model.TaintTemplate
	if err := s.db.Where("name = ?", req.Name).First(&existingTemplate).Error; err == nil {
		return nil, fmt.Errorf("template name already exists: %s", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}

	// 检查是否存在同名但已软删除的记录，如果存在则硬删除
	if err := s.db.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).Delete(&model.TaintTemplate{}).Error; err != nil {
		s.logger.Warningf("Failed to clean up soft-deleted template with name %s: %v", req.Name, err)
		// 不返回错误，继续创建
	}

	// 序列化污点
	taintsJSON, err := json.Marshal(req.Taints)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize taints: %w", err)
	}

	template := model.TaintTemplate{
		Name:        req.Name,
		Description: req.Description,
		Taints:      string(taintsJSON),
		CreatedBy:   userID,
	}

	if err := s.db.Create(&template).Error; err != nil {
		s.logger.Errorf("Failed to create taint template %s: %v", req.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceTaintTemplate,
			Details:      fmt.Sprintf("Failed to create taint template %s", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	// 获取完整的模板信息
	templateInfo, err := s.getTemplateInfo(&template)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("Successfully created taint template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionCreate,
		ResourceType: model.ResourceTaintTemplate,
		Details:      fmt.Sprintf("Created taint template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return templateInfo, nil
}

// UpdateTemplate 更新污点模板
func (s *Service) UpdateTemplate(id uint, req TemplateUpdateRequest, userID uint) (*TemplateInfo, error) {
	var template model.TaintTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != "" && req.Name != template.Name {
		// 检查新名称是否已存在
		var existingTemplate model.TaintTemplate
		if err := s.db.Where("name = ? AND id != ?", req.Name, id).First(&existingTemplate).Error; err == nil {
			return nil, fmt.Errorf("template name already exists: %s", req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to check template name: %w", err)
		}
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.Taints != nil {
		if err := s.validateTaints(req.Taints, "add"); err != nil {
			return nil, fmt.Errorf("invalid taints: %w", err)
		}

		taintsJSON, err := json.Marshal(req.Taints)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize taints: %w", err)
		}
		updates["taints"] = string(taintsJSON)
	}

	if len(updates) > 0 {
		if err := s.db.Model(&template).Updates(updates).Error; err != nil {
			s.logger.Errorf("Failed to update taint template %s: %v", template.Name, err)
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceTaintTemplate,
				Details:      fmt.Sprintf("Failed to update taint template %s", template.Name),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     err.Error(),
			})
			return nil, fmt.Errorf("failed to update template: %w", err)
		}

		// 重新获取更新后的模板
		if err := s.db.First(&template, id).Error; err != nil {
			return nil, fmt.Errorf("failed to get updated template: %w", err)
		}
	}

	templateInfo, err := s.getTemplateInfo(&template)
	if err != nil {
		return nil, err
	}

	s.logger.Infof("Successfully updated taint template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaintTemplate,
		Details:      fmt.Sprintf("Updated taint template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return templateInfo, nil
}

// DeleteTemplate 删除污点模板
func (s *Service) DeleteTemplate(id uint, userID uint) error {
	var template model.TaintTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	if err := s.db.Delete(&template).Error; err != nil {
		s.logger.Errorf("Failed to delete taint template %s: %v", template.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionDelete,
			ResourceType: model.ResourceTaintTemplate,
			Details:      fmt.Sprintf("Failed to delete taint template %s", template.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Infof("Successfully deleted taint template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionDelete,
		ResourceType: model.ResourceTaintTemplate,
		Details:      fmt.Sprintf("Deleted taint template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// ListTemplates 获取污点模板列表
func (s *Service) ListTemplates(req TemplateListRequest, userID uint) (*TemplateListResponse, error) {
	query := s.db.Model(&model.TaintTemplate{}).Preload("Creator")

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}

	// 搜索功能：搜索名称和描述
	if req.Search != "" {
		searchTerm := "%" + req.Search + "%"
		query = query.Where("name LIKE ? OR description LIKE ?", searchTerm, searchTerm)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count templates: %w", err)
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	var templates []model.TaintTemplate
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&templates).Error; err != nil {
		return nil, fmt.Errorf("failed to list templates: %w", err)
	}

	var templateInfos []TemplateInfo
	for _, template := range templates {
		info, err := s.getTemplateInfo(&template)
		if err != nil {
			s.logger.Errorf("Failed to parse template %s: %v", template.Name, err)
			continue
		}

		// 按效果筛选
		if req.Effect != "" {
			hasEffect := false
			for _, taint := range info.Taints {
				if taint.Effect == req.Effect {
					hasEffect = true
					break
				}
			}
			if !hasEffect {
				continue
			}
		}

		// 搜索污点Key（如果有搜索词）
		if req.Search != "" {
			searchTerm := strings.ToLower(req.Search)
			// 如果已经通过名称或描述匹配，则直接添加
			nameMatch := strings.Contains(strings.ToLower(template.Name), searchTerm)
			descMatch := strings.Contains(strings.ToLower(template.Description), searchTerm)

			if !nameMatch && !descMatch {
				// 检查是否匹配污点Key
				keyMatch := false
				for _, taint := range info.Taints {
					if strings.Contains(strings.ToLower(taint.Key), searchTerm) {
						keyMatch = true
						break
					}
				}
				if !keyMatch {
					continue
				}
			}
		}

		templateInfos = append(templateInfos, *info)
	}

	return &TemplateListResponse{
		Templates: templateInfos,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// ApplyTemplate 应用污点模板到节点
func (s *Service) ApplyTemplate(req ApplyTemplateRequest, userID uint) error {
	// 获取模板
	var template model.TaintTemplate
	if err := s.db.First(&template, req.TemplateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 使用用户提供的污点值，如果没有提供则使用模板污点
	var taints []k8s.TaintInfo
	if req.Taints != nil && len(req.Taints) > 0 {
		// 使用前端发送的用户选择的污点值（已经清理过MULTI_VALUE分隔符）
		taints = req.Taints
		s.logger.Infof("Using user-selected taints: %+v", taints)
	} else {
		// 回退到模板的原始污点
		if err := json.Unmarshal([]byte(template.Taints), &taints); err != nil {
			return fmt.Errorf("failed to parse template taints: %w", err)
		}
		s.logger.Infof("Using template taints: %+v", taints)

		// 清理模板污点中的多值分隔符（防止直接应用模板时出错）
		for i := range taints {
			if strings.Contains(taints[i].Value, "|MULTI_VALUE|") {
				// 如果包含多值分隔符，取第一个值
				values := strings.Split(taints[i].Value, "|MULTI_VALUE|")
				cleanValues := make([]string, 0)
				for _, value := range values {
					if trimmed := strings.TrimSpace(value); trimmed != "" {
						cleanValues = append(cleanValues, trimmed)
					}
				}
				if len(cleanValues) > 0 {
					taints[i].Value = cleanValues[0]
					s.logger.Infof("Cleaned multi-value taint: %s = %s (from %d values)", taints[i].Key, taints[i].Value, len(cleanValues))
				} else {
					taints[i].Value = ""
				}
			}
		}
	}

	// 应用到所有指定节点
	operation := req.Operation
	if operation == "" {
		operation = TaintOperationAdd
	}

	batchReq := BatchUpdateRequest{
		ClusterName: req.ClusterName,
		NodeNames:   req.NodeNames,
		Taints:      taints,
		Operation:   operation,
	}

	if err := s.BatchUpdateTaints(batchReq, userID); err != nil {
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to apply template %s to nodes", template.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to apply template: %w", err)
	}

	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Applied template %s to %d nodes in cluster %s", template.Name, len(req.NodeNames), req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// GetTaintUsage 获取集群中污点使用情况
func (s *Service) GetTaintUsage(clusterName string, userID uint) ([]TaintUsage, error) {
	// 强制刷新缓存，确保获取最新的污点信息
	nodes, err := s.k8sSvc.ListNodesWithCache(clusterName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 统计污点使用情况
	taintMap := make(map[string]map[string][]string) // key -> value -> nodes
	effectMap := make(map[string][]string)           // key -> effects

	for _, node := range nodes {
		for _, taint := range node.Taints {
			if taintMap[taint.Key] == nil {
				taintMap[taint.Key] = make(map[string][]string)
			}
			taintMap[taint.Key][taint.Value] = append(taintMap[taint.Key][taint.Value], node.Name)

			// 记录效果
			found := false
			for _, effect := range effectMap[taint.Key] {
				if effect == taint.Effect {
					found = true
					break
				}
			}
			if !found {
				effectMap[taint.Key] = append(effectMap[taint.Key], taint.Effect)
			}
		}
	}

	var usages []TaintUsage
	for key, values := range taintMap {
		var allValues []string
		var allNodes []string
		nodeSet := make(map[string]bool)

		for value, nodes := range values {
			allValues = append(allValues, value)
			for _, node := range nodes {
				if !nodeSet[node] {
					nodeSet[node] = true
					allNodes = append(allNodes, node)
				}
			}
		}

		usages = append(usages, TaintUsage{
			Key:       key,
			Values:    allValues,
			Effects:   effectMap[key],
			NodeCount: len(allNodes),
			Nodes:     allNodes,
		})
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionView,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Viewed taint usage for cluster %s", clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return usages, nil
}

// validateTaints 验证污点信息
func (s *Service) validateTaints(taints []k8s.TaintInfo, operation string) error {
	validEffects := map[string]bool{
		TaintEffectNoSchedule:       true,
		TaintEffectPreferNoSchedule: true,
		TaintEffectNoExecute:        true,
	}

	// 用于检查相同key的污点值
	keyValueMap := make(map[string][]string)

	for i, taint := range taints {
		if taint.Key == "" {
			return fmt.Errorf("taint %d: key cannot be empty", i+1)
		}

		// 对于删除操作，不需要验证Effect字段
		if operation != "remove" {
			if !validEffects[taint.Effect] {
				return fmt.Errorf("taint %d: invalid effect %s, must be one of: %s, %s, %s",
					i+1, taint.Effect, TaintEffectNoSchedule, TaintEffectPreferNoSchedule, TaintEffectNoExecute)
			}
		}

		// 检查键名格式
		if strings.Contains(taint.Key, " ") {
			return fmt.Errorf("taint %d: key cannot contain spaces", i+1)
		}

		// 记录相同key的所有值
		keyValueMap[taint.Key] = append(keyValueMap[taint.Key], taint.Value)
	}

	// 检查同一个key的污点值不能同时包含空值和非空值
	for key, values := range keyValueMap {
		hasEmpty := false
		hasNonEmpty := false

		for _, value := range values {
			if value == "" {
				hasEmpty = true
			} else {
				hasNonEmpty = true
			}
		}

		if hasEmpty && hasNonEmpty {
			return fmt.Errorf("taint key '%s': cannot have both empty and non-empty values simultaneously", key)
		}
	}

	return nil
}

// getTemplateInfo 获取模板信息
func (s *Service) getTemplateInfo(template *model.TaintTemplate) (*TemplateInfo, error) {
	var taints []k8s.TaintInfo
	if err := json.Unmarshal([]byte(template.Taints), &taints); err != nil {
		return nil, fmt.Errorf("failed to parse template taints: %w", err)
	}

	// 加载创建者信息
	var creator model.User
	if err := s.db.First(&creator, template.CreatedBy).Error; err != nil {
		s.logger.Errorf("Failed to load creator for template %s: %v", template.Name, err)
		// 不返回错误，继续处理
	}

	return &TemplateInfo{
		ID:          template.ID,
		Name:        template.Name,
		Description: template.Description,
		Taints:      taints,
		CreatedBy:   template.CreatedBy,
		CreatedAt:   template.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   template.UpdatedAt.Format("2006-01-02 15:04:05"),
		Creator:     creator,
	}, nil
}

// RemoveTaint 移除指定的污点
func (s *Service) RemoveTaint(clusterName, nodeName, taintKey string, userID uint) error {
	// 获取当前节点信息
	node, err := s.k8sSvc.GetNode(clusterName, nodeName)
	if err != nil {
		return fmt.Errorf("failed to get node %s in cluster %s: %w", nodeName, clusterName, err)
	}

	// 过滤掉指定的污点
	var updatedTaints []k8s.TaintInfo
	found := false
	for _, taint := range node.Taints {
		if taint.Key != taintKey {
			updatedTaints = append(updatedTaints, taint)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("taint %s not found on node %s", taintKey, nodeName)
	}

	// 更新节点污点
	updateReq := k8s.TaintUpdateRequest{
		NodeName: nodeName,
		Taints:   updatedTaints,
	}

	if err := s.k8sSvc.UpdateNodeTaints(clusterName, updateReq); err != nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			NodeName:     nodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to remove taint %s from node %s", taintKey, nodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to update node taints: %w", err)
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		NodeName:     nodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Removed taint %s from node %s in cluster %s", taintKey, nodeName, clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// CopyNodeTaints 复制节点污点
// 从源节点复制所有污点到目标节点，完全替代目标节点的现有污点
func (s *Service) CopyNodeTaints(req CopyTaintsRequest, userID uint) error {
	// 获取源节点信息，强制刷新缓存确保获取最新的污点
	sourceNode, err := s.k8sSvc.GetNodeWithCache(req.ClusterName, req.SourceNodeName, true)
	if err != nil {
		s.logger.Errorf("Failed to get source node %s in cluster %s: %v", req.SourceNodeName, req.ClusterName, err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.SourceNodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to copy taints from node %s: source node not found", req.SourceNodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to get source node %s in cluster %s: %w", req.SourceNodeName, req.ClusterName, err)
	}

	// 验证目标节点存在，强制刷新缓存
	_, err = s.k8sSvc.GetNodeWithCache(req.ClusterName, req.TargetNodeName, true)
	if err != nil {
		s.logger.Errorf("Failed to get target node %s in cluster %s: %v", req.TargetNodeName, req.ClusterName, err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.TargetNodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to copy taints to node %s: target node not found", req.TargetNodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to get target node %s in cluster %s: %w", req.TargetNodeName, req.ClusterName, err)
	}

	// 复制源节点的所有污点
	copiedTaints := make([]k8s.TaintInfo, len(sourceNode.Taints))
	copy(copiedTaints, sourceNode.Taints)

	// 设置时间戳
	now := time.Now()
	for i := range copiedTaints {
		copiedTaints[i].TimeAdded = &now
	}

	// 使用 replace 操作完全替代目标节点的污点
	updateReq := UpdateTaintsRequest{
		ClusterName: req.ClusterName,
		NodeName:    req.TargetNodeName,
		Taints:      copiedTaints,
		Operation:   TaintOperationReplace,
	}

	if err := s.UpdateNodeTaints(updateReq, userID); err != nil {
		s.logger.Errorf("Failed to copy taints from node %s to node %s: %v", req.SourceNodeName, req.TargetNodeName, err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.TargetNodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to copy taints from node %s to node %s", req.SourceNodeName, req.TargetNodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to copy taints: %w", err)
	}

	s.logger.Infof("Successfully copied %d taints from node %s to node %s", len(copiedTaints), req.SourceNodeName, req.TargetNodeName)
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		NodeName:     req.TargetNodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Copied %d taints from node %s to node %s in cluster %s", len(copiedTaints), req.SourceNodeName, req.TargetNodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// TaintCopyProcessor 实现 BatchProcessor 接口用于批量复制污点
type TaintCopyProcessor struct {
	svc          *Service
	req          BatchCopyTaintsRequest
	userID       uint
	sourceTaints []k8s.TaintInfo
}

func (p *TaintCopyProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	// 验证目标节点存在，强制刷新缓存
	_, err := p.svc.k8sSvc.GetNodeWithCache(p.req.ClusterName, nodeName, true)
	if err != nil {
		p.svc.logger.Errorf("Failed to get target node %s in cluster %s: %v", nodeName, p.req.ClusterName, err)
		return fmt.Errorf("failed to get target node %s in cluster %s: %w", nodeName, p.req.ClusterName, err)
	}

	// 复制源节点的污点并设置时间戳
	copiedTaints := make([]k8s.TaintInfo, len(p.sourceTaints))
	copy(copiedTaints, p.sourceTaints)

	now := time.Now()
	for i := range copiedTaints {
		copiedTaints[i].TimeAdded = &now
	}

	// 使用 replace 操作完全替代目标节点的污点
	updateReq := UpdateTaintsRequest{
		ClusterName: p.req.ClusterName,
		NodeName:    nodeName,
		Taints:      copiedTaints,
		Operation:   TaintOperationReplace,
	}

	if err := p.svc.UpdateNodeTaints(updateReq, p.userID); err != nil {
		p.svc.logger.Errorf("Failed to copy taints to node %s: %v", nodeName, err)
		return fmt.Errorf("failed to copy taints to node %s: %w", nodeName, err)
	}

	p.svc.logger.Infof("Successfully copied %d taints to node %s", len(copiedTaints), nodeName)
	return nil
}

// BatchCopyTaints 批量复制节点污点（不带进度推送）
func (s *Service) BatchCopyTaints(req BatchCopyTaintsRequest, userID uint) error {
	return s.BatchCopyTaintsWithProgress(req, userID, "")
}

// BatchCopyTaintsWithProgress 批量复制节点污点（带进度推送）
func (s *Service) BatchCopyTaintsWithProgress(req BatchCopyTaintsRequest, userID uint, taskID string) error {
	s.logger.Infof("Starting batch taint copy from node %s to %d target nodes in cluster %s", req.SourceNodeName, len(req.TargetNodeNames), req.ClusterName)

	// 注意：使用 Informer + WebSocket 实时同步后，无需手动清除缓存
	// Informer 会自动检测到节点变化并通过 WebSocket 推送给前端

	// 验证源节点存在并获取污点，强制刷新缓存
	sourceNode, err := s.k8sSvc.GetNodeWithCache(req.ClusterName, req.SourceNodeName, true)
	if err != nil {
		s.logger.Errorf("Failed to get source node %s in cluster %s: %v", req.SourceNodeName, req.ClusterName, err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.SourceNodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceTaint,
			Details:      fmt.Sprintf("Failed to batch copy taints from node %s: source node not found", req.SourceNodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to get source node %s in cluster %s: %w", req.SourceNodeName, req.ClusterName, err)
	}

	// 如果提供了taskID，则使用进度推送
	if taskID != "" && s.progressSvc != nil {
		// 预先复制源节点的污点
		sourceTaints := make([]k8s.TaintInfo, len(sourceNode.Taints))
		copy(sourceTaints, sourceNode.Taints)

		processor := &TaintCopyProcessor{
			svc:          s,
			req:          req,
			userID:       userID,
			sourceTaints: sourceTaints,
		}

		// 使用进度推送的并发处理
		maxConcurrency := 5 // 限制并发数避免过载
		if err := s.progressSvc.ProcessBatchWithProgress(
			context.Background(),
			taskID,
			"batch_copy_taint",
			req.TargetNodeNames,
			userID,
			maxConcurrency,
			processor,
		); err != nil {
			var clusterID *uint
			if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
				clusterID = &cID
			}
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				ClusterID:    clusterID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceTaint,
				Details:      fmt.Sprintf("Batch copy taints from node %s failed for %d target nodes", req.SourceNodeName, len(req.TargetNodeNames)),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     err.Error(),
			})
			return err
		}
	} else {
		// 传统的顺序处理方式（向后兼容）
		var errors []string
		successCount := 0
		for _, targetNodeName := range req.TargetNodeNames {
			copyReq := CopyTaintsRequest{
				ClusterName:    req.ClusterName,
				SourceNodeName: req.SourceNodeName,
				TargetNodeName: targetNodeName,
			}

			if err := s.CopyNodeTaints(copyReq, userID); err != nil {
				errorMsg := fmt.Sprintf("Node %s: %v", targetNodeName, err)
				errors = append(errors, errorMsg)
				s.logger.Errorf("Failed to copy taints to node %s: %v", targetNodeName, err)
			} else {
				successCount++
			}
		}

		if len(errors) > 0 {
			combinedError := strings.Join(errors, "; ")
			var clusterID *uint
			if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
				clusterID = &cID
			}
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				ClusterID:    clusterID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceTaint,
				Details:      fmt.Sprintf("Batch copy taints from node %s failed for %d nodes", req.SourceNodeName, len(errors)),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     combinedError,
			})
			return fmt.Errorf("batch copy failed for some nodes: %s", combinedError)
		}
	}

	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceTaint,
		Details:      fmt.Sprintf("Batch copied taints from node %s to %d target nodes in cluster %s", req.SourceNodeName, len(req.TargetNodeNames), req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}
