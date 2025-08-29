package cluster

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"time"

	"gorm.io/gorm"
)

// Service 集群管理服务
type Service struct {
	db       *gorm.DB
	logger   *logger.Logger
	auditSvc *audit.Service
	k8sSvc   *k8s.Service
}

// CreateRequest 创建集群请求
type CreateRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config" binding:"required"`
}

// UpdateRequest 更新集群请求
type UpdateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	KubeConfig  string `json:"kube_config"`
}

// ListRequest 集群列表请求
type ListRequest struct {
	Page     int                 `json:"page"`
	PageSize int                 `json:"page_size"`
	Name     string              `json:"name"`
	Status   model.ClusterStatus `json:"status"`
}

// ListResponse 集群列表响应
type ListResponse struct {
	Clusters []model.Cluster `json:"clusters"`
	Total    int64           `json:"total"`
	Page     int             `json:"page"`
	PageSize int             `json:"page_size"`
}

// ClusterWithNodes 带节点信息的集群
type ClusterWithNodes struct {
	*model.Cluster
	Nodes []k8s.NodeInfo `json:"nodes,omitempty"`
}

// NewService 创建新的集群管理服务实例
func NewService(db *gorm.DB, logger *logger.Logger, auditSvc *audit.Service, k8sSvc *k8s.Service) *Service {
	service := &Service{
		db:       db,
		logger:   logger,
		auditSvc: auditSvc,
		k8sSvc:   k8sSvc,
	}

	// 初始化已存在的集群客户端连接
	service.initializeExistingClients()

	return service
}

// Create 创建集群
func (s *Service) Create(req CreateRequest, userID uint) (*model.Cluster, error) {
	// 验证kubeconfig
	if err := s.k8sSvc.TestConnection(req.KubeConfig); err != nil {
		s.logger.Error("Invalid kubeconfig for cluster %s: %v", req.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to create cluster %s: invalid kubeconfig", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("invalid kubeconfig: %w", err)
	}

	// 检查集群名称在当前用户下是否已存在
	var existingCluster model.Cluster
	if err := s.db.Where("name = ? AND created_by = ?", req.Name, userID).First(&existingCluster).Error; err == nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to create cluster %s: name already exists for user", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "cluster name already exists for this user",
		})
		return nil, fmt.Errorf("cluster name already exists: %s", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		s.logger.Error("Failed to check cluster name existence: %v", err)
		return nil, fmt.Errorf("failed to check cluster name: %w", err)
	}

	// 创建集群记录
	cluster := model.Cluster{
		Name:        req.Name,
		Description: req.Description,
		KubeConfig:  req.KubeConfig,
		Status:      model.ClusterStatusActive,
		CreatedBy:   userID,
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		s.logger.Error("Failed to create cluster %s: %v", req.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to create cluster %s", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("failed to create cluster: %w", err)
	}

	// 创建Kubernetes客户端
	if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
		s.logger.Error("Failed to create k8s client for cluster %s: %v", cluster.Name, err)
		// 不返回错误，但记录日志
	}

	// 同步集群信息
	s.syncClusterInfo(&cluster)

	s.logger.Info("Successfully created cluster: %s", cluster.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionCreate,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Created cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return &cluster, nil
}

// GetByID 根据ID获取集群
func (s *Service) GetByID(id uint, userID uint) (*ClusterWithNodes, error) {
	var cluster model.Cluster
	query := s.db.Preload("Creator")

	// 检查用户权限
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能访问自己创建的集群
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found")
		}
		s.logger.Error("Failed to get cluster %d: %v", id, err)
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	result := &ClusterWithNodes{
		Cluster: &cluster,
	}

	// 获取节点信息
	if nodes, err := s.k8sSvc.ListNodes(cluster.Name); err != nil {
		s.logger.Warning("Failed to get nodes for cluster %s: %v", cluster.Name, err)
		// 不返回错误，继续执行
	} else {
		result.Nodes = nodes
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionView,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Viewed cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return result, nil
}

