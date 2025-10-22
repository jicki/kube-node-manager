package anomaly

import (
	"context"
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/pkg/logger"
	"sync"
	"time"

	"gorm.io/gorm"
)

// Service 异常监控服务
type Service struct {
	db         *gorm.DB
	logger     *logger.Logger
	k8sSvc     *k8s.Service
	clusterSvc *cluster.Service
	interval   time.Duration
	enabled    bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// NewService 创建异常监控服务实例
func NewService(db *gorm.DB, logger *logger.Logger, k8sSvc *k8s.Service, clusterSvc *cluster.Service, enabled bool, intervalSeconds int) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:         db,
		logger:     logger,
		k8sSvc:     k8sSvc,
		clusterSvc: clusterSvc,
		interval:   time.Duration(intervalSeconds) * time.Second,
		enabled:    enabled,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// StartMonitoring 启动后台监控协程
func (s *Service) StartMonitoring() {
	if !s.enabled {
		s.logger.Info("Node anomaly monitoring is disabled")
		return
	}

	s.logger.Infof("Starting node anomaly monitoring with interval: %v", s.interval)

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		// 立即执行一次检查
		s.checkAllClusters()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.checkAllClusters()
			case <-s.ctx.Done():
				s.logger.Info("Node anomaly monitoring stopped")
				return
			}
		}
	}()
}

// StopMonitoring 停止监控服务
func (s *Service) StopMonitoring() {
	if !s.enabled {
		return
	}

	s.logger.Info("Stopping node anomaly monitoring...")
	s.cancel()
	s.wg.Wait()
	s.logger.Info("Node anomaly monitoring stopped successfully")
}

// checkAllClusters 检查所有集群的节点
func (s *Service) checkAllClusters() {
	// 直接从数据库查询所有活跃集群（避免依赖 userID）
	var clusters []model.Cluster
	if err := s.db.Where("status = ?", model.ClusterStatusActive).Find(&clusters).Error; err != nil {
		s.logger.Errorf("Failed to list clusters for anomaly monitoring: %v", err)
		return
	}

	if len(clusters) == 0 {
		s.logger.Infof("No clusters found for anomaly monitoring")
		return
	}

	var wg sync.WaitGroup
	for _, cls := range clusters {
		wg.Add(1)
		go func(c model.Cluster) {
			defer wg.Done()
			if err := s.checkClusterNodes(c); err != nil {
				s.logger.Errorf("Failed to check cluster %s: %v", c.Name, err)
			}
		}(cls)
	}

	wg.Wait()
}

// checkClusterNodes 检查单个集群的所有节点
func (s *Service) checkClusterNodes(cluster model.Cluster) error {
	nodes, err := s.k8sSvc.ListNodes(cluster.Name)
	if err != nil {
		return fmt.Errorf("failed to list nodes: %w", err)
	}

	// 获取该集群所有活跃的异常记录
	var activeAnomalies []model.NodeAnomaly
	if err := s.db.Where("cluster_id = ? AND status = ?", cluster.ID, model.AnomalyStatusActive).Find(&activeAnomalies).Error; err != nil {
		s.logger.Errorf("Failed to get active anomalies for cluster %s: %v", cluster.Name, err)
	}

	// 创建一个 map 便于快速查找
	activeAnomalyMap := make(map[string]map[model.AnomalyType]*model.NodeAnomaly)
	for i := range activeAnomalies {
		if _, exists := activeAnomalyMap[activeAnomalies[i].NodeName]; !exists {
			activeAnomalyMap[activeAnomalies[i].NodeName] = make(map[model.AnomalyType]*model.NodeAnomaly)
		}
		activeAnomalyMap[activeAnomalies[i].NodeName][activeAnomalies[i].AnomalyType] = &activeAnomalies[i]
	}

	// 当前检测到的异常
	currentAnomalies := make(map[string]map[model.AnomalyType]bool)

	// 检测每个节点的异常
	for _, node := range nodes {
		anomalies := s.detectAnomalies(node)
		if len(anomalies) > 0 {
			currentAnomalies[node.Name] = make(map[model.AnomalyType]bool)
			for _, anomaly := range anomalies {
				currentAnomalies[node.Name][anomaly.AnomalyType] = true
				// 记录或更新异常
				if err := s.recordAnomaly(cluster, node.Name, anomaly, activeAnomalyMap); err != nil {
					s.logger.Errorf("Failed to record anomaly for node %s: %v", node.Name, err)
				}
			}
		}
	}

	// 检查之前活跃的异常是否已恢复
	for nodeName, anomalyMap := range activeAnomalyMap {
		for anomalyType, anomaly := range anomalyMap {
			// 如果当前检测中没有这个异常，说明已经恢复
			if currentAnomalies[nodeName] == nil || !currentAnomalies[nodeName][anomalyType] {
				if err := s.resolveAnomaly(anomaly); err != nil {
					s.logger.Errorf("Failed to resolve anomaly for node %s: %v", nodeName, err)
				}
			}
		}
	}

	return nil
}

