package anomaly

import (
	"context"
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/cache"
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
	cache      cache.Cache
	cacheTTL   *CacheTTL
	cleanupSvc *CleanupService
	interval   time.Duration
	enabled    bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// CacheTTL 缓存TTL配置
type CacheTTL struct {
	Statistics time.Duration
	Active     time.Duration
	Clusters   time.Duration
	TypeStats  time.Duration
}

// NewService 创建异常监控服务实例
func NewService(db *gorm.DB, logger *logger.Logger, k8sSvc *k8s.Service, clusterSvc *cluster.Service, cache cache.Cache, cacheTTL *CacheTTL, cleanupSvc *CleanupService, enabled bool, intervalSeconds int) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	return &Service{
		db:         db,
		logger:     logger,
		k8sSvc:     k8sSvc,
		clusterSvc: clusterSvc,
		cache:      cache,
		cacheTTL:   cacheTTL,
		cleanupSvc: cleanupSvc,
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

	// 启动清理服务
	if s.cleanupSvc != nil {
		s.cleanupSvc.Start()
	}

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

	// 停止清理服务
	if s.cleanupSvc != nil {
		s.cleanupSvc.Stop()
	}

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

	// 清除相关缓存
	s.invalidateCache(cluster.ID)

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

	// 清除相关缓存
	s.invalidateCache(anomaly.ClusterID)

	return nil
}

// invalidateCache 使缓存失效
func (s *Service) invalidateCache(clusterID uint) {
	ctx := context.Background()
	// 清除指定集群的缓存
	patterns := []string{
		fmt.Sprintf("anomaly:statistics:%d:*", clusterID),
		fmt.Sprintf("anomaly:active:%d", clusterID),
		fmt.Sprintf("anomaly:type_stats:%d:*", clusterID),
		"anomaly:statistics:all:*", // 全局统计缓存
		"anomaly:active:all",       // 全局活跃异常缓存
	}

	for _, pattern := range patterns {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warningf("Failed to clear cache pattern %s: %v", pattern, err)
		}
	}
}

// buildCacheKey 构建缓存键
func (s *Service) buildCacheKey(prefix string, params ...interface{}) string {
	key := prefix
	for _, p := range params {
		key += fmt.Sprintf(":%v", p)
	}
	return key
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

	// 构建缓存键
	clusterIDStr := "all"
	if req.ClusterID != nil {
		clusterIDStr = fmt.Sprintf("%d", *req.ClusterID)
	}
	cacheKey := s.buildCacheKey("anomaly:statistics",
		clusterIDStr,
		req.Dimension,
		req.StartTime.Format("2006-01-02"),
		req.EndTime.Format("2006-01-02"),
		req.AnomalyType,
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var statistics []model.AnomalyStatistics
		if err := json.Unmarshal(cached, &statistics); err == nil {
			// Cache hit
			return statistics, nil
		}
	}

	// 缓存未命中，查询数据库

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

	// 写入缓存
	if data, err := json.Marshal(statistics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache statistics: %v", err)
		}
	}

	return statistics, nil
}

// GetActiveAnomalies 获取当前活跃的异常
func (s *Service) GetActiveAnomalies(clusterID *uint) ([]model.NodeAnomaly, error) {
	// 构建缓存键
	clusterIDStr := "all"
	if clusterID != nil {
		clusterIDStr = fmt.Sprintf("%d", *clusterID)
	}
	cacheKey := s.buildCacheKey("anomaly:active", clusterIDStr)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var anomalies []model.NodeAnomaly
		if err := json.Unmarshal(cached, &anomalies); err == nil {
			// Cache hit
			return anomalies, nil
		}
	}

	// 缓存未命中，查询数据库

	query := s.db.Model(&model.NodeAnomaly{}).
		Where("status = ?", model.AnomalyStatusActive)

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}

	var anomalies []model.NodeAnomaly
	if err := query.Order("start_time DESC").Find(&anomalies).Error; err != nil {
		return nil, fmt.Errorf("failed to get active anomalies: %w", err)
	}

	// 写入缓存
	if data, err := json.Marshal(anomalies); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Active); err != nil {
			s.logger.Warningf("Failed to cache active anomalies: %v", err)
		}
	}

	return anomalies, nil
}

