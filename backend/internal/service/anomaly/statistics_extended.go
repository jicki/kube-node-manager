package anomaly

import (
	"context"
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/model"
	"time"
)

// GetNodeHistoryTrend 获取单个节点的历史异常趋势
func (s *Service) GetNodeHistoryTrend(clusterID uint, nodeName string, startTime, endTime *time.Time) ([]model.NodeHistoryTrend, error) {
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
	cacheKey := s.buildCacheKey("anomaly:node_trend",
		fmt.Sprintf("%d", clusterID),
		nodeName,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var trend []model.NodeHistoryTrend
		if err := json.Unmarshal(cached, &trend); err == nil {
			return trend, nil
		}
	}

	// 查询数据库
	var trend []model.NodeHistoryTrend
	dateFormat := "DATE(start_time)"
	if s.db.Dialector.Name() == "postgres" {
		dateFormat = "DATE(start_time)"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as date,
			COUNT(*) as total_count,
			SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active_count,
			SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved_count,
			AVG(CASE WHEN status = 'Resolved' THEN duration ELSE NULL END) as avg_duration
		FROM node_anomalies
		WHERE cluster_id = ? AND node_name = ? AND start_time >= ? AND start_time <= ?
		GROUP BY date
		ORDER BY date
	`, dateFormat)

	if err := s.db.Raw(query, clusterID, nodeName, startTime, endTime).Scan(&trend).Error; err != nil {
		return nil, fmt.Errorf("failed to get node history trend: %w", err)
	}

	// 写入缓存
	if data, err := json.Marshal(trend); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache node history trend: %v", err)
		}
	}

	return trend, nil
}

// GetMTTRStatistics 计算平均恢复时间统计
func (s *Service) GetMTTRStatistics(entityType string, clusterID *uint, startTime, endTime *time.Time) ([]model.MTTRStatistics, error) {
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
	cacheKey := s.buildCacheKey("anomaly:mttr",
		entityType,
		clusterIDStr,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var statistics []model.MTTRStatistics
		if err := json.Unmarshal(cached, &statistics); err == nil {
			return statistics, nil
		}
	}

	var statistics []model.MTTRStatistics

	// 根据实体类型构建查询
	switch entityType {
	case "node":
		query := `
			SELECT 
				'node' as entity_type,
				node_name as entity_name,
				AVG(duration) as mttr,
				COUNT(*) as resolved_count,
				SUM(duration) as total_duration,
				MIN(duration) as min_duration,
				MAX(duration) as max_duration
			FROM node_anomalies
			WHERE status = 'Resolved' AND start_time >= ? AND start_time <= ?
		`
		args := []interface{}{startTime, endTime}
		if clusterID != nil {
			query += " AND cluster_id = ?"
			args = append(args, *clusterID)
		}
		query += " GROUP BY node_name ORDER BY mttr DESC"

		if err := s.db.Raw(query, args...).Scan(&statistics).Error; err != nil {
			return nil, fmt.Errorf("failed to get node MTTR statistics: %w", err)
		}

	case "cluster":
		query := `
			SELECT 
				'cluster' as entity_type,
				cluster_name as entity_name,
				AVG(duration) as mttr,
				COUNT(*) as resolved_count,
				SUM(duration) as total_duration,
				MIN(duration) as min_duration,
				MAX(duration) as max_duration
			FROM node_anomalies
			WHERE status = 'Resolved' AND start_time >= ? AND start_time <= ?
			GROUP BY cluster_name
			ORDER BY mttr DESC
		`

		if err := s.db.Raw(query, startTime, endTime).Scan(&statistics).Error; err != nil {
			return nil, fmt.Errorf("failed to get cluster MTTR statistics: %w", err)
		}

	default:
		return nil, fmt.Errorf("unsupported entity type: %s", entityType)
	}

	// 写入缓存
	if data, err := json.Marshal(statistics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache MTTR statistics: %v", err)
		}
	}

	return statistics, nil
}

// GetSLAMetrics 计算 SLA 可用性指标
func (s *Service) GetSLAMetrics(entityType string, entityName string, clusterID *uint, startTime, endTime *time.Time) (*model.SLAMetrics, error) {
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
	cacheKey := s.buildCacheKey("anomaly:sla",
		entityType,
		entityName,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var metrics model.SLAMetrics
		if err := json.Unmarshal(cached, &metrics); err == nil {
			return &metrics, nil
		}
	}

	// 计算总时间（秒）
	totalTime := int64(endTime.Sub(*startTime).Seconds())

	// 查询异常时长
	query := s.db.Model(&model.NodeAnomaly{}).
		Select("SUM(duration) as downtime, COUNT(*) as count").
		Where("start_time >= ? AND start_time <= ?", startTime, endTime)

	if entityType == "node" {
		query = query.Where("node_name = ?", entityName)
		if clusterID != nil {
			query = query.Where("cluster_id = ?", *clusterID)
		}
	} else if entityType == "cluster" {
		if clusterID != nil {
			query = query.Where("cluster_id = ?", *clusterID)
		}
	}

	var result struct {
		Downtime int64
		Count    int64
	}

	if err := query.Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate SLA metrics: %w", err)
	}

	// 计算可用性
	uptimeDuration := totalTime - result.Downtime
	if uptimeDuration < 0 {
		uptimeDuration = 0
	}

	availability := 100.0
	if totalTime > 0 {
		availability = float64(uptimeDuration) / float64(totalTime) * 100.0
	}

	metrics := &model.SLAMetrics{
		EntityType:       entityType,
		EntityName:       entityName,
		StartTime:        *startTime,
		EndTime:          *endTime,
		TotalTime:        totalTime,
		DowntimeDuration: result.Downtime,
		UptimeDuration:   uptimeDuration,
		Availability:     availability,
		AnomalyCount:     result.Count,
	}

	// 写入缓存
	if data, err := json.Marshal(metrics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache SLA metrics: %v", err)
		}
	}

	return metrics, nil
}

// GetRecoveryMetrics 计算异常恢复率和复发率
func (s *Service) GetRecoveryMetrics(entityType string, entityName string, clusterID *uint, startTime, endTime *time.Time) (*model.RecoveryMetrics, error) {
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
	cacheKey := s.buildCacheKey("anomaly:recovery",
		entityType,
		entityName,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var metrics model.RecoveryMetrics
		if err := json.Unmarshal(cached, &metrics); err == nil {
			return &metrics, nil
		}
	}

	// 查询总数和已恢复数
	query := s.db.Model(&model.NodeAnomaly{}).
		Select("COUNT(*) as total, SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active, SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved").
		Where("start_time >= ? AND start_time <= ?", startTime, endTime)

	if entityType == "node" {
		query = query.Where("node_name = ?", entityName)
		if clusterID != nil {
			query = query.Where("cluster_id = ?", *clusterID)
		}
	} else if entityType == "cluster" {
		if clusterID != nil {
			query = query.Where("cluster_id = ?", *clusterID)
		}
	}

	var result struct {
		Total    int64
		Active   int64
		Resolved int64
	}

	if err := query.Scan(&result).Error; err != nil {
		return nil, fmt.Errorf("failed to calculate recovery metrics: %w", err)
	}

	// 计算恢复率
	recoveryRate := 0.0
	if result.Total > 0 {
		recoveryRate = float64(result.Resolved) / float64(result.Total) * 100.0
	}

	// 计算复发率（简化版：同一节点同一类型异常出现多次）
	recurringCount := int64(0)
	recurringQuery := s.db.Model(&model.NodeAnomaly{}).
		Select("node_name, anomaly_type, COUNT(*) as count").
		Where("start_time >= ? AND start_time <= ?", startTime, endTime)

	if entityType == "node" {
		recurringQuery = recurringQuery.Where("node_name = ?", entityName)
		if clusterID != nil {
			recurringQuery = recurringQuery.Where("cluster_id = ?", *clusterID)
		}
	} else if entityType == "cluster" {
		if clusterID != nil {
			recurringQuery = recurringQuery.Where("cluster_id = ?", *clusterID)
		}
	}

	var recurring []struct {
		NodeName    string
		AnomalyType string
		Count       int64
	}

	if err := recurringQuery.Group("node_name, anomaly_type").Having("COUNT(*) > 1").Find(&recurring).Error; err == nil {
		for _, r := range recurring {
			recurringCount += r.Count - 1 // 减1，因为第一次不算复发
		}
	}

	recurrenceRate := 0.0
	if result.Resolved > 0 {
		recurrenceRate = float64(recurringCount) / float64(result.Resolved) * 100.0
	}

	metrics := &model.RecoveryMetrics{
		EntityType:     entityType,
		EntityName:     entityName,
		TotalCount:     result.Total,
		ResolvedCount:  result.Resolved,
		ActiveCount:    result.Active,
		RecoveryRate:   recoveryRate,
		RecurringCount: recurringCount,
		RecurrenceRate: recurrenceRate,
	}

	// 写入缓存
	if data, err := json.Marshal(metrics); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache recovery metrics: %v", err)
		}
	}

	return metrics, nil
}
