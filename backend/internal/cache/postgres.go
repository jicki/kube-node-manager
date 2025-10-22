package cache

import (
	"context"
	"fmt"
	"strings"
	"time"

	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// PostgresCache PostgreSQL 缓存实现
type PostgresCache struct {
	db        *gorm.DB
	tableName string
	logger    *logger.Logger
	stopChan  chan struct{}
}

// NewPostgresCache 创建 PostgreSQL 缓存实例
func NewPostgresCache(db *gorm.DB, tableName string, cleanupInterval int, useUnlogged bool, logger *logger.Logger) (*PostgresCache, error) {
	pc := &PostgresCache{
		db:        db,
		tableName: tableName,
		logger:    logger,
		stopChan:  make(chan struct{}),
	}

	// 自动迁移表
	if err := db.AutoMigrate(&model.CacheEntry{}); err != nil {
		return nil, fmt.Errorf("failed to migrate cache table: %w", err)
	}

	// 如果配置了 UNLOGGED，设置表属性（提升性能，重启后数据丢失）
	if useUnlogged {
		result := db.Exec(fmt.Sprintf("ALTER TABLE %s SET UNLOGGED", tableName))
		if result.Error != nil {
			logger.Warningf("Failed to set table as UNLOGGED (may already be set): %v", result.Error)
		} else {
			logger.Infof("Cache table %s set to UNLOGGED mode for better performance", tableName)
		}
	}

	// 启动定期清理
	if cleanupInterval > 0 {
		go pc.cleanupLoop(time.Duration(cleanupInterval) * time.Second)
		logger.Infof("Started cache cleanup loop with interval %d seconds", cleanupInterval)
	}

	logger.Infof("PostgreSQL cache initialized with table: %s", tableName)
	return pc, nil
}

// Get 获取缓存
func (c *PostgresCache) Get(ctx context.Context, key string) ([]byte, error) {
	var entry model.CacheEntry
	err := c.db.WithContext(ctx).
		Where("key = ? AND expires_at > ?", key, time.Now()).
		First(&entry).Error

	if err == gorm.ErrRecordNotFound {
		return nil, ErrCacheMiss
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	return entry.Value, nil
}

// Set 设置缓存
func (c *PostgresCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	entry := model.CacheEntry{
		Key:       key,
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}

	// UPSERT 操作（插入或更新）
	err := c.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "expires_at", "updated_at"}),
		}).
		Create(&entry).Error

	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	return nil
}

// Delete 删除缓存
func (c *PostgresCache) Delete(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	err := c.db.WithContext(ctx).
		Where("key IN ?", keys).
		Delete(&model.CacheEntry{}).Error

	if err != nil {
		return fmt.Errorf("failed to delete cache: %w", err)
	}

	return nil
}

// Clear 清除匹配模式的缓存（支持通配符 *）
func (c *PostgresCache) Clear(ctx context.Context, pattern string) error {
	// 将通配符 * 转换为 SQL 的 %
	sqlPattern := strings.ReplaceAll(pattern, "*", "%")

	err := c.db.WithContext(ctx).
		Where("key LIKE ?", sqlPattern).
		Delete(&model.CacheEntry{}).Error

	if err != nil {
		return fmt.Errorf("failed to clear cache: %w", err)
	}

	return nil
}

// Exists 检查缓存是否存在
func (c *PostgresCache) Exists(ctx context.Context, key string) (bool, error) {
	var count int64
	err := c.db.WithContext(ctx).
		Model(&model.CacheEntry{}).
		Where("key = ? AND expires_at > ?", key, time.Now()).
		Count(&count).Error

	if err != nil {
		return false, fmt.Errorf("failed to check cache existence: %w", err)
	}

	return count > 0, nil
}

// Close 关闭缓存连接
func (c *PostgresCache) Close() error {
	close(c.stopChan)
	c.logger.Infof("PostgreSQL cache closed")
	return nil
}

// cleanupLoop 定期清理过期缓存
func (c *PostgresCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopChan:
			c.logger.Infof("Cache cleanup loop stopped")
			return
		}
	}
}

// cleanup 清理过期缓存
func (c *PostgresCache) cleanup() {
	ctx := context.Background()
	result := c.db.WithContext(ctx).
		Where("expires_at < ?", time.Now()).
		Delete(&model.CacheEntry{})

	if result.Error != nil {
		c.logger.Errorf("Failed to cleanup expired cache entries: %v", result.Error)
		return
	}

	if result.RowsAffected > 0 {
		c.logger.Infof("Cleaned up %d expired cache entries", result.RowsAffected)
	}
}
