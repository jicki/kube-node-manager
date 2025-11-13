package cluster

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Service 集群管理服务
type Service struct {
	db            *gorm.DB
	logger        *logger.Logger
	auditSvc      *audit.Service
	k8sSvc        *k8s.Service
	healthChecker *HealthChecker // 健康检查器（断路器模式）
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
	Priority    *int   `json:"priority"` // 优先级（可选）
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
		db:            db,
		logger:        logger,
		auditSvc:      auditSvc,
		k8sSvc:        k8sSvc,
		healthChecker: NewHealthChecker(), // 初始化健康检查器
	}

	// 异步初始化已存在的集群客户端连接（不阻塞服务启动）
	// 这样即使有集群连接超时，也不会影响 HTTP 服务器启动和健康检查端点
	go func() {
		service.logger.Info("Starting asynchronous cluster initialization...")
		service.initializeExistingClients()
	}()

	// 启动定期同步检查（每5分钟检查一次是否有未加载的集群）
	go service.startPeriodicSyncCheck()

	return service
}

// Create 创建集群
func (s *Service) Create(req CreateRequest, userID uint) (*model.Cluster, error) {
	// 检查用户权限 - 只有管理员可以创建集群
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Errorf("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Role != model.RoleAdmin {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to create cluster %s: insufficient permissions", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "only admin users can create clusters",
		})
		return nil, fmt.Errorf("insufficient permissions: only admin users can create clusters")
	}

	// 验证kubeconfig
	if err := s.k8sSvc.TestConnection(req.KubeConfig); err != nil {
		s.logger.Errorf("Invalid kubeconfig for cluster %s: %v", req.Name, err)
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

	// 检查集群名称是否全局已存在
	var existingCluster model.Cluster
	if err := s.db.Where("name = ?", req.Name).First(&existingCluster).Error; err == nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionCreate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to create cluster %s: name already exists", req.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "cluster name already exists",
		})
		return nil, fmt.Errorf("cluster name already exists: %s", req.Name)
	} else if err != gorm.ErrRecordNotFound {
		s.logger.Errorf("Failed to check cluster name existence: %v", err)
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
		s.logger.Errorf("Failed to create cluster %s: %v", req.Name, err)
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
	// 注意：在多实例部署中，每个实例都需要独立创建 client
	if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
		s.logger.Errorf("Failed to create k8s client for cluster %s: %v", cluster.Name, err)
		// 更新集群状态为错误
		s.db.Model(&cluster).Update("status", model.ClusterStatusError)
		// 返回错误，让前端知道集群创建失败
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	s.logger.Infof("Successfully created cluster: %s (client created, starting background sync)", cluster.Name)
	
	// 异步同步集群信息和广播（避免阻塞响应）
	// 这样即使集群 API 不可达或响应慢，也不会影响用户体验
	go func(c model.Cluster) {
		// 同步集群信息
		if err := s.syncClusterInfo(&c); err != nil {
			s.logger.Warningf("Failed to sync cluster info for %s: %v", c.Name, err)
			// 更新状态为需要同步
			s.db.Model(&c).Update("status", model.ClusterStatusError)
		} else {
			s.logger.Infof("Successfully synced cluster info for: %s", c.Name)
		}

		// 广播集群创建事件到所有实例
		s.BroadcastClusterCreation(c.Name)
	}(cluster)

	// 立即记录审计日志并返回（不等待同步完成）
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

	// 检查用户权限 - 只有管理员可以更新集群
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return nil, fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Role != model.RoleAdmin {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to update cluster %d: insufficient permissions", id),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "only admin users can update clusters",
		})
		return nil, fmt.Errorf("insufficient permissions: only admin users can update clusters")
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
		// 检查新名称是否全局已存在
		var existingCluster model.Cluster
		if err := s.db.Where("name = ? AND id != ?", req.Name, id).First(&existingCluster).Error; err == nil {
			return nil, fmt.Errorf("cluster name already exists: %s", req.Name)
		} else if err != gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("failed to check cluster name: %w", err)
		}
		updates["name"] = req.Name
	}

	if req.Description != "" {
		updates["description"] = req.Description
	}

	if req.Priority != nil {
		updates["priority"] = *req.Priority
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

	// 检查用户权限 - 只有管理员可以删除集群
	var currentUser model.User
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		s.logger.Error("Failed to get current user %d: %v", userID, err)
		return fmt.Errorf("failed to get current user: %w", err)
	}

	if currentUser.Role != model.RoleAdmin {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionDelete,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to delete cluster %d: insufficient permissions", id),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "only admin users can delete clusters",
		})
		return fmt.Errorf("insufficient permissions: only admin users can delete clusters")
	}

	if err := query.First(&cluster, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("cluster not found")
		}
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 在事务中删除集群及相关记录
	err := s.db.Transaction(func(tx *gorm.DB) error {
		// 1. 解除审计日志的集群关联（保留审计记录）
		// 注意：数据库层面已配置 ON DELETE SET NULL，这里是双保险
		if err := tx.Model(&model.AuditLog{}).
			Where("cluster_id = ?", cluster.ID).
			Update("cluster_id", nil).Error; err != nil {
			s.logger.Error("Failed to unlink audit logs for cluster %s: %v", cluster.Name, err)
			return fmt.Errorf("failed to unlink audit logs: %w", err)
		}
		s.logger.Info("Unlinked audit logs for cluster %s", cluster.Name)

		// 2. 删除节点异常记录
		// 注意：数据库层面已配置 ON DELETE CASCADE，这里是双保险
		if err := tx.Where("cluster_id = ?", cluster.ID).
			Delete(&model.NodeAnomaly{}).Error; err != nil {
			s.logger.Error("Failed to delete node anomalies for cluster %s: %v", cluster.Name, err)
			return fmt.Errorf("failed to delete node anomalies: %w", err)
		}
		s.logger.Info("Deleted node anomalies for cluster %s", cluster.Name)

		// 3. 解除 Ansible 清单的集群关联
		// 注意：数据库层面已配置 ON DELETE SET NULL（在 004 迁移中）
		if err := tx.Model(&model.AnsibleInventory{}).
			Where("cluster_id = ?", cluster.ID).
			Update("cluster_id", nil).Error; err != nil {
			s.logger.Error("Failed to unlink ansible inventories for cluster %s: %v", cluster.Name, err)
			return fmt.Errorf("failed to unlink ansible inventories: %w", err)
		}
		s.logger.Info("Unlinked ansible inventories for cluster %s", cluster.Name)

		// 4. 最后删除集群记录（软删除）
		if err := tx.Delete(&cluster).Error; err != nil {
			s.logger.Error("Failed to delete cluster %s: %v", cluster.Name, err)
			return fmt.Errorf("failed to delete cluster: %w", err)
		}

		return nil
	})

	if err != nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			ClusterID:    &cluster.ID,
			Action:       model.ActionDelete,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to delete cluster %s", cluster.Name),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return err
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

	// 所有用户都可以查看集群列表，但只有admin可以管理集群

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
		s.logger.Errorf("Failed to count clusters: %v", err)
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

	// 获取集群列表（按优先级排序）
	clusters := make([]model.Cluster, 0) // 初始化为空数组而不是nil
	if err := query.Order("priority DESC, created_at DESC").Offset(offset).Limit(req.PageSize).Find(&clusters).Error; err != nil {
		s.logger.Errorf("Failed to list clusters: %v", err)
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

	// 只有管理员可以同步集群
	if currentUser.Role != model.RoleAdmin {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionUpdate,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Failed to sync cluster %d: insufficient permissions", id),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     "only admin users can sync clusters",
		})
		return fmt.Errorf("insufficient permissions: only admin users can sync clusters")
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