// detectAnomalies 检测节点异常条件
func (s *Service) detectAnomalies(node k8s.NodeInfo) []model.NodeAnomaly {
	var anomalies []model.NodeAnomaly
	now := time.Now()

	for _, condition := range node.Conditions {
		var anomalyType model.AnomalyType
		var isAbnormal bool

		switch condition.Type {
		case "Ready":
			if condition.Status != "True" {
				anomalyType = model.AnomalyTypeNotReady
				isAbnormal = true
			}
		case "MemoryPressure":
			if condition.Status == "True" {
				anomalyType = model.AnomalyTypeMemoryPressure
				isAbnormal = true
			}
		case "DiskPressure":
			if condition.Status == "True" {
				anomalyType = model.AnomalyTypeDiskPressure
				isAbnormal = true
			}
		case "PIDPressure":
			if condition.Status == "True" {
				anomalyType = model.AnomalyTypePIDPressure
				isAbnormal = true
			}
		case "NetworkUnavailable":
			if condition.Status == "True" {
				anomalyType = model.AnomalyTypeNetworkUnavailable
				isAbnormal = true
			}
		}

		if isAbnormal {
			anomalies = append(anomalies, model.NodeAnomaly{
				AnomalyType: anomalyType,
				Reason:      condition.Reason,
				Message:     condition.Message,
				StartTime:   now,
				LastCheck:   now,
			})
		}
	}

	return anomalies
}

// recordAnomaly 记录新的异常或更新现有异常状态
func (s *Service) recordAnomaly(cluster model.Cluster, nodeName string, anomaly model.NodeAnomaly, activeAnomalyMap map[string]map[model.AnomalyType]*model.NodeAnomaly) error {
	// 检查是否已有活跃的异常记录
	if activeAnomalyMap[nodeName] != nil && activeAnomalyMap[nodeName][anomaly.AnomalyType] != nil {
		// 更新最后检查时间
		existing := activeAnomalyMap[nodeName][anomaly.AnomalyType]
		existing.LastCheck = time.Now()
		existing.Reason = anomaly.Reason
		existing.Message = anomaly.Message
		return s.db.Save(existing).Error
	}

	// 创建新的异常记录
	newAnomaly := model.NodeAnomaly{
		ClusterID:   cluster.ID,
		ClusterName: cluster.Name,
		NodeName:    nodeName,
		AnomalyType: anomaly.AnomalyType,
		Status:      model.AnomalyStatusActive,
		StartTime:   anomaly.StartTime,
		LastCheck:   anomaly.LastCheck,
		Reason:      anomaly.Reason,
		Message:     anomaly.Message,
		Duration:    0,
	}

	if err := s.db.Create(&newAnomaly).Error; err != nil {
		return fmt.Errorf("failed to create anomaly record: %w", err)
	}

	s.logger.Infof("New anomaly detected: cluster=%s, node=%s, type=%s", cluster.Name, nodeName, anomaly.AnomalyType)
	return nil
}

// resolveAnomaly 标记异常为已恢复
func (s *Service) resolveAnomaly(anomaly *model.NodeAnomaly) error {
	now := time.Now()
	anomaly.Status = model.AnomalyStatusResolved
	anomaly.EndTime = &now
	anomaly.Duration = int64(now.Sub(anomaly.StartTime).Seconds())

	if err := s.db.Save(anomaly).Error; err != nil {
		return fmt.Errorf("failed to resolve anomaly: %w", err)
	}

	s.logger.Infof("Anomaly resolved: cluster=%s, node=%s, type=%s, duration=%ds",
		anomaly.ClusterName, anomaly.NodeName, anomaly.AnomalyType, anomaly.Duration)
	return nil
}

// ListRequest 异常记录查询请求
type ListRequest struct {
	ClusterID   *uint               `json:"cluster_id"`
	NodeName    string              `json:"node_name"`
	AnomalyType model.AnomalyType   `json:"anomaly_type"`
	Status      model.AnomalyStatus `json:"status"`
	StartTime   *time.Time          `json:"start_time"`
	EndTime     *time.Time          `json:"end_time"`
	Page        int                 `json:"page"`
	PageSize    int                 `json:"page_size"`
}

// ListResponse 异常记录查询响应
type ListResponse struct {
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
	Items      []model.NodeAnomaly `json:"items"`
}

