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
	Status       model.AuditStatus  `json:"status"`
	ErrorMsg     string             `json:"error_msg,omitempty"`
	IPAddress    string             `json:"ip_address,omitempty"`
	UserAgent    string             `json:"user_agent,omitempty"`
}

type ListRequest struct {
	Page         int                `json:"page"`
	PageSize     int                `json:"page_size"`
	UserID       uint               `json:"user_id"`
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
		Status:       req.Status,
		ErrorMsg:     req.ErrorMsg,
		IPAddress:    req.IPAddress,
		UserAgent:    req.UserAgent,
	}

	if err := s.db.Create(&auditLog).Error; err != nil {
		s.logger.Error("Failed to create audit log:", err)
	}
}

func (s *Service) List(req ListRequest) (*ListResponse, error) {
	query := s.db.Model(&model.AuditLog{}).Preload("User").Preload("Cluster")

	if req.UserID > 0 {
		query = query.Where("user_id = ?", req.UserID)
	}
	if req.ClusterID > 0 {
		query = query.Where("cluster_id = ?", req.ClusterID)
	}
	if req.Action != "" {
		query = query.Where("action = ?", req.Action)
	}
	if req.ResourceType != "" {
		query = query.Where("resource_type = ?", req.ResourceType)
	}
	if req.Status != "" {
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