// CheckClusterStatus 检查集群状态
func (s *Service) CheckClusterStatus(clusterName string, userID uint) error {
	err := s.k8sSvc.CheckClusterConnection(clusterName)
	if err != nil {
		s.auditSvc.Log(audit.LogRequest{
			UserID:       userID,
			Action:       model.ActionView,
			ResourceType: model.ResourceCluster,
			Details:      fmt.Sprintf("Cluster status check failed for %s", clusterName),
			Status:       model.AuditStatusFailed,
			ErrorMsg:     err.Error(),
		})
		return err
	}

	s.auditSvc.Log(audit.LogRequest{
		UserID:       userID,
		Action:       model.ActionView,
		ResourceType: model.ResourceCluster,
		Details:      fmt.Sprintf("Cluster status check successful for %s", clusterName),
		Status:       model.AuditStatusSuccess,
	})
	return nil
}

// syncClusterInfo 同步集群信息
func (s *Service) syncClusterInfo(cluster *model.Cluster) error {
	clusterInfo, err := s.k8sSvc.GetClusterInfo(cluster.Name)
	if err != nil {
		// 检查是否是客户端不存在的错误，如果是则尝试重新创建
		if strings.Contains(err.Error(), "kubernetes client not found") {
			s.logger.Warning("Kubernetes client not found for cluster %s, attempting to recreate", cluster.Name)
			
			// 调试：输出 kubeconfig 长度（不输出完整内容以保护敏感信息）
			s.logger.Info("Kubeconfig length for cluster %s: %d bytes", cluster.Name, len(cluster.KubeConfig))
			
			// 尝试重新创建客户端
			if createErr := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); createErr != nil {
				// 检查是否是凭证相关错误
				errMsg := createErr.Error()
				if strings.Contains(errMsg, "provide credentials") || strings.Contains(errMsg, "Unauthorized") {
					s.logger.Error("Failed to recreate k8s client for cluster %s: credentials error - %v", cluster.Name, createErr)
					s.logger.Warning("Please check if the kubeconfig contains valid client-certificate-data and client-key-data")
				} else {
					s.logger.Error("Failed to recreate k8s client for cluster %s: %v", cluster.Name, createErr)
				}
				// 更新状态为错误
				s.db.Model(cluster).Updates(map[string]interface{}{
					"status":    model.ClusterStatusError,
					"last_sync": time.Now(),
				})
				return fmt.Errorf("failed to recreate client: %w", createErr)
			}
			
			// 重新尝试获取集群信息
			clusterInfo, err = s.k8sSvc.GetClusterInfo(cluster.Name)
			if err != nil {
				s.logger.Error("Failed to get cluster info for %s after recreating client: %v", cluster.Name, err)
				// 更新状态为错误
				s.db.Model(cluster).Updates(map[string]interface{}{
					"status":    model.ClusterStatusError,
					"last_sync": time.Now(),
				})
				return err
			}
			s.logger.Info("Successfully recreated client and synced cluster: %s", cluster.Name)
		} else {
			s.logger.Error("Failed to get cluster info for %s: %v", cluster.Name, err)
			// 更新状态为错误
			s.db.Model(cluster).Updates(map[string]interface{}{
				"status":    model.ClusterStatusError,
				"last_sync": time.Now(),
			})
			return err
		}
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
// 优化：使用并行处理，避免单个集群失败阻塞其他集群同步
// 优化：按优先级排序，优先同步高优先级集群
func (s *Service) SyncAll() error {
	var clusters []model.Cluster
	// 按优先级降序排序
	if err := s.db.Where("status != ?", model.ClusterStatusInactive).
		Order("priority DESC, id ASC").
		Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to get clusters for sync: %v", err)
		return err
	}

	s.logger.Info("Starting parallel sync for %d clusters (priority-based)", len(clusters))

	// 使用并行处理，限制并发数为5
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)
	errChan := make(chan error, len(clusters))

	for _, cluster := range clusters {
		// 检查断路器状态
		if s.healthChecker.ShouldSkip(cluster.Name) {
			health := s.healthChecker.GetHealth(cluster.Name)
			s.logger.Warning("Skipping cluster %s sync (circuit breaker open, failures: %d)",
				cluster.Name, health.FailureCount)
			continue
		}

		wg.Add(1)
		go func(c model.Cluster) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			if err := s.syncClusterInfo(&c); err != nil {
				// 记录失败到健康检查器
				s.healthChecker.RecordFailure(c.Name, err)
				s.logger.Error("Failed to sync cluster %s: %v", c.Name, err)
				errChan <- fmt.Errorf("cluster %s: %w", c.Name, err)
			} else {
				// 记录成功到健康检查器
				s.healthChecker.RecordSuccess(c.Name)
				s.logger.Info("Successfully synced cluster: %s", c.Name)
			}
		}(cluster)
	}

	// 等待所有同步完成
	wg.Wait()
	close(errChan)

	// 收集错误
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		s.logger.Warning("Completed syncing all clusters with %d failures", len(errors))
		return fmt.Errorf("sync completed with %d failures", len(errors))
	}

	s.logger.Info("Completed syncing all clusters successfully")
	return nil
}

