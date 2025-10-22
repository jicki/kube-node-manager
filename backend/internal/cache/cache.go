package cache

import (
	"context"
	"errors"
	"time"
)

// 缓存相关错误
var (
	ErrCacheMiss = errors.New("cache miss")
	ErrCacheSet  = errors.New("failed to set cache")
	ErrCacheDel  = errors.New("failed to delete cache")
)

// Cache 缓存接口（支持多种实现）
type Cache interface {
	// Get 获取缓存
	Get(ctx context.Context, key string) ([]byte, error)

	// Set 设置缓存
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error

	// Delete 删除缓存
	Delete(ctx context.Context, keys ...string) error

	// Clear 清除匹配模式的缓存（支持通配符 *）
	Clear(ctx context.Context, pattern string) error

	// Exists 检查缓存是否存在
	Exists(ctx context.Context, key string) (bool, error)

	// Close 关闭缓存连接
	Close() error
}