// GetAnomalySummary 获取异常统计摘要（包括所有状态）
func (s *Service) GetAnomalySummary(clusterID *uint) (map[string]interface{}, error) {
	// 构建缓存键
	clusterIDStr := "all"
	if clusterID != nil {
		clusterIDStr = fmt.Sprintf("%d", *clusterID)
	}
	cacheKey := s.buildCacheKey("anomaly:summary", clusterIDStr)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var summary map[string]interface{}
		if err := json.Unmarshal(cached, &summary); err == nil {
			// Cache hit
			return summary, nil
		}
	}

	// 缓存未命中，查询数据库
	// 查询最近的异常（包括所有状态）
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)

	query := s.db.Model(&model.NodeAnomaly{}).
		Where("start_time >= ?", thirtyDaysAgo)

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}

	var anomalies []model.NodeAnomaly
	if err := query.Order("start_time DESC").Find(&anomalies).Error; err != nil {
		return nil, fmt.Errorf("failed to get anomalies: %w", err)
	}

	// 计算统计摘要
	totalCount := int64(len(anomalies))
	activeCount := int64(0)
	resolvedCount := int64(0)
	affectedNodesMap := make(map[string]bool)

	for _, anomaly := range anomalies {
		if anomaly.Status == model.AnomalyStatusActive {
			activeCount++
		} else if anomaly.Status == model.AnomalyStatusResolved {
			resolvedCount++
		}
		affectedNodesMap[anomaly.NodeName] = true
	}

	summary := map[string]interface{}{
		"total_count":    totalCount,
		"active_count":   activeCount,
		"resolved_count": resolvedCount,
		"affected_nodes": int64(len(affectedNodesMap)),
	}

	// 写入缓存
	if data, err := json.Marshal(summary); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Active); err != nil {
			s.logger.Warningf("Failed to cache anomaly summary: %v", err)
		}
	}

	return summary, nil
}

// TriggerCheck 手动触发检测
func (s *Service) TriggerCheck() error {
	s.logger.Info("Manual anomaly check triggered")
	s.checkAllClusters()

	// 清除所有缓存
	ctx := context.Background()
	patterns := []string{
		"anomaly:statistics:*",
		"anomaly:active:*",
		"anomaly:summary:*",
		"anomaly:type_stats:*",
	}
	for _, pattern := range patterns {
		if err := s.cache.Clear(ctx, pattern); err != nil {
			s.logger.Warningf("Failed to clear cache pattern %s: %v", pattern, err)
		}
	}

	return nil
}

// GetCleanupService 获取清理服务
func (s *Service) GetCleanupService() *CleanupService {
	return s.cleanupSvc
}

// GetTypeStatistics 获取异常类型统计
func (s *Service) GetTypeStatistics(clusterID *uint, startTime, endTime *time.Time) ([]model.AnomalyTypeStatistics, error) {
	// 构建缓存键
	clusterIDStr := "all"
	if clusterID != nil {
		clusterIDStr = fmt.Sprintf("%d", *clusterID)
	}
	startTimeStr := "all"
	if startTime != nil {
		startTimeStr = startTime.Format("2006-01-02")
	}
	endTimeStr := "all"
	if endTime != nil {
		endTimeStr = endTime.Format("2006-01-02")
	}
	cacheKey := s.buildCacheKey("anomaly:type_stats", clusterIDStr, startTimeStr, endTimeStr)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var statistics []model.AnomalyTypeStatistics
		if err := json.Unmarshal(cached, &statistics); err == nil {
			// Cache hit
			return statistics, nil
		}
	}

	// 缓存未命中，查询数据库

	query := s.db.Model(&model.NodeAnomaly{}).
		Select("anomaly_type, COUNT(*) as total_count")

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
	if err := query.Group("anomaly_type").Order("total_count DESC").Find(&statistics).Error; err != nil {
		return nil, fmt.Errorf("failed to get type statistics: %w", err)
	}

	// 写入缓存
	if data, err := json.Marshal(statistics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.TypeStats); err != nil {
			s.logger.Warningf("Failed to cache type statistics: %v", err)
		}
	}

	return statistics, nil
}