// GetClusterHealth 获取集群健康状态
func (s *Service) GetClusterHealth(clusterName string) *ClusterHealth {
	return s.healthChecker.GetHealth(clusterName)
}

// GetAllClustersHealth 获取所有集群的健康状态
func (s *Service) GetAllClustersHealth() map[string]*ClusterHealth {
	return s.healthChecker.GetAllHealth()
}

// ResetClusterCircuitBreaker 重置集群断路器（手动恢复）
func (s *Service) ResetClusterCircuitBreaker(clusterName string) {
	s.healthChecker.ResetCircuitBreaker(clusterName)
	s.logger.Info("Manually reset circuit breaker for cluster: %s", clusterName)
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

	// 移除频繁的节点列表查看审计日志，减少日志噪音
	// 如果需要审计，可以在具体的节点操作中记录

	return nodes, nil
}

// ReloadCluster 重新加载单个集群（用于多实例同步）
func (s *Service) ReloadCluster(clusterName string) error {
	var cluster model.Cluster
	if err := s.db.Where("name = ?", clusterName).First(&cluster).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("cluster not found: %s", clusterName)
		}
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// 检查是否已存在客户端，如果存在则先移除
	s.k8sSvc.RemoveClient(cluster.Name)

	// 创建新客户端
	if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
		s.logger.Error("Failed to reload k8s client for cluster %s: %v", cluster.Name, err)
		return fmt.Errorf("failed to reload kubernetes client: %w", err)
	}

	s.logger.Info("Successfully reloaded cluster: %s", cluster.Name)
	return nil
}