// Update 更新集群
func (s *Service) Update(id uint, req UpdateRequest, userID uint) (*model.Cluster, error) {
	var cluster model.Cluster
	query := s.db

	// 检查用户权限
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能更新自己创建的集群
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found")
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	oldName := cluster.Name
	updates := make(map[string]interface{})

	// 更新字段
	if req.Name != "" && req.Name != cluster.Name {
		// 检查新名称在当前用户下是否已存在
		var existingCluster model.Cluster
		if err := s.db.Where("name = ? AND created_by = ? AND id != ?", req.Name, userID, id).First(&existingCluster).Error; err == nil {
			return nil, fmt.Errorf("cluster name already exists: %s", req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to check cluster name: %w", err)
		}
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.KubeConfig != "" && req.KubeConfig != cluster.KubeConfig {
		// 验证新的kubeconfig
		if err := s.k8sSvc.TestConnection(req.KubeConfig); err != nil {
			s.auditSvc.Log(audit.LogRequest{
				UserID:       userID,
				ClusterID:    &cluster.ID,
				Action:       model.ActionUpdate,
				ResourceType: model.ResourceCluster,
				Details:      fmt.Sprintf("Failed to update cluster %s: invalid kubeconfig", cluster.Name),
				Status:       model.AuditStatusFailed,
				ErrorMsg:     err.Error(),
			})
			return nil, fmt.Errorf("invalid kubeconfig: %w", err)
		}
		updates["kube_config"] = req.KubeConfig
	}

	if len(updates) == 0 {
		return &cluster, nil
	}

	// 更新数据库记录
	if err := s.db.Model(&cluster).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update cluster %s: %v", cluster.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    &cluster.ID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to update cluster %s", cluster.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return nil, fmt.Errorf("failed to update cluster: %w", err)
	}

	// 如果名称或kubeconfig发生变化，需要重新创建客户端
	if req.Name != "" || req.KubeConfig != "" {
		// 移除旧客户端
		s.k8sSvc.RemoveClient(oldName)

		// 重新获取更新后的cluster
		if err := s.db.First(&cluster, id).Error; err != nil {
			return nil, fmt.Errorf("failed to get updated cluster: %w", err)
		}

		// 创建新客户端
		if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
			s.logger.Error("Failed to create k8s client for updated cluster %s: %v", cluster.Name, err)
		}
	}

	// 同步集群信息
	s.syncClusterInfo(&cluster)

	s.logger.Info("Successfully updated cluster: %s", cluster.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Updated cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return &cluster, nil
}

// Delete 删除集群
func (s *Service) Delete(id uint, userID uint) error {
	var cluster model.Cluster
	query := s.db

	// 检查用户权限
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能删除自己创建的集群
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("cluster not found")
		}
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 删除数据库记录
	if err := s.db.Delete(&cluster).Error; err != nil {
		s.logger.Error("Failed to delete cluster %s: %v", cluster.Name, err)
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    &cluster.ID,
			Action:       model.ActionDelete,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to delete cluster %s", cluster.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to delete cluster: %w", err)
	}

	// 移除Kubernetes客户端
	s.k8sSvc.RemoveClient(cluster.Name)

	s.logger.Info("Successfully deleted cluster: %s", cluster.Name)
	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionDelete,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Deleted cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// List 获取集群列表
func (s *Service) List(req ListRequest, userID uint) (*ListResponse, error) {
	query := s.db.Model(&model.Cluster{}).Preload("Creator")

	// 检查用户权限 - 只有管理员可以看到所有集群，其他用户只能看到自己创建的
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能看到自己创建的集群
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	// 应用过滤条件
	if req.Name != "" {
		query = query.Where("name LIKE ?", "%"+req.Name+"%")
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		s.logger.Error("Failed to count clusters: %v", err)
		return nil, fmt.Errorf("failed to count clusters: %w", err)
	}

	// 设置默认分页参数
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 || req.PageSize > 100 {
		req.PageSize = 10
	}

	offset := (req.Page - 1) * req.PageSize

	// 获取集群列表
	var clusters []model.Cluster
	if err := query.Order("created_at DESC").Offset(offset).Limit(req.PageSize).Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to list clusters: %v", err)
		return nil, fmt.Errorf("failed to list clusters: %w", err)
	}

	return &ListResponse{
		Clusters: clusters,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	}, nil
}

