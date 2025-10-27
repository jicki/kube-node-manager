package anomaly

import (
	"context"
	"encoding/json"
	"fmt"
	"kube-node-manager/internal/model"
	"time"
)

// GetNodeHealthScore 计算节点健康度评分
func (s *Service) GetNodeHealthScore(clusterID uint, nodeName string, startTime, endTime *time.Time) (*model.NodeHealthScore, error) {
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
	cacheKey := s.buildCacheKey("anomaly:health_score",
		fmt.Sprintf("%d", clusterID),
		nodeName,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var score model.NodeHealthScore
		if err := json.Unmarshal(cached, &score); err == nil {
			return &score, nil
		}
	}

	// 获取集群名称
	var cluster model.Cluster
	if err := s.db.First(&cluster, clusterID).Error; err != nil {
		return nil, fmt.Errorf("failed to find cluster: %w", err)
	}

	// 查询节点异常统计
	var stats struct {
		TotalCount    int64
		ActiveCount   int64
		ResolvedCount int64
		AvgDuration   float64
	}

	query := `
		SELECT 
			COUNT(*) as total_count,
			SUM(CASE WHEN status = 'Active' THEN 1 ELSE 0 END) as active_count,
			SUM(CASE WHEN status = 'Resolved' THEN 1 ELSE 0 END) as resolved_count,
			AVG(CASE WHEN status = 'Resolved' THEN duration ELSE 0 END) as avg_duration
		FROM node_anomalies
		WHERE cluster_id = ? AND node_name = ? AND start_time >= ? AND start_time <= ?
	`

	if err := s.db.Raw(query, clusterID, nodeName, startTime, endTime).Scan(&stats).Error; err != nil {
		return nil, fmt.Errorf("failed to get node statistics: %w", err)
	}

	// 获取 SLA 指标
	slaMetrics, err := s.GetSLAMetrics("node", nodeName, &clusterID, startTime, endTime)
	if err != nil {
		s.logger.Warningf("Failed to get SLA metrics for node %s: %v", nodeName, err)
		slaMetrics = &model.SLAMetrics{Availability: 100.0}
	}

	// 获取最后一次异常时间
	var lastAnomaly *time.Time
	if err := s.db.Model(&model.NodeAnomaly{}).
		Select("start_time").
		Where("cluster_id = ? AND node_name = ?", clusterID, nodeName).
		Order("start_time DESC").
		Limit(1).
		Scan(&lastAnomaly).Error; err == nil && lastAnomaly != nil {
		// 获取成功
	}

	// 计算健康度评分
	// 评分算法（0-100分）：
	// - 可用性权重 40%
	// - 活跃异常率权重 30%（越低越好）
	// - 平均恢复时间权重 20%（越短越好）
	// - 异常频率权重 10%（越低越好）

	// 1. 可用性得分（40分）
	availabilityScore := slaMetrics.Availability * 0.4

	// 2. 活跃异常率得分（30分）
	activeRate := 0.0
	if stats.TotalCount > 0 {
		activeRate = float64(stats.ActiveCount) / float64(stats.TotalCount)
	}
	activeScore := (1.0 - activeRate) * 30

	// 3. 恢复时间得分（20分）
	// 假设 1小时内恢复为满分，超过24小时为0分
	mttrScore := 20.0
	if stats.AvgDuration > 0 {
		hours := stats.AvgDuration / 3600.0
		if hours >= 24 {
			mttrScore = 0
		} else if hours > 1 {
			mttrScore = 20.0 * (1.0 - (hours-1.0)/23.0)
		}
	}

	// 4. 异常频率得分（10分）
	// 假设30天内0次异常为满分，10次以上为0分
	days := endTime.Sub(*startTime).Hours() / 24.0
	if days == 0 {
		days = 1
	}
	anomalyRate := float64(stats.TotalCount) / days
	frequencyScore := 10.0
	if anomalyRate >= 10.0/30.0 { // 平均每3天一次异常
		frequencyScore = 0
	} else if anomalyRate > 0 {
		frequencyScore = 10.0 * (1.0 - anomalyRate/(10.0/30.0))
	}

	// 综合得分
	healthScore := availabilityScore + activeScore + mttrScore + frequencyScore
	if healthScore < 0 {
		healthScore = 0
	} else if healthScore > 100 {
		healthScore = 100
	}

	score := &model.NodeHealthScore{
		NodeName:        nodeName,
		ClusterID:       clusterID,
		ClusterName:     cluster.Name,
		HealthScore:     healthScore,
		TotalAnomalies:  stats.TotalCount,
		ActiveAnomalies: stats.ActiveCount,
		AvgMTTR:         stats.AvgDuration,
		Availability:    slaMetrics.Availability,
		LastAnomaly:     lastAnomaly,
		Factors: map[string]float64{
			"availability": availabilityScore,
			"active_rate":  activeScore,
			"mttr":         mttrScore,
			"frequency":    frequencyScore,
		},
	}
	score.ScoreLevel = score.GetScoreLevel()

	// 写入缓存
	if data, err := json.Marshal(score); err == nil {
		if err := s.cache.Set(ctx, cacheKey, data, s.cacheTTL.Statistics); err != nil {
			s.logger.Warningf("Failed to cache node health score: %v", err)
		}
	}

	return score, nil
}