// BroadcastClusterCreation 广播集群创建事件到所有实例（带重试机制）
func (s *Service) BroadcastClusterCreation(clusterName string) {
	// 获取所有实例的地址
	instances := s.getAllInstances()
	if len(instances) == 0 {
		s.logger.Warning("No other instances found for broadcasting cluster creation")
		return
	}

	s.logger.Info("Broadcasting cluster %s creation to %d instances", clusterName, len(instances))

	// 使用并行请求，提高效率
	var wg sync.WaitGroup
	successCount := 0
	failedInstances := make([]string, 0)
	var mu sync.Mutex
	
	for _, instance := range instances {
		wg.Add(1)
		go func(addr string) {
			defer wg.Done()

			// 带重试的广播（最多重试3次）
			success := false
			var lastErr error
			
			for retry := 0; retry < 3; retry++ {
				if retry > 0 {
					// 重试前等待递增延迟（指数退避）
					backoff := time.Duration(retry) * 2 * time.Second
					s.logger.Info("Retrying broadcast to %s (attempt %d/3) after %v", addr, retry+1, backoff)
					time.Sleep(backoff)
				}
				
				url := fmt.Sprintf("http://%s/api/v1/internal/clusters/%s/reload", addr, clusterName)
				
				// 创建带超时的 HTTP 客户端
				client := &http.Client{
					Timeout: 10 * time.Second, // 增加超时时间
				}

				req, err := http.NewRequest("POST", url, nil)
				if err != nil {
					lastErr = err
					s.logger.Warning("Failed to create reload request for %s: %v", addr, err)
					continue
				}

				resp, err := client.Do(req)
				if err != nil {
					lastErr = err
					s.logger.Warning("Failed to broadcast to %s (attempt %d/3): %v", addr, retry+1, err)
					continue
				}
				defer resp.Body.Close()

				if resp.StatusCode == http.StatusOK {
					s.logger.Info("Successfully broadcasted cluster %s to instance %s", clusterName, addr)
					mu.Lock()
					successCount++
					mu.Unlock()
					success = true
					break
				} else {
					lastErr = fmt.Errorf("status code: %d", resp.StatusCode)
					s.logger.Warning("Failed to broadcast to %s (attempt %d/3), status code: %d", addr, retry+1, resp.StatusCode)
				}
			}
			
			if !success {
				s.logger.Error("Failed to broadcast cluster %s to %s after 3 attempts: %v", clusterName, addr, lastErr)
				mu.Lock()
				failedInstances = append(failedInstances, addr)
				mu.Unlock()
			}
		}(instance)
	}

	// 等待所有广播完成
	wg.Wait()
	
	// 输出广播结果摘要
	if len(failedInstances) > 0 {
		s.logger.Error("Broadcast completed for cluster %s: %d succeeded, %d failed. Failed instances: %v", 
			clusterName, successCount, len(failedInstances), failedInstances)
		s.logger.Warning("Some instances may not have the cluster %s loaded. You may need to restart them or manually trigger reload.", clusterName)
	} else {
		s.logger.Info("Successfully broadcasted cluster %s to all %d instances", clusterName, successCount)
	}
}