// Sync 同步集群信息
func (s *Service) Sync(id uint, userID uint) error {
	var cluster model.Cluster
	query := s.db

	// 检查用户权限
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能同步自己创建的集群
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("cluster not found")
		}
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 同步集群信息
	if err := s.syncClusterInfo(&cluster); err != nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    &cluster.ID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to sync cluster %s", cluster.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return fmt.Errorf("failed to sync cluster: %w", err)
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionUpdate,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Synced cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return nil
}

// TestConnection 测试集群连接
func (s *Service) TestConnection(kubeconfig string) error {
	return s.k8sSvc.TestConnection(kubeconfig)
}

// syncClusterInfo 同步集群信息
func (s *Service) syncClusterInfo(cluster *model.Cluster) error {
	clusterInfo, err := s.k8sSvc.GetClusterInfo(cluster.Name)
	if err != nil {
		s.logger.Error("Failed to get cluster info for %s: %v", cluster.Name, err)
		// 更新状态为错误
		s.db.Model(cluster).Updates(map[string]interface{}{
			"status":    model.ClusterStatusError,
			"last_sync": time.Now(),
		})
		return err
	}

	// 更新集群信息
	now := time.Now()
	updates := map[string]interface{}{
		"version":    clusterInfo.Version,
		"node_count": clusterInfo.NodeCount,
		"status":     model.ClusterStatusActive,
		"last_sync":  now,
	}

	if err := s.db.Model(cluster).Updates(updates).Error; err != nil {
		s.logger.Error("Failed to update cluster info for %s: %v", cluster.Name, err)
		return err
	}

	s.logger.Info("Successfully synced cluster info for: %s", cluster.Name)
	return nil
}

// SyncAll 同步所有集群信息
func (s *Service) SyncAll() error {
	var clusters []model.Cluster
	if err := s.db.Where("status != ?", model.ClusterStatusInactive).Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to get clusters for sync: %v", err)
		return err
	}

	for _, cluster := range clusters {
		if err := s.syncClusterInfo(&cluster); err != nil {
			s.logger.Error("Failed to sync cluster %s: %v", cluster.Name, err)
			continue
		}
	}

	s.logger.Info("Completed syncing all clusters")
	return nil
}

// GetNodes 获取集群节点
func (s *Service) GetNodes(id uint, userID uint) ([]k8s.NodeInfo, error) {
	var cluster model.Cluster
	query := s.db

	// 检查用户权限
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	// 如果不是管理员，只能查看自己创建的集群节点
	if currentUser.Role != model.RoleAdmin {
		query = query.Where("created_by = ?", userID)
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("cluster not found")
		}
		return nil, fmt.Errorf("failed to get cluster: %w", err)
	}

	nodes, err := s.k8sSvc.ListNodes(cluster.Name)
	if err != nil {
		s.logger.Error("Failed to get nodes for cluster %s: %v", cluster.Name, err)
		return nil, fmt.Errorf("failed to get nodes: %w", err)
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		ClusterID:    &cluster.ID,
		Action:       model.ActionView,
		ResourceType: model.ResourceNode,
		Details:      fmt.Sprintf("Viewed nodes for cluster %s", cluster.Name),
		Status:       model.AuditStatusSuccess,
	})

	return nodes, nil
}

// initializeExistingClients 初始化已存在的集群客户端连接
func (s *Service) initializeExistingClients() {
	var clusters []model.Cluster
	if err := s.db.Where("status = ?", model.ClusterStatusActive).Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to load existing clusters: %v", err)
		return
	}

	s.logger.Info("Initializing %d existing cluster connections", len(clusters))

	for _, cluster := range clusters {
		if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
			s.logger.Warning("Failed to initialize client for cluster %s: %v", cluster.Name, err)
			// 更新集群状态为不可用
			s.db.Model(&cluster).Update("status", model.ClusterStatusInactive)
		} else {
			s.logger.Info("Successfully initialized client for cluster: %s", cluster.Name)
		}
	}
}