// GetRoleStatistics 获取按节点角色聚合统计
func (s *Service) GetRoleStatistics(clusterID *uint, startTime, endTime *time.Time) ([]model.NodeRoleStatistics, error) {
	// 设置默认时间范围（最近30天）
	if startTime == nil {
		t := time.Now().AddDate(0, 0, -30)
		startTime = &t
	}
	if endTime == nil {
		t := time.Now()
		endTime = &t
	}

	// 构建缓存键
	clusterIDStr := "all"
	if clusterID != nil {
		clusterIDStr = fmt.Sprintf("%d", *clusterID)
	}
	cacheKey := s.buildCacheKey("anomaly:role_stats",
		clusterIDStr,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var statistics []model.NodeRoleStatistics
		if err := json.Unmarshal(cached, &statistics); err == nil {
			return statistics, nil
		}
	}

	// 获取所有异常记录（包含集群信息）
	query := s.db.Model(&model.NodeAnomaly{}).
		Select("node_name, cluster_id, cluster_name, COUNT(*) as count, SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active, SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved, AVG(CASE WHEN status = 'Resolved' THEN duration ELSE 0 END) as avg_duration").
		Where("start_time >= ? AND start_time <= ?", startTime, endTime)

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}

	var anomalyData []struct {
		NodeName    string
		ClusterID   uint
		ClusterName string
		Count       int64
		Active      int64
		Resolved    int64
		AvgDuration float64
	}

	if err := query.Group("node_name, cluster_id, cluster_name").Find(&anomalyData).Error; err != nil {
		return nil, fmt.Errorf("failed to query anomaly data: %w", err)
	}

	// 获取所有相关集群的节点信息，以便提取角色
	clusterMap := make(map[uint]model.Cluster)
	var clusters []model.Cluster
	clusterQuery := s.db.Where("status = ?", model.ClusterStatusActive)
	if clusterID != nil {
		clusterQuery = clusterQuery.Where("id = ?", *clusterID)
	}
	if err := clusterQuery.Find(&clusters).Error; err != nil {
		return nil, fmt.Errorf("failed to query clusters: %w", err)
	}
	for _, c := range clusters {
		clusterMap[c.ID] = c
	}

	// 构建节点角色映射
	nodeRoleMap := make(map[uint]map[string]string) // clusterID -> nodeName -> role
	for cID, c := range clusterMap {
		nodeRoleMap[cID] = make(map[string]string)
		nodes, err := s.k8sSvc.ListNodes(c.Name)
		if err != nil {
			s.logger.Warningf("Failed to list nodes for cluster %s: %v", c.Name, err)
			continue
		}
		for _, node := range nodes {
			// 获取主要角色（取第一个）
			if len(node.Roles) > 0 {
				nodeRoleMap[cID][node.Name] = node.Roles[0]
			} else {
				nodeRoleMap[cID][node.Name] = "worker"
			}
		}
	}

	// 按角色聚合统计
	roleStatsMap := make(map[string]*model.NodeRoleStatistics)
	nodesByRole := make(map[string]map[string]bool) // role -> set of node names

	for _, data := range anomalyData {
		role := nodeRoleMap[data.ClusterID][data.NodeName]
		if role == "" {
			role = "worker" // 默认角色
		}

		key := role
		if clusterID != nil {
			key = fmt.Sprintf("%s-%d", role, data.ClusterID)
		}

		if _, exists := roleStatsMap[key]; !exists {
			roleStatsMap[key] = &model.NodeRoleStatistics{
				Role:        role,
				ClusterName: data.ClusterName,
			}
			if clusterID != nil {
				roleStatsMap[key].ClusterID = clusterID
			}
			nodesByRole[key] = make(map[string]bool)
		}

		stats := roleStatsMap[key]
		stats.TotalCount += data.Count
		stats.ActiveCount += data.Active
		stats.ResolvedCount += data.Resolved
		stats.AverageDuration += data.AvgDuration * float64(data.Resolved)
		nodesByRole[key][data.NodeName] = true
	}

	// 计算平均持续时长和受影响节点数
	var result []model.NodeRoleStatistics
	for key, stats := range roleStatsMap {
		if stats.ResolvedCount > 0 {
			stats.AverageDuration = stats.AverageDuration / float64(stats.ResolvedCount)
		}
		stats.AffectedNodes = int64(len(nodesByRole[key]))

		// 计算该角色的节点总数
		roleNodeCount := int64(0)
		for cID := range nodeRoleMap {
			if clusterID == nil || *clusterID == cID {
				for _, nodeRole := range nodeRoleMap[cID] {
					if nodeRole == stats.Role {
						roleNodeCount++
					}
				}
			}
		}
		stats.NodeCount = roleNodeCount

		result = append(result, *stats)
	}

	// 写入缓存
	if data, err := json.Marshal(result); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache role statistics: %v", err)
		}
	}

	return result, nil
}