// getAllInstances 获取所有实例的地址（IP:PORT），排除当前实例
func (s *Service) getAllInstances() []string {
	var allInstances []string
	
	// 获取当前实例的 IP 地址
	currentPodIP := os.Getenv("POD_IP")

	// 方法1: 从环境变量获取（适用于 Kubernetes Headless Service）
	// 格式: POD_IPS=10.10.12.95,10.10.12.96,10.10.12.97,10.10.12.98
	if podIPs := os.Getenv("POD_IPS"); podIPs != "" {
		ips := strings.Split(podIPs, ",")
		port := os.Getenv("POD_PORT")
		if port == "" {
			port = "8080" // 默认端口
		}
		for _, ip := range ips {
			ip = strings.TrimSpace(ip)
			// 排除当前实例自己
			if ip != currentPodIP && ip != "" {
				allInstances = append(allInstances, ip+":"+port)
			}
		}
		s.logger.Info("Found %d other instances from POD_IPS (current: %s, total: %d)", 
			len(allInstances), currentPodIP, len(ips))
		return allInstances
	}

	// 方法2: 从环境变量获取完整地址列表
	// 格式: INSTANCE_ADDRESSES=10.10.12.95:8080,10.10.12.96:8080
	if addrs := os.Getenv("INSTANCE_ADDRESSES"); addrs != "" {
		port := os.Getenv("POD_PORT")
		if port == "" {
			port = "8080"
		}
		currentAddr := currentPodIP + ":" + port
		
		addresses := strings.Split(addrs, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			// 排除当前实例自己
			if addr != currentAddr && addr != "" {
				allInstances = append(allInstances, addr)
			}
		}
		s.logger.Info("Found %d other instances from INSTANCE_ADDRESSES (current: %s, total: %d)", 
			len(allInstances), currentAddr, len(addresses))
		return allInstances
	}

	// 方法3: 使用 Kubernetes Service 进行服务发现
	// 这需要在 Kubernetes 环境中运行，并且需要配置 Headless Service
	serviceName := os.Getenv("SERVICE_NAME")
	namespace := os.Getenv("POD_NAMESPACE")
	if serviceName != "" && namespace != "" {
		allInstances = s.discoverInstancesFromK8s(serviceName, namespace)
		// 从发现的实例中排除当前实例
		if currentPodIP != "" {
			port := os.Getenv("POD_PORT")
			if port == "" {
				port = "8080"
			}
			currentAddr := currentPodIP + ":" + port
			
			var otherInstances []string
			for _, addr := range allInstances {
				if addr != currentAddr {
					otherInstances = append(otherInstances, addr)
				}
			}
			allInstances = otherInstances
		}
		
		if len(allInstances) > 0 {
			s.logger.Info("Found %d other instances from Kubernetes service discovery (current: %s)", 
				len(allInstances), currentPodIP)
			return allInstances
		}
	}

	s.logger.Warning("No instance discovery method configured, cluster reload will not be broadcasted")
	return allInstances
}

// discoverInstancesFromK8s 从 Kubernetes API 发现实例
func (s *Service) discoverInstancesFromK8s(serviceName, namespace string) []string {
	var instances []string

	// 尝试通过 DNS 解析 Headless Service
	// 格式: <service-name>.<namespace>.svc.cluster.local
	fqdn := fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace)
	
	ips, err := net.LookupIP(fqdn)
	if err != nil {
		s.logger.Warning("Failed to lookup service %s: %v", fqdn, err)
		return instances
	}

	port := os.Getenv("POD_PORT")
	if port == "" {
		port = "8080"
	}

	for _, ip := range ips {
		if ip.To4() != nil { // 只使用 IPv4
			instances = append(instances, ip.String()+":"+port)
		}
	}

	return instances
}

