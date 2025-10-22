package anomaly

import (
	"fmt"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// CleanupConfig 清理配置
type CleanupConfig struct {
	Enabled       bool   `json:"enabled"`        // 是否启用自动清理
	RetentionDays int    `json:"retention_days"` // 保留天数
	CleanupTime   string `json:"cleanup_time"`   // 清理时间（格式：HH:MM）
	BatchSize     int    `json:"batch_size"`     // 批量删除大小
}

// CleanupService 数据清理服务
type CleanupService struct {
	db       *gorm.DB
	logger   *logger.Logger
	config   *CleanupConfig
	stopChan chan struct{}
}

// NewCleanupService 创建清理服务
func NewCleanupService(db *gorm.DB, logger *logger.Logger, config *CleanupConfig) *CleanupService {
	if config.RetentionDays == 0 {
		config.RetentionDays = 90 // 默认保留90天
	}
	if config.CleanupTime == "" {
		config.CleanupTime = "02:00" // 默认凌晨2点
	}
	if config.BatchSize == 0 {
		config.BatchSize = 1000 // 默认每批1000条
	}

	return &CleanupService{
		db:       db,
		logger:   logger,
		config:   config,
		stopChan: make(chan struct{}),
	}
}

// Start 启动自动清理服务
func (s *CleanupService) Start() {
	if !s.config.Enabled {
		s.logger.Info("Data cleanup service is disabled")
		return
	}

	s.logger.Infof("Starting data cleanup service (retention: %d days, time: %s)",
		s.config.RetentionDays, s.config.CleanupTime)

	go s.cleanupLoop()
}

// Stop 停止清理服务
func (s *CleanupService) Stop() {
	if !s.config.Enabled {
		return
	}

	s.logger.Info("Stopping data cleanup service...")
	close(s.stopChan)
	s.logger.Info("Data cleanup service stopped")
}

// cleanupLoop 定时清理循环
func (s *CleanupService) cleanupLoop() {
	// 计算首次执行时间
	nextRun := s.calculateNextRunTime()
	s.logger.Infof("Next cleanup scheduled at: %s", nextRun.Format("2006-01-02 15:04:05"))

	for {
		now := time.Now()
		duration := nextRun.Sub(now)

		select {
		case <-time.After(duration):
			// 执行清理
			if err := s.Cleanup(); err != nil {
				s.logger.Errorf("Scheduled cleanup failed: %v", err)
			}
			// 计算下次执行时间（明天同一时间）
			nextRun = s.calculateNextRunTime()
			s.logger.Infof("Next cleanup scheduled at: %s", nextRun.Format("2006-01-02 15:04:05"))

		case <-s.stopChan:
			return
		}
	}
}

// calculateNextRunTime 计算下次执行时间
func (s *CleanupService) calculateNextRunTime() time.Time {
	now := time.Now()

	// 解析清理时间
	hour, minute := 2, 0
	fmt.Sscanf(s.config.CleanupTime, "%d:%d", &hour, &minute)

	// 今天的清理时间
	nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location())

	// 如果今天的清理时间已经过了，设置为明天
	if now.After(nextRun) {
		nextRun = nextRun.Add(24 * time.Hour)
	}

	return nextRun
}

// Cleanup 执行清理操作
func (s *CleanupService) Cleanup() error {
	startTime := time.Now()
	s.logger.Infof("Starting data cleanup (retention: %d days)", s.config.RetentionDays)

	// 计算截止日期
	cutoffDate := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	// 统计待清理的记录数
	var totalCount int64
	if err := s.db.Model(&model.NodeAnomaly{}).
		Where("status = ? AND end_time < ?", model.AnomalyStatusResolved, cutoffDate).
		Count(&totalCount).Error; err != nil {
		return fmt.Errorf("failed to count records for cleanup: %w", err)
	}

	if totalCount == 0 {
		s.logger.Info("No records to cleanup")
		return nil
	}

	s.logger.Infof("Found %d records to cleanup", totalCount)

	// 分批删除
	deletedCount := int64(0)
	for {
		result := s.db.
			Where("status = ? AND end_time < ?", model.AnomalyStatusResolved, cutoffDate).
			Limit(s.config.BatchSize).
			Delete(&model.NodeAnomaly{})

		if result.Error != nil {
			return fmt.Errorf("failed to delete records: %w", result.Error)
		}

		deletedCount += result.RowsAffected
		s.logger.Infof("Cleanup progress: %d/%d", deletedCount, totalCount)

		// 如果没有更多记录，退出循环
		if result.RowsAffected == 0 {
			break
		}

		// 短暂休息，避免长时间锁表
		time.Sleep(100 * time.Millisecond)
	}

	duration := time.Since(startTime)
	s.logger.Infof("Data cleanup completed: %d records deleted in %v", deletedCount, duration)

	return nil
}