// GetClusterAggregateStatistics 获取按集群聚合统计
func (s *Service) GetClusterAggregateStatistics(startTime, endTime *time.Time) ([]model.ClusterAggregateStatistics, error) {
	// 设置默认时间范围（最近30天）
	if startTime == nil {
		t := time.Now().AddDate(0, 0, -30)
		startTime = &t
	}
	if endTime == nil {
		t := time.Now()
		endTime = &t
	}

	// 构建缓存键
	cacheKey := s.buildCacheKey("anomaly:cluster_aggregate",
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var statistics []model.ClusterAggregateStatistics
		if err := json.Unmarshal(cached, &statistics); err == nil {
			return statistics, nil
		}
	}

	// 查询数据库
	var statistics []model.ClusterAggregateStatistics
	query := `
		SELECT 
			cluster_id,
			cluster_name,
			COUNT(*) as total_count,
			SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active_count,
			SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved_count,
			AVG(CASE WHEN status = 'Resolved' THEN duration ELSE 0 END) as average_duration,
			COUNT(DISTINCT node_name) as affected_nodes
		FROM node_anomalies
		WHERE start_time >= ? AND start_time <= ?
		GROUP BY cluster_id, cluster_name
		ORDER BY total_count DESC
	`

	if err := s.db.Raw(query, startTime, endTime).Scan(&statistics).Error; err != nil {
		return nil, fmt.Errorf("failed to get cluster aggregate statistics: %w", err)
	}

	// 获取每个集群的节点总数并计算健康度评分
	for i := range statistics {
		// 获取集群信息
		var cluster model.Cluster
		if err := s.db.First(&cluster, statistics[i].ClusterID).Error; err == nil {
			nodes, err := s.k8sSvc.ListNodes(cluster.Name)
			if err == nil {
				statistics[i].TotalNodes = int64(len(nodes))
			}
		}

		// 计算健康度评分
		// 评分逻辑：100分 - (异常率 × 50) - (活跃异常率 × 30)
		if statistics[i].TotalNodes > 0 {
			anomalyRate := float64(statistics[i].AffectedNodes) / float64(statistics[i].TotalNodes)
			activeRate := float64(statistics[i].ActiveCount) / float64(statistics[i].TotalCount)
			score := 100.0 - (anomalyRate * 50) - (activeRate * 30)
			if score < 0 {
				score = 0
			}
			statistics[i].HealthScore = score
		} else {
			statistics[i].HealthScore = 100.0
		}
	}

	// 写入缓存
	if data, err := json.Marshal(statistics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache cluster aggregate statistics: %v", err)
		}
	}

	return statistics, nil
}