// initializeExistingClients 初始化已存在的集群客户端连接
// 优化：使用并行处理，避免单个集群失败阻塞其他集群初始化
// 优化：按优先级排序，优先初始化高优先级集群
func (s *Service) initializeExistingClients() {
	var clusters []model.Cluster
	// 按优先级降序排序（priority DESC），优先级高的先初始化
	if err := s.db.Where("status = ?", model.ClusterStatusActive).
		Order("priority DESC, id ASC").
		Find(&clusters).Error; err != nil {
		s.logger.Error("Failed to load existing clusters: %v", err)
		return
	}

	s.logger.Info("Initializing %d existing cluster connections (parallel mode, priority-based)", len(clusters))

	// 使用并行处理，限制并发数为5，避免同时创建过多连接
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5)

	for _, cluster := range clusters {
		// 检查断路器状态
		if s.healthChecker.ShouldSkip(cluster.Name) {
			health := s.healthChecker.GetHealth(cluster.Name)
			s.logger.Warning("Skipping cluster %s initialization (circuit breaker open, failures: %d, last error: %v)",
				cluster.Name, health.FailureCount, health.LastError)
			continue
		}

		wg.Add(1)
		go func(c model.Cluster) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 输出 kubeconfig 基本信息（不输出完整内容）
			s.logger.Info("Initializing cluster %s (kubeconfig length: %d bytes)", c.Name, len(c.KubeConfig))

			if err := s.k8sSvc.CreateClient(c.Name, c.KubeConfig); err != nil {
				// 记录失败到健康检查器
				s.healthChecker.RecordFailure(c.Name, err)

				errMsg := err.Error()
				if strings.Contains(errMsg, "provide credentials") || strings.Contains(errMsg, "Unauthorized") {
					s.logger.Warning("Failed to initialize client for cluster %s: credentials missing or invalid - %v", c.Name, err)
					s.logger.Warning("Cluster %s may need kubeconfig update with valid credentials", c.Name)
				} else {
					s.logger.Warning("Failed to initialize client for cluster %s: %v", c.Name, err)
				}
				// 更新集群状态为错误
				s.db.Model(&c).Update("status", model.ClusterStatusError)
			} else {
				// 记录成功到健康检查器
				s.healthChecker.RecordSuccess(c.Name)
				s.logger.Info("Successfully initialized client for cluster: %s", c.Name)
			}
		}(cluster)
	}

	// 等待所有初始化完成
	wg.Wait()
	s.logger.Info("Completed initializing all cluster connections")
}

// startPeriodicSyncCheck 启动定期同步检查，确保所有集群都已加载
// 这个机制可以防止由于广播失败导致的集群未同步问题
func (s *Service) startPeriodicSyncCheck() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("Started periodic sync check (every 5 minutes)")

	for range ticker.C {
		s.logger.Info("Running periodic sync check for clusters...")
		
		// 获取数据库中所有active状态的集群
		var dbClusters []model.Cluster
		if err := s.db.Where("status = ?", model.ClusterStatusActive).Find(&dbClusters).Error; err != nil {
			s.logger.Error("Failed to load clusters from database for sync check: %v", err)
			continue
		}

		// 获取当前已加载的集群列表
		loadedClusters := s.k8sSvc.GetLoadedClusters()
		loadedMap := make(map[string]bool)
		for _, name := range loadedClusters {
			loadedMap[name] = true
		}

		// 检查是否有未加载的集群
		unloadedClusters := make([]model.Cluster, 0)
		for _, cluster := range dbClusters {
			if !loadedMap[cluster.Name] {
				unloadedClusters = append(unloadedClusters, cluster)
			}
		}

		if len(unloadedClusters) > 0 {
			s.logger.Warning("Found %d unloaded clusters, attempting to load them...", len(unloadedClusters))
			
			// 尝试加载未同步的集群
			for _, cluster := range unloadedClusters {
				s.logger.Info("Loading unsynced cluster: %s", cluster.Name)
				if err := s.k8sSvc.CreateClient(cluster.Name, cluster.KubeConfig); err != nil {
					s.logger.Error("Failed to load cluster %s during sync check: %v", cluster.Name, err)
					// 更新状态为错误
					s.db.Model(&cluster).Update("status", model.ClusterStatusError)
				} else {
					s.logger.Info("Successfully loaded unsynced cluster: %s", cluster.Name)
				}
			}
		} else {
			s.logger.Info("All clusters are in sync (%d clusters loaded)", len(loadedClusters))
		}
	}
}
