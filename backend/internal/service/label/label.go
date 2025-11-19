package label

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

// Service 标签管理服务
type Service struct {
	db          *gorm.DB
	logger      *logger.Logger
	auditSvc    *audit.Service
	k8sSvc      *k8s.Service
	progressSvc *progress.Service
}

// UpdateLabelsRequest 更新节点标签请求
type UpdateLabelsRequest struct {
	ClusterName string            `json:"cluster_name" binding:"required"`
	NodeName    string            `json:"node_name" binding:"required"`
	Labels      map[string]string `json:"labels" binding:"required"`
	Operation   string            `json:"operation"` // add, remove, replace
}

// BatchUpdateRequest 批量更新标签请求
type BatchUpdateRequest struct {
	ClusterName string            `json:"cluster_name" binding:"required"`
	NodeNames   []string          `json:"node_names" binding:"required"`
	Labels      map[string]string `json:"labels" binding:"required"`
	Operation   string            `json:"operation"` // add, remove, replace
}

// TemplateCreateRequest 创建标签模板请求
type TemplateCreateRequest struct {
	Name        string                 `json:"name" binding:"required"`
	Description string                 `json:"description"`
	Labels      map[string]interface{} `json:"labels" binding:"required"`
}

// TemplateUpdateRequest 更新标签模板请求
type TemplateUpdateRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Labels      map[string]interface{} `json:"labels"`
}

