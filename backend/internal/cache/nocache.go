package cache

import (
	"context"
	"time"
)

// NoCache 无缓存实现（禁用缓存时使用）
type NoCache struct{}

// NewNoCache 创建无缓存实例
func NewNoCache() *NoCache {
	return &NoCache{}
}

// Get 总是返回缓存未命中
func (n *NoCache) Get(ctx context.Context, key string) ([]byte, error) {
	return nil, ErrCacheMiss
}

// Set 不执行任何操作
func (n *NoCache) Set(ctx context.Context, key string, value []byte, ttl time.Duration) error {
	return nil
}

// Delete 不执行任何操作
func (n *NoCache) Delete(ctx context.Context, keys ...string) error {
	return nil
}

// Clear 不执行任何操作
func (n *NoCache) Clear(ctx context.Context, pattern string) error {
	return nil
}

// Exists 总是返回 false
func (n *NoCache) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}

// Close 不执行任何操作
func (n *NoCache) Close() error {
	return nil
}
