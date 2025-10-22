package cache

import (
	"context"
	"strings"
	"sync"
	"time"

	"kube-node-manager/pkg/logger"
)

// cacheItem 内存缓存项
type cacheItem struct {
	Value     []byte
	ExpiresAt time.Time
}

// MemoryCache 内存缓存实现
// ⚠️ 警告：不推荐用于多副本部署，会导致缓存不一致
type MemoryCache struct {
	data     sync.Map
	logger   *logger.Logger
	stopChan chan struct{}
}

// NewMemoryCache 创建内存缓存实例
func NewMemoryCache(logger *logger.Logger) *MemoryCache {
	mc := &MemoryCache{
		logger:   logger,
		stopChan: make(chan struct{}),
	}

	// 启动定期清理
	go mc.cleanupLoop(5 * time.Minute)

	logger.Warningf("Memory cache initialized - NOT recommended for multi-replica deployment!")
	return mc
}

// Get 获取缓存
func (m *MemoryCache) Get(ctx context.Context, key string) ([]byte, error) {
	value, ok := m.data.Load(key)
	if !ok {
		return nil, ErrCacheMiss
	}

	item := value.(cacheItem)
	if time.Now().After(item.ExpiresAt) {
		m.data.Delete(key)
		return nil, ErrCacheMiss
	}

	return item.Value, nil
}

// Set 设置缓存
func (m *MemoryCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	item := cacheItem{
		Value:     value,
		ExpiresAt: time.Now().Add(ttl),
	}
	m.data.Store(key, item)
	return nil
}

// Delete 删除缓存
func (m *MemoryCache) Delete(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		m.data.Delete(key)
	}
	return nil
}

// Clear 清除匹配模式的缓存
func (m *MemoryCache) Clear(ctx context.Context, pattern string) error {
	// 将通配符 * 转换为前缀匹配
	prefix := strings.TrimSuffix(pattern, "*")

	m.data.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if strings.HasPrefix(keyStr, prefix) {
			m.data.Delete(key)
		}
		return true
	})

	return nil
}

// Exists 检查缓存是否存在
func (m *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	value, ok := m.data.Load(key)
	if !ok {
		return false, nil
	}

	item := value.(cacheItem)
	if time.Now().After(item.ExpiresAt) {
		m.data.Delete(key)
		return false, nil
	}

	return true, nil
}

// Close 关闭缓存
func (m *MemoryCache) Close() error {
	close(m.stopChan)
	m.logger.Infof("Memory cache closed")
	return nil
}

// cleanupLoop 定期清理过期缓存
func (m *MemoryCache) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.cleanup()
		case <-m.stopChan:
			return
		}
	}
}

// cleanup 清理过期缓存
func (m *MemoryCache) cleanup() {
	now := time.Now()
	count := 0

	m.data.Range(func(key, value interface{}) bool {
		item := value.(cacheItem)
		if now.After(item.ExpiresAt) {
			m.data.Delete(key)
			count++
		}
		return true
	})

	if count > 0 {
		m.logger.Infof("Cleaned up %d expired memory cache entries", count)
	}
}