// GetHeatmapData 获取热力图数据（时间 × 节点的异常分布）
func (s *Service) GetHeatmapData(clusterID *uint, startTime, endTime *time.Time) ([]model.HeatmapDataPoint, error) {
	// 设置默认时间范围（最近7天，热力图数据量大，不宜过长）
	if startTime == nil {
		t := time.Now().AddDate(0, 0, -7)
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
	cacheKey := s.buildCacheKey("anomaly:heatmap",
		clusterIDStr,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var data []model.HeatmapDataPoint
		if err := json.Unmarshal(cached, &data); err == nil {
			return data, nil
		}
	}

	// 查询数据库
	// 按小时聚合（对于较长时间范围可以按天聚合）
	timeFormat := "strftime('%Y-%m-%d %H:00:00', start_time)"
	if s.db.Dialector.Name() == "postgres" {
		timeFormat = "TO_CHAR(start_time, 'YYYY-MM-DD HH24:00:00')"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as time,
			node_name,
			COUNT(*) as value
		FROM node_anomalies
		WHERE start_time >= ? AND start_time <= ?
	`, timeFormat)

	args := []interface{}{startTime, endTime}
	if clusterID != nil {
		query += " AND cluster_id = ?"
		args = append(args, *clusterID)
	}

	query += " GROUP BY time, node_name ORDER BY time, node_name"

	var data []model.HeatmapDataPoint
	if err := s.db.Raw(query, args...).Scan(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get heatmap data: %w", err)
	}

	// 写入缓存（热力图数据缓存时间可以稍长）
	cacheTTL := 10 * time.Minute
	if dataBytes, err := json.Marshal(data); err == nil {
		if err := s.cache.Set(ctx, cacheKey, dataBytes, cacheTTL); err != nil {
			s.logger.Warningf("Failed to cache heatmap data: %v", err)
		}
	}

	return data, nil
}

// GetCalendarData 获取日历热力图数据（按日期聚合）
func (s *Service) GetCalendarData(clusterID *uint, startTime, endTime *time.Time) ([]model.CalendarDataPoint, error) {
	// 设置默认时间范围（最近90天）
	if startTime == nil {
		t := time.Now().AddDate(0, 0, -90)
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
	cacheKey := s.buildCacheKey("anomaly:calendar",
		clusterIDStr,
		startTime.Format("2006-01-02"),
		endTime.Format("2006-01-02"),
	)

	// 尝试从缓存获取
	ctx := context.Background()
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var data []model.CalendarDataPoint
		if err := json.Unmarshal(cached, &data); err == nil {
			return data, nil
		}
	}

	// 查询数据库
	dateFormat := "DATE(start_time)"
	if s.db.Dialector.Name() == "postgres" {
		dateFormat = "DATE(start_time)"
	}

	query := fmt.Sprintf(`
		SELECT 
			%s as date,
			COUNT(*) as value
		FROM node_anomalies
		WHERE start_time >= ? AND start_time <= ?
	`, dateFormat)

	args := []interface{}{startTime, endTime}
	if clusterID != nil {
		query += " AND cluster_id = ?"
		args = append(args, *clusterID)
	}

	query += " GROUP BY date ORDER BY date"

	var data []model.CalendarDataPoint
	if err := s.db.Raw(query, args...).Scan(&data).Error; err != nil {
		return nil, fmt.Errorf("failed to get calendar data: %w", err)
	}

	// 写入缓存
	cacheTTL := 10 * time.Minute
	if dataBytes, err := json.Marshal(data); err == nil {
		if err := s.cache.Set(ctx, cacheKey, dataBytes, cacheTTL); err != nil {
			s.logger.Warningf("Failed to cache calendar data: %v", err)
		}
	}

	return data, nil
}

// GetTopUnhealthyNodes 获取健康度最低的节点列表
func (s *Service) GetTopUnhealthyNodes(clusterID *uint, limit int, startTime, endTime *time.Time) ([]model.NodeHealthScore, error) {
	// 设置默认时间范围（最近30天）
	if startTime == nil {
		t := time.Now().AddDate(0, 0, -30)
		startTime = &t
	}
	if endTime == nil {
		t := time.Now()
		endTime = &t
	}

	if limit <= 0 {
		limit = 10
	}

	// 获取所有有异常的节点列表
	query := s.db.Model(&model.NodeAnomaly{}).
		Select("DISTINCT cluster_id, node_name").
		Where("start_time >= ? AND start_time <= ?", startTime, endTime)

	if clusterID != nil {
		query = query.Where("cluster_id = ?", *clusterID)
	}

	var nodes []struct {
		ClusterID uint
		NodeName  string
	}

	if err := query.Find(&nodes).Error; err != nil {
		return nil, fmt.Errorf("failed to get node list: %w", err)
	}

	// 计算每个节点的健康度评分
	var scores []model.NodeHealthScore
	for _, node := range nodes {
		score, err := s.GetNodeHealthScore(node.ClusterID, node.NodeName, startTime, endTime)
		if err != nil {
			s.logger.Warningf("Failed to get health score for node %s: %v", node.NodeName, err)
			continue
		}
		scores = append(scores, *score)
	}

	// 按健康度评分排序（从低到高）
	// 使用简单的冒泡排序
	for i := 0; i < len(scores); i++ {
		for j := i + 1; j < len(scores); j++ {
			if scores[i].HealthScore > scores[j].HealthScore {
				scores[i], scores[j] = scores[j], scores[i]
			}
		}
	}

	// 返回前 N 个
	if len(scores) > limit {
		scores = scores[:limit]
	}

	return scores, nil
}