// TemplateListRequest 模板列表请求
type TemplateListRequest struct {
	Page     int    `json:"page"`
	PageSize int    `json:"page_size"`
	Name     string `json:"name"`
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
	ID          uint              `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Labels      map[string]string `json:"labels"`
	CreatedBy   uint              `json:"created_by"`
	CreatedAt   string            `json:"created_at"`
	UpdatedAt   string            `json:"updated_at"`
	Creator     model.User        `json:"creator"`
}

// ApplyTemplateRequest 应用模板请求
type ApplyTemplateRequest struct {
	ClusterName string            `json:"cluster_name" binding:"required"`
	NodeNames   []string          `json:"node_names" binding:"required"`
	TemplateID  uint              `json:"template_id" binding:"required"`
	Operation   string            `json:"operation"` // add, replace
	Labels      map[string]string `json:"labels"`    // 用户选择的具体标签值
}

// LabelUsage 标签使用情况
type LabelUsage struct {
	Key       string   `json:"key"`
	Values    []string `json:"values"`
	NodeCount int      `json:"node_count"`
	Nodes     []string `json:"nodes,omitempty"`
}

// NewService 创建新的标签管理服务实例
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

// UpdateNodeLabels 更新单个节点标签
func (s *Service) UpdateNodeLabels(req UpdateLabelsRequest, userID uint) error {
	s.logger.Infof("[UpdateNodeLabels] Starting for node %s in cluster %s", req.NodeName, req.ClusterName)
	
	// 获取当前节点信息，强制刷新缓存确保获取最新的标签
	s.logger.Infof("[UpdateNodeLabels] Getting node info for %s", req.NodeName)
	currentNode, err := s.k8sSvc.GetNodeWithCache(req.ClusterName, req.NodeName, true)
	if err != nil {
		s.logger.Errorf("[UpdateNodeLabels] Failed to get node %s: %v", req.NodeName, err)
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
			ResourceType: model.ResourceLabel,
			Details:      fmt.Sprintf("Failed to update labels for node %s in cluster %s: node not found or inaccessible", req.NodeName, req.ClusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to get node %s in cluster %s: %w", req.NodeName, req.ClusterName, err)
	}

	// 准备更新的标签
	updatedLabels := make(map[string]string)

	// 复制现有标签
	if currentNode.Labels != nil {
		for k, v := range currentNode.Labels {
			updatedLabels[k] = v
		}
	}

	switch strings.ToLower(req.Operation) {
	case "add", "":
		// 添加或更新标签
		for k, v := range req.Labels {
			// 验证并清理标签值
			s.logger.Infof("Processing label: key=%s, original_value=%s", k, v)
			cleanedValue := s.sanitizeLabelValue(v)
			s.logger.Infof("Cleaned label: key=%s, cleaned_value=%s", k, cleanedValue)
			if cleanedValue != "" {
				updatedLabels[k] = cleanedValue
			} else {
				s.logger.Warningf("Skipping invalid label value for key %s: original=%s, cleaned=%s", k, v, cleanedValue)
				// 如果清理后的值为空，保留原有标签值（如果存在）
				if existingValue, exists := currentNode.Labels[k]; exists {
					s.logger.Infof("Preserving existing label value for key %s: %s", k, existingValue)
					updatedLabels[k] = existingValue
				}
			}
		}
	case "remove":
		// 删除指定标签
		s.logger.Infof("Starting remove operation for labels: %+v", req.Labels)
		for k := range req.Labels {
			s.logger.Infof("Removing label key: %s", k)
			if _, exists := updatedLabels[k]; exists {
				delete(updatedLabels, k)
				s.logger.Infof("Successfully removed label key: %s", k)
			} else {
				s.logger.Warningf("Label key %s not found on node %s", k, req.NodeName)
			}
		}
		s.logger.Infof("Labels after removal: %+v", updatedLabels)
	case "replace":
		// 替换所有自定义标签（保留系统标签）
		systemLabels := make(map[string]string)
		for k, v := range currentNode.Labels {
			if s.isSystemLabel(k) {
				systemLabels[k] = v
			}
		}
		updatedLabels = systemLabels
		for k, v := range req.Labels {
			// 验证并清理标签值
			s.logger.Infof("Processing replace label: key=%s, original_value=%s", k, v)
			cleanedValue := s.sanitizeLabelValue(v)
			s.logger.Infof("Cleaned replace label: key=%s, cleaned_value=%s", k, cleanedValue)
			if cleanedValue != "" {
				updatedLabels[k] = cleanedValue
			} else {
				s.logger.Warningf("Skipping invalid label value for key %s: original=%s, cleaned=%s", k, v, cleanedValue)
			}
		}
	default:
		return fmt.Errorf("invalid operation: %s", req.Operation)
	}

	// 更新节点标签
	s.logger.Infof("Final labels to apply: %+v", updatedLabels)
	s.logger.Infof("[UpdateNodeLabels] Calling k8sSvc.UpdateNodeLabels for node %s", req.NodeName)
	updateReq := k8s.LabelUpdateRequest{
		NodeName: req.NodeName,
		Labels:   updatedLabels,
	}

	if err := s.k8sSvc.UpdateNodeLabels(req.ClusterName, updateReq); err != nil {
		s.logger.Errorf("[UpdateNodeLabels] k8sSvc.UpdateNodeLabels failed for node %s: %v", req.NodeName, err)
		s.logger.Errorf("Failed to update node labels: %v", err)
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			NodeName:     req.NodeName,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceLabel,
			Details:      fmt.Sprintf("Failed to update labels for node %s", req.NodeName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to update node labels: %w", err)
	}

	// 成功日志已在k8s服务中记录，避免重复
	s.logger.Infof("[UpdateNodeLabels] Successfully updated labels for node %s", req.NodeName)
	
	s.logger.Infof("[UpdateNodeLabels] Logging audit for node %s", req.NodeName)
	var clusterID *uint
	if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		NodeName:     req.NodeName,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceLabel,
		Details:      fmt.Sprintf("Updated labels for node %s in cluster %s", req.NodeName, req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})
	
	s.logger.Infof("[UpdateNodeLabels] Completed for node %s", req.NodeName)

	return nil
}

// LabelProcessor 实现 BatchProcessor 接口
type LabelProcessor struct {
	svc    *Service
	req    BatchUpdateRequest
	userID uint
}

func (p *LabelProcessor) ProcessNode(ctx context.Context, nodeName string, index int) error {
	p.svc.logger.Infof("[ProcessNode] Starting for node %s (index %d)", nodeName, index)
	
	updateReq := UpdateLabelsRequest{
		ClusterName: p.req.ClusterName,
		NodeName:    nodeName,
		Labels:      p.req.Labels,
		Operation:   p.req.Operation,
	}

	p.svc.logger.Infof("[ProcessNode] Calling UpdateNodeLabels for node %s", nodeName)
	err := p.svc.UpdateNodeLabels(updateReq, p.userID)
	
	if err != nil {
		p.svc.logger.Errorf("[ProcessNode] Failed for node %s: %v", nodeName, err)
	} else {
		p.svc.logger.Infof("[ProcessNode] Completed successfully for node %s", nodeName)
	}
	
	return err
}

// BatchUpdateLabels 批量更新节点标签 (带进度推送)
func (s *Service) BatchUpdateLabels(req BatchUpdateRequest, userID uint) error {
	return s.BatchUpdateLabelsWithProgress(req, userID, "")
}

// BatchUpdateLabelsWithProgress 批量更新节点标签 (带进度推送)
func (s *Service) BatchUpdateLabelsWithProgress(req BatchUpdateRequest, userID uint, taskID string) error {
	s.logger.Infof("Starting batch update for %d nodes in cluster %s", len(req.NodeNames), req.ClusterName)

	// 注意：使用 Informer + WebSocket 实时同步后，无需手动清除缓存
	// Informer 会自动检测到节点变化并通过 WebSocket 推送给前端

	// 如果提供了taskID，则使用进度推送
	if taskID != "" && s.progressSvc != nil {
		processor := &LabelProcessor{
			svc:    s,
			req:    req,
			userID: userID,
		}

		// 使用进度推送的并发处理
		maxConcurrency := 5 // 限制并发数避免过载
		if err := s.progressSvc.ProcessBatchWithProgress(
			context.Background(),
			taskID,
			"batch_label",
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
				ResourceType: model.ResourceLabel,
				Details:      fmt.Sprintf("Batch update labels failed for %d nodes", len(req.NodeNames)),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     err.Error(),
			})
			return err
		}
	} else {
		// 传统的顺序处理方式（向后兼容）
		var errors []string
		successCount := 0
		for i, nodeName := range req.NodeNames {
			updateReq := UpdateLabelsRequest{
				ClusterName: req.ClusterName,
				NodeName:    nodeName,
				Labels:      req.Labels,
				Operation:   req.Operation,
			}

			if err := s.UpdateNodeLabels(updateReq, userID); err != nil {
				errorMsg := fmt.Sprintf("Node %s: %v", nodeName, err)
				errors = append(errors, errorMsg)
				s.logger.Errorf("Failed to update labels for node %s: %v", nodeName, err)
			} else {
				successCount++
				s.logger.Infof("Successfully updated labels for node %s (%d/%d)", nodeName, i+1, len(req.NodeNames))
			}

			// 在批量操作之间添加小延迟，避免过度并发
			if i < len(req.NodeNames)-1 {
				time.Sleep(50 * time.Millisecond)
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
				ResourceType: model.ResourceLabel,
				Details:      fmt.Sprintf("Batch update labels failed for %d nodes", len(errors)),
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
		ResourceType: model.ResourceLabel,
		Details:      fmt.Sprintf("Batch updated labels for %d nodes in cluster %s", len(req.NodeNames), req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// CreateTemplate 创建标签模板
func (s *Service) CreateTemplate(req TemplateCreateRequest, userID uint) (*TemplateInfo, error) {
	// 检查模板名称是否已存在（包括未删除的记录）
	var existingTemplate model.LabelTemplate
	if err := s.db.Where("name = ?", req.Name).First(&existingTemplate).Error; err == nil {
		return nil, fmt.Errorf("template name already exists: %s", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		return nil, fmt.Errorf("failed to check template name: %w", err)
	}

	// 检查是否存在同名但已软删除的记录，如果存在则硬删除
	if err := s.db.Unscoped().Where("name = ? AND deleted_at IS NOT NULL", req.Name).Delete(&model.LabelTemplate{}).Error; err != nil {
		s.logger.Warningf("Failed to clean up soft-deleted template with name %s: %v", req.Name, err)
		// 不返回错误，继续创建
	}

	// 将labels转换为map[string]string格式（处理多值情况）
	processedLabels := make(map[string]string)
	for key, value := range req.Labels {
		switch v := value.(type) {
		case string:
			processedLabels[key] = v
		case []interface{}:
			// 将数组转换为逗号分隔的字符串
			var values []string
			for _, item := range v {
				if str, ok := item.(string); ok && str != "" {
					values = append(values, str)
				}
			}
			if len(values) > 0 {
				processedLabels[key] = strings.Join(values, "|MULTI_VALUE|")
			}
		case []string:
			// 直接处理字符串数组
			var values []string
			for _, str := range v {
				if str != "" {
					values = append(values, str)
				}
			}
			if len(values) > 0 {
				processedLabels[key] = strings.Join(values, "|MULTI_VALUE|")
			}
		default:
			// 其他类型转为字符串
			processedLabels[key] = fmt.Sprintf("%v", v)
		}
	}

	// 序列化处理后的标签
	labelsJSON, err := json.Marshal(processedLabels)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize labels: %w", err)
	}

	template := model.LabelTemplate{
		Name:        req.Name,
		Description: req.Description,
		Labels:      string(labelsJSON),
		CreatedBy:   userID,
	}

	if err := s.db.Create(&template).Error; err != nil {
		s.logger.Errorf("Failed to create label template %s: %v", req.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceLabelTemplate,
			Details:      fmt.Sprintf("Failed to create label template %s", req.Name),
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

	s.logger.Infof("Successfully created label template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionCreate,
		ResourceType: model.ResourceLabelTemplate,
		Details:      fmt.Sprintf("Created label template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return templateInfo, nil
}

// UpdateTemplate 更新标签模板
func (s *Service) UpdateTemplate(id uint, req TemplateUpdateRequest, userID uint) (*TemplateInfo, error) {
	var template model.LabelTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("template not found")
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	updates := make(map[string]interface{})

	if req.Name != "" && req.Name != template.Name {
		// 检查新名称是否已存在
		var existingTemplate model.LabelTemplate
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

	if req.Labels != nil {
		// 将labels转换为map[string]string格式（处理多值情况）
		processedLabels := make(map[string]string)
		for key, value := range req.Labels {
			switch v := value.(type) {
			case string:
				processedLabels[key] = v
			case []interface{}:
				// 将数组转换为逗号分隔的字符串
				var values []string
				for _, item := range v {
					if str, ok := item.(string); ok && str != "" {
						values = append(values, str)
					}
				}
				if len(values) > 0 {
					processedLabels[key] = strings.Join(values, "|MULTI_VALUE|")
				}
			case []string:
				// 直接处理字符串数组
				var values []string
				for _, str := range v {
					if str != "" {
						values = append(values, str)
					}
				}
				if len(values) > 0 {
					processedLabels[key] = strings.Join(values, "|MULTI_VALUE|")
				}
			default:
				// 其他类型转为字符串
				processedLabels[key] = fmt.Sprintf("%v", v)
			}
		}

		labelsJSON, err := json.Marshal(processedLabels)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize labels: %w", err)
		}
		updates["labels"] = string(labelsJSON)
	}

	if len(updates) > 0 {
		if err := s.db.Model(&template).Updates(updates).Error; err != nil {
			s.logger.Errorf("Failed to update label template %s: %v", template.Name, err)
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceLabelTemplate,
				Details:      fmt.Sprintf("Failed to update label template %s", template.Name),
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

	s.logger.Infof("Successfully updated label template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceLabelTemplate,
		Details:      fmt.Sprintf("Updated label template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return templateInfo, nil
}

// DeleteTemplate 删除标签模板
func (s *Service) DeleteTemplate(id uint, userID uint) error {
	var template model.LabelTemplate
	if err := s.db.First(&template, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	if err := s.db.Delete(&template).Error; err != nil {
		s.logger.Errorf("Failed to delete label template %s: %v", template.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionDelete,
			ResourceType: model.ResourceLabelTemplate,
			Details:      fmt.Sprintf("Failed to delete label template %s", template.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to delete template: %w", err)
	}

	s.logger.Infof("Successfully deleted label template: %s", template.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionDelete,
		ResourceType: model.ResourceLabelTemplate,
		Details:      fmt.Sprintf("Deleted label template %s", template.Name),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// ListTemplates 获取标签模板列表
func (s *Service) ListTemplates(req TemplateListRequest, userID uint) (*TemplateListResponse, error) {
	query := s.db.Model(&model.LabelTemplate{}).Preload("Creator")

	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
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

	var templates []model.LabelTemplate
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
		templateInfos = append(templateInfos, *info)
	}

	return &TemplateListResponse{
		Templates: templateInfos,
		Total:     total,
		Page:      req.Page,
		PageSize:  req.PageSize,
	}, nil
}

// ApplyTemplate 应用标签模板到节点
func (s *Service) ApplyTemplate(req ApplyTemplateRequest, userID uint) error {
	// 获取模板
	var template model.LabelTemplate
	if err := s.db.First(&template, req.TemplateID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("template not found")
		}
		return fmt.Errorf("failed to get template: %w", err)
	}

	// 使用用户提供的标签值，如果没有提供则使用模板标签
	var labels map[string]string
	if len(req.Labels) > 0 {
		// 使用前端发送的用户选择的标签值
		labels = req.Labels
		s.logger.Infof("Using user-selected labels: %+v", labels)
	} else {
		// 回退到模板的原始标签
		if err := json.Unmarshal([]byte(template.Labels), &labels); err != nil {
			return fmt.Errorf("failed to parse template labels: %w", err)
		}
		s.logger.Infof("Using template labels: %+v", labels)
	}

	// 应用到所有指定节点
	operation := req.Operation
	if operation == "" {
		operation = "add"
	}

	batchReq := BatchUpdateRequest{
		ClusterName: req.ClusterName,
		NodeNames:   req.NodeNames,
		Labels:      labels,
		Operation:   operation,
	}

	if err := s.BatchUpdateLabels(batchReq, userID); err != nil {
		var clusterID *uint
		if cID, err := s.getClusterIDByName(req.ClusterName); err == nil {
			clusterID = &cID
		}
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    clusterID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceLabel,
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
		ResourceType: model.ResourceLabel,
		Details:      fmt.Sprintf("Applied template %s to %d nodes in cluster %s", template.Name, len(req.NodeNames), req.ClusterName),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// GetLabelUsage 获取集群中标签使用情况
func (s *Service) GetLabelUsage(clusterName string, userID uint) ([]LabelUsage, error) {
	// 强制刷新缓存，确保获取最新的标签信息
	nodes, err := s.k8sSvc.ListNodesWithCache(clusterName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	// 统计标签使用情况
	labelMap := make(map[string]map[string][]string) // key -> value -> nodes

	for _, node := range nodes {
		for key, value := range node.Labels {
			// 跳过系统标签
			if s.isSystemLabel(key) {
				continue
			}

			if labelMap[key] == nil {
				labelMap[key] = make(map[string][]string)
			}
			labelMap[key][value] = append(labelMap[key][value], node.Name)
		}
	}

	var usages []LabelUsage
	for key, values := range labelMap {
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

		usages = append(usages, LabelUsage{
			Key:       key,
			Values:    allValues,
			NodeCount: len(allNodes),
			Nodes:     allNodes,
		})
	}

	var clusterID *uint
	if cID, err := s.getClusterIDByName(clusterName); err == nil {
		clusterID = &cID
	}
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    clusterID,
		Action:       model.ActionView,
		ResourceType: model.ResourceLabel,
		Details:      fmt.Sprintf("Viewed label usage for cluster %s", clusterName),
		Status:       model.AuditStatusSuccess,
	})

	return usages, nil
}

// isSystemLabel 检查是否为系统标签
func (s *Service) isSystemLabel(key string) bool {
	systemPrefixes := []string{
		"kubernetes.io/",
		"k8s.io/",
		"node.kubernetes.io/",
		"node-role.kubernetes.io/",
		"beta.kubernetes.io/",
		"failure-domain.beta.kubernetes.io/",
		"topology.kubernetes.io/",
	}

	for _, prefix := range systemPrefixes {
		if strings.HasPrefix(key, prefix) {
			return true
		}
	}

	return false
}

// sanitizeLabelValue 清理标签值，确保符合Kubernetes格式要求
func (s *Service) sanitizeLabelValue(value string) string {
	// 移除|MULTI_VALUE|分隔符，只取第一个值
	if strings.Contains(value, "|MULTI_VALUE|") {
		parts := strings.Split(value, "|MULTI_VALUE|")
		for _, part := range parts {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				value = trimmed
				break
			}
		}
	}

	// Kubernetes标签值的正则表达式验证
	// 必须是空字符串或包含字母数字字符、'-'、'_' 或 '.'，并且必须以字母数字字符开始和结束
	// 最大长度63字符
	if len(value) > 63 {
		value = value[:63]
	}

	// 移除不合法字符，只保留字母数字字符、'-'、'_' 和 '.'
	cleaned := ""
	for _, r := range value {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' {
			cleaned += string(r)
		}
	}

	// 确保以字母数字字符开始和结束
	if cleaned != "" {
		// 移除开头的非字母数字字符
		for len(cleaned) > 0 && !((cleaned[0] >= 'A' && cleaned[0] <= 'Z') || (cleaned[0] >= 'a' && cleaned[0] <= 'z') || (cleaned[0] >= '0' && cleaned[0] <= '9')) {
			cleaned = cleaned[1:]
		}
		// 移除结尾的非字母数字字符
		for len(cleaned) > 0 && !((cleaned[len(cleaned)-1] >= 'A' && cleaned[len(cleaned)-1] <= 'Z') || (cleaned[len(cleaned)-1] >= 'a' && cleaned[len(cleaned)-1] <= 'z') || (cleaned[len(cleaned)-1] >= '0' && cleaned[len(cleaned)-1] <= '9')) {
			cleaned = cleaned[:len(cleaned)-1]
		}
	}

	return cleaned
}

// getTemplateInfo 获取模板信息
func (s *Service) getTemplateInfo(template *model.LabelTemplate) (*TemplateInfo, error) {
	var labels map[string]string
	if err := json.Unmarshal([]byte(template.Labels), &labels); err != nil {
		return nil, fmt.Errorf("failed to parse template labels: %w", err)
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
		Labels:      labels,
		CreatedBy:   template.CreatedBy,
		CreatedAt:   template.CreatedAt.Format("2006-01-02 15:04:05"),
		UpdatedAt:   template.UpdatedAt.Format("2006-01-02 15:04:05"),
		Creator:     creator,
	}, nil
}