// CleanupByDate 按日期清理（用于手动清理）
func (s *CleanupService) CleanupByDate(beforeDate time.Time) (int64, error) {
	s.logger.Infof("Manual cleanup: deleting records before %s", beforeDate.Format("2006-01-02"))

	var deletedCount int64
	for {
		result := s.db.
			Where("status = ? AND end_time < ?", model.AnomalyStatusResolved, beforeDate).
			Limit(s.config.BatchSize).
			Delete(&model.NodeAnomaly{})

		if result.Error != nil {
			return deletedCount, fmt.Errorf("failed to delete records: %w", result.Error)
		}

		deletedCount += result.RowsAffected

		if result.RowsAffected == 0 {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	s.logger.Infof("Manual cleanup completed: %d records deleted", deletedCount)
	return deletedCount, nil
}

// GetCleanupStats 获取清理统计信息
func (s *CleanupService) GetCleanupStats() (map[string]interface{}, error) {
	cutoffDate := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	// 统计待清理的记录数
	var pendingCount int64
	if err := s.db.Model(&model.NodeAnomaly{}).
		Where("status = ? AND end_time < ?", model.AnomalyStatusResolved, cutoffDate).
		Count(&pendingCount).Error; err != nil {
		return nil, fmt.Errorf("failed to count pending cleanup records: %w", err)
	}

	// 统计总的已恢复记录数
	var totalResolved int64
	if err := s.db.Model(&model.NodeAnomaly{}).
		Where("status = ?", model.AnomalyStatusResolved).
		Count(&totalResolved).Error; err != nil {
		return nil, fmt.Errorf("failed to count resolved records: %w", err)
	}

	// 统计最旧的记录
	var oldestAnomaly model.NodeAnomaly
	s.db.Where("status = ?", model.AnomalyStatusResolved).
		Order("end_time ASC").
		First(&oldestAnomaly)

	stats := map[string]interface{}{
		"enabled":         s.config.Enabled,
		"retention_days":  s.config.RetentionDays,
		"cleanup_time":    s.config.CleanupTime,
		"pending_cleanup": pendingCount,
		"total_resolved":  totalResolved,
		"cutoff_date":     cutoffDate.Format("2006-01-02"),
		"next_cleanup":    s.calculateNextRunTime().Format("2006-01-02 15:04:05"),
	}

	if oldestAnomaly.ID > 0 && oldestAnomaly.EndTime != nil {
		stats["oldest_record"] = oldestAnomaly.EndTime.Format("2006-01-02 15:04:05")
	}

	return stats, nil
}

// UpdateConfig 更新清理配置
func (s *CleanupService) UpdateConfig(config *CleanupConfig) error {
	if config.RetentionDays < 1 {
		return fmt.Errorf("retention_days must be greater than 0")
	}

	if config.BatchSize < 100 || config.BatchSize > 10000 {
		return fmt.Errorf("batch_size must be between 100 and 10000")
	}

	// 验证时间格式
	var hour, minute int
	if _, err := fmt.Sscanf(config.CleanupTime, "%d:%d", &hour, &minute); err != nil {
		return fmt.Errorf("invalid cleanup_time format, expected HH:MM")
	}
	if hour < 0 || hour > 23 || minute < 0 || minute > 59 {
		return fmt.Errorf("invalid cleanup_time, hour must be 0-23 and minute must be 0-59")
	}

	s.config.RetentionDays = config.RetentionDays
	s.config.CleanupTime = config.CleanupTime
	s.config.BatchSize = config.BatchSize
	s.config.Enabled = config.Enabled

	s.logger.Infof("Cleanup configuration updated: retention=%d days, time=%s, enabled=%v",
		config.RetentionDays, config.CleanupTime, config.Enabled)

	return nil
}

// GetConfig 获取当前配置
func (s *CleanupService) GetConfig() *CleanupConfig {
	return &CleanupConfig{
		Enabled:       s.config.Enabled,
		RetentionDays: s.config.RetentionDays,
		CleanupTime:   s.config.CleanupTime,
		BatchSize:     s.config.BatchSize,
	}
}
