package audit

import (
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

type Service struct {
	db     *gorm.DB
	logger *logger.Logger
}

type LogRequest struct {
	UserID       uint               `json:"user_id"`
	ClusterID    *uint              `json:"cluster_id,omitempty"`
	NodeName     string             `json:"node_name,omitempty"`
	Action       model.AuditAction  `json:"action"`
	ResourceType model.ResourceType `json:"resource_type"`
	Details      string             `json:"details"`
	Reason       string             `json:"reason,omitempty"` // 操作原因，特别用于记录禁止调度的原因
	Status       model.AuditStatus  `json:"status"`
	ErrorMsg     string             `json:"error_msg,omitempty"`
	IPAddress    string             `json:"ip_address,omitempty"`
	UserAgent    string             `json:"user_agent,omitempty"`
}

type ListRequest struct {
	Page         int                `json:"page"`
	PageSize     int                `json:"page_size"`
	UserID       uint               `json:"user_id"`
	Username     string             `json:"username"`
	ClusterID    uint               `json:"cluster_id"`
	Action       model.AuditAction  `json:"action"`
	ResourceType model.ResourceType `json:"resource_type"`
	Status       model.AuditStatus  `json:"status"`
	StartDate    string             `json:"start_date"`
	EndDate      string             `json:"end_date"`
}

type ListResponse struct {
	Logs     []model.AuditLog `json:"logs"`
	Total    int64            `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}

func NewService(db *gorm.DB, logger *logger.Logger) *Service {
	return &Service{
		db:     db,
		logger: logger,
	}
}

func (s *Service) Log(req LogRequest) {
	var clusterID uint
	if req.ClusterID != nil {
		clusterID = *req.ClusterID
	}

	auditLog := model.AuditLog{
		UserID:       req.UserID,
		ClusterID:    clusterID,
		NodeName:     req.NodeName,
		Action:       req.Action,
		ResourceType: req.ResourceType,
		Details:      req.Details,
		Reason:       req.Reason,
		Status:       req.Status,
		ErrorMsg:     req.ErrorMsg,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
	}

	if err := s.db.Create(&auditLog).Error; err != nil {
		s.logger.Errorf("Failed to create audit log: %v", err)
	}
}

func (s *Service) List(req ListRequest) (*ListResponse, error) {
	query := s.db.Model(&model.AuditLog{}).Preload("User").Preload("Cluster")

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.Username != "" {
		query = query.Joins("LEFT JOIN users ON audit_logs.user_id = users.id").Where("users.username LIKE ?", "%"+req.Username+"%")
	}
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.Action != "" {
		s.logger.Infof("添加action过滤条件: %s", req.Action)
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		s.logger.Infof("添加resource_type过滤条件: %s", req.ResourceType)
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.Status != "" {
		s.logger.Infof("添加status过滤条件: %s", req.Status)
		query = query.Where("status = ?", req.Status)
	}
	if req.StartDate != "" {
		query = query.Where("created_at >= ?", req.StartDate)
	}
	if req.EndDate != "" {
		query = query.Where("created_at <= ?", req.EndDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	var logs []model.AuditLog
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	return &ListResponse{
		Logs:     logs,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

func (s *Service) GetByID(id uint) (*model.AuditLog, error) {
	var log model.AuditLog
	if err := s.db.Preload("User").Preload("Cluster").First(&log, id).Error; err != nil {
		return nil, err
	}
	return &log, nil
}

// GetLatestCordonRecord 获取节点最新的禁止调度记录
func (s *Service) GetLatestCordonRecord(nodeName string, clusterID uint) (*model.AuditLog, error) {
	var log model.AuditLog
	query := s.db.Model(&model.AuditLog{}).
		Preload("User").
		Where("node_name = ?", nodeName).
		Where("resource_type = ?", model.ResourceNode).
		Where("action = ?", model.ActionUpdate).
		Where("status = ?", model.AuditStatusSuccess)

	// 如果指定了集群ID，添加集群过滤条件
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	// 只查询包含"Cordoned"或"Uncordoned"关键词的记录，按时间倒序获取最新一条
	err := query.Where("details LIKE ? OR details LIKE ?", "%Cordoned%", "%Uncordoned%").
		Order("created_at DESC").
		First(&log).Error

	if err != nil {
		return nil, err
	}

	return &log, nil
}

// GetLatestCordonRecords 批量获取多个节点的最新禁止调度记录
func (s *Service) GetLatestCordonRecords(nodeNames []string, clusterID uint) (map[string]*model.AuditLog, error) {
	if len(nodeNames) == 0 {
		return make(map[string]*model.AuditLog), nil
	}

	var logs []model.AuditLog
	query := s.db.Model(&model.AuditLog{}).
		Preload("User").
		Where("node_name IN ?", nodeNames).
		Where("resource_type = ?", model.ResourceNode).
		Where("action = ?", model.ActionUpdate).
		Where("status = ?", model.AuditStatusSuccess)

	// 如果指定了集群ID，添加集群过滤条件
	if clusterID > 0 {
		query = query.Where("cluster_id = ?", clusterID)
	}

	// 查询包含禁止调度或解除调度关键词的记录
	err := query.Where("details LIKE ? OR details LIKE ?", "%Cordoned%", "%Uncordoned%").
		Order("node_name, created_at DESC").
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	// 为每个节点保留最新的一条记录
	result := make(map[string]*model.AuditLog)
	for _, log := range logs {
		if _, exists := result[log.NodeName]; !exists {
			logCopy := log
			result[log.NodeName] = &logCopy
		}
	}

	return result, nil
}