// GetAnomalies 获取异常记录列表
func (s *Service) GetAnomalies(req ListRequest) (*ListResponse, error) {
	query := s.db.Model(&model.NodeAnomaly{}).Preload("Cluster")

	// 应用过滤条件
	if req.ClusterID != nil {
		query = query.Where("cluster_id = ?", *req.ClusterID)
	}
	if req.NodeName != "" {
		query = query.Where("node_name = ?", req.NodeName)
	}
	if req.AnomalyType != "" {
		query = query.Where("anomaly_type = ?", req.AnomalyType)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.StartTime != nil {
		query = query.Where("start_time >= ?", req.StartTime)
	}
	if req.EndTime != nil {
		query = query.Where("start_time <= ?", req.EndTime)
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("failed to count anomalies: %w", err)
	}

	// 设置默认分页参数
	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 {
		req.PageSize = 20
	}

	// 分页查询
	var anomalies []model.NodeAnomaly
	offset := (req.Page - 1) * req.PageSize
	if err := query.Order("start_time DESC").
		Limit(req.PageSize).
		Offset(offset).
		Find(&anomalies).Error; err != nil {
		return nil, fmt.Errorf("failed to query anomalies: %w", err)
	}

	// 计算总页数
	totalPages := int(total) / req.PageSize
	if int(total)%req.PageSize > 0 {
		totalPages++
	}

	return &ListResponse{
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Items:      anomalies,
	}, nil
}

// StatisticsRequest 统计查询请求
type StatisticsRequest struct {
	ClusterID   *uint             `json:"cluster_id"`
	AnomalyType model.AnomalyType `json:"anomaly_type"`
	StartTime   *time.Time        `json:"start_time"`
	EndTime     *time.Time        `json:"end_time"`
	Dimension   string            `json:"dimension"` // "day" or "week"
}

// GetStatistics 获取统计数据
func (s *Service) GetStatistics(req StatisticsRequest) ([]model.AnomalyStatistics, error) {
	// 设置默认时间范围（最近30天）
	if req.StartTime == nil {
		t := time.Now().AddDate(0, 0, -30)
		req.StartTime = &t
	}
	if req.EndTime == nil {
		t := time.Now()
		req.EndTime = &t
	}

	// 默认按天统计
	if req.Dimension == "" {
		req.Dimension = "day"
	}

	// 构建基础查询
	baseQuery := s.db.Model(&model.NodeAnomaly{}).
		Where("start_time >= ? AND start_time <= ?", req.StartTime, req.EndTime)

	if req.ClusterID != nil {
		baseQuery = baseQuery.Where("cluster_id = ?", *req.ClusterID)
	}
	if req.AnomalyType != "" {
		baseQuery = baseQuery.Where("anomaly_type = ?", req.AnomalyType)
	}

	var statistics []model.AnomalyStatistics

	// 根据维度进行分组统计
	var dateFormat string
	if req.Dimension == "week" {
		// SQLite 和 PostgreSQL 的周统计语法不同
		dateFormat = "strftime('%Y-%W', start_time)"
		if s.db.Dialector.Name() == "postgres" {
			dateFormat = "TO_CHAR(start_time, 'IYYY-IW')"
		}
	} else {
		// 按天统计
		dateFormat = "DATE(start_time)"
		if s.db.Dialector.Name() == "postgres" {
			dateFormat = "DATE(start_time)"
		}
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as date,
			COUNT(*) as total_count,
			SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active_count,
			SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved_count,
			AVG(CASE WHEN status = 'Resolved' THEN duration ELSE 0 END) as average_duration,
			COUNT(DISTINCT node_name) as affected_nodes
		FROM node_anomalies
		WHERE start_time >= ? AND start_time <= ?
	`, dateFormat)

	args := []interface{}{req.StartTime, req.EndTime}

	if req.ClusterID != nil {
		query += " AND cluster_id = ?"
		args = append(args, *req.ClusterID)
	}
	if req.AnomalyType != "" {
		query += " AND anomaly_type = ?"
		args = append(args, req.AnomalyType)
	}

	query += " GROUP BY date ORDER BY date"

	if err := s.db.Raw(query, args...).Scan(&statistics).Error; err != nil {
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}

	return statistics, nil
}

// GetActiveAnomalies 获取当前活跃的异常
func (s *Service) GetActiveAnomalies(clusterID *uint) ([]model.NodeAnomaly, error) {
	query := s.db.Model(&model.NodeAnomaly{}).
		Preload("Cluster").
		Where("status = ?", model.AnomalyStatusActive)

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}

	var anomalies []model.NodeAnomaly
	if err := query.Order("start_time DESC").Find(&anomalies).Error; err != nil {
		return nil, fmt.Errorf("failed to get active anomalies: %w", err)
	}

	return anomalies, nil
}

// TriggerCheck 手动触发检测
func (s *Service) TriggerCheck() error {
	s.logger.Info("Manual anomaly check triggered")
	s.checkAllClusters()
	return nil
}

// GetTypeStatistics 获取异常类型统计
func (s *Service) GetTypeStatistics(clusterID *uint, startTime, endTime *time.Time) ([]model.AnomalyTypeStatistics, error) {
	query := s.db.Model(&model.NodeAnomaly{}).
		Select("anomaly_type, COUNT(*) as count")

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}
	if startTime != nil {
		query = query.Where("start_time >= ?", startTime)
	}
	if endTime != nil {
		query = query.Where("start_time <= ?", endTime)
	}

	var statistics []model.AnomalyTypeStatistics
	if err := query.Group("anomaly_type").Order("count DESC").Find(&statistics).Error; err != nil {
		return nil, fmt.Errorf("failed to get type statistics: %w", err)
	}

	return statistics, nil
}
