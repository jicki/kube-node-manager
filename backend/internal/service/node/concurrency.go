package node

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ConcurrencyController 动态并发控制器
// 根据操作类型、集群规模和网络延迟动态调整并发数
type ConcurrencyController struct {
	mu sync.RWMutex

	// 操作类型对应的基础并发配置
	operationConfigs map[string]*OperationConfig

	// 延迟监控
	latencyTracker *LatencyTracker
}

// OperationConfig 操作并发配置
type OperationConfig struct {
	BaseLimit int     // 基础并发数
	MaxLimit  int     // 最大并发数
	MinLimit  int     // 最小并发数
	Weight    float64 // 操作权重（越重延迟越高）
}

// LatencyTracker 延迟追踪器
type LatencyTracker struct {
	mu         sync.RWMutex
	latencies  []time.Duration // 最近的延迟记录
	maxSamples int             // 最大样本数
	avgLatency time.Duration   // 平均延迟
	lastUpdate time.Time       // 最后更新时间
}

// NewConcurrencyController 创建并发控制器
func NewConcurrencyController() *ConcurrencyController {
	return &ConcurrencyController{
		operationConfigs: map[string]*OperationConfig{
			"cordon": {
				BaseLimit: 15,
				MaxLimit:  20,
				MinLimit:  5,
				Weight:    0.3, // 轻量级操作
			},
			"uncordon": {
				BaseLimit: 15,
				MaxLimit:  20,
				MinLimit:  5,
				Weight:    0.3, // 轻量级操作
			},
			"label": {
				BaseLimit: 12,
				MaxLimit:  18,
				MinLimit:  5,
				Weight:    0.5, // 中等操作
			},
			"taint": {
				BaseLimit: 10,
				MaxLimit:  15,
				MinLimit:  3,
				Weight:    0.6, // 中等偏重操作
			},
			"drain": {
				BaseLimit: 5,
				MaxLimit:  8,
				MinLimit:  2,
				Weight:    1.0, // 重量级操作
			},
		},
		latencyTracker: &LatencyTracker{
			latencies:  make([]time.Duration, 0, 100),
			maxSamples: 100,
		},
	}
}

// Calculate 计算最优并发数
// 综合考虑：操作类型、集群规模、平均延迟
func (c *ConcurrencyController) Calculate(operationType string, clusterSize int, avgLatency time.Duration) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 获取操作配置
	config, exists := c.operationConfigs[operationType]
	if !exists {
		config = &OperationConfig{
			BaseLimit: 10,
			MaxLimit:  15,
			MinLimit:  5,
			Weight:    0.5,
		}
	}

	// 基础并发数
	concurrency := config.BaseLimit

	// 1. 根据集群规模调整
	// 小集群(<50节点): 基础并发数
	// 中集群(50-200节点): +20%
	// 大集群(>200节点): +50%
	if clusterSize > 200 {
		concurrency = int(float64(concurrency) * 1.5)
	} else if clusterSize > 50 {
		concurrency = int(float64(concurrency) * 1.2)
	}

	// 2. 根据平均延迟调整
	// 如果平均延迟过高，降低并发数避免过载
	if avgLatency > 0 {
		switch {
		case avgLatency > 5*time.Second:
			// 延迟非常高，大幅降低并发
			concurrency = int(float64(concurrency) * 0.4)
		case avgLatency > 3*time.Second:
			// 延迟高，降低并发
			concurrency = int(float64(concurrency) * 0.6)
		case avgLatency > 2*time.Second:
			// 延迟中等，适度降低并发
			concurrency = int(float64(concurrency) * 0.8)
		case avgLatency < 500*time.Millisecond:
			// 延迟很低，可以适度提升并发
			concurrency = int(float64(concurrency) * 1.2)
		}
	}

	// 3. 限制在最小和最大范围内
	if concurrency < config.MinLimit {
		concurrency = config.MinLimit
	}
	if concurrency > config.MaxLimit {
		concurrency = config.MaxLimit
	}

	return concurrency
}

// RecordLatency 记录操作延迟
func (c *ConcurrencyController) RecordLatency(latency time.Duration) {
	c.latencyTracker.mu.Lock()
	defer c.latencyTracker.mu.Unlock()

	// 添加新的延迟记录
	c.latencyTracker.latencies = append(c.latencyTracker.latencies, latency)

	// 保持样本数量在限制内
	if len(c.latencyTracker.latencies) > c.latencyTracker.maxSamples {
		c.latencyTracker.latencies = c.latencyTracker.latencies[1:]
	}

	// 更新平均延迟
	c.latencyTracker.updateAverage()
	c.latencyTracker.lastUpdate = time.Now()
}

// GetAverageLatency 获取平均延迟
func (c *ConcurrencyController) GetAverageLatency() time.Duration {
	c.latencyTracker.mu.RLock()
	defer c.latencyTracker.mu.RUnlock()

	return c.latencyTracker.avgLatency
}

// updateAverage 更新平均延迟（内部方法，调用前需加锁）
func (lt *LatencyTracker) updateAverage() {
	if len(lt.latencies) == 0 {
		lt.avgLatency = 0
		return
	}

	var total time.Duration
	for _, latency := range lt.latencies {
		total += latency
	}
	lt.avgLatency = total / time.Duration(len(lt.latencies))
}

// ResetLatencies 重置延迟统计
func (c *ConcurrencyController) ResetLatencies() {
	c.latencyTracker.mu.Lock()
	defer c.latencyTracker.mu.Unlock()

	c.latencyTracker.latencies = make([]time.Duration, 0, c.latencyTracker.maxSamples)
	c.latencyTracker.avgLatency = 0
}

// GetStats 获取并发控制器统计信息
func (c *ConcurrencyController) GetStats() map[string]interface{} {
	c.latencyTracker.mu.RLock()
	defer c.latencyTracker.mu.RUnlock()

	return map[string]interface{}{
		"avg_latency_ms": c.latencyTracker.avgLatency.Milliseconds(),
		"sample_count":   len(c.latencyTracker.latencies),
		"last_update":    c.latencyTracker.lastUpdate,
	}
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries      int           // 最大重试次数
	InitialDelay    time.Duration // 初始延迟
	MaxDelay        time.Duration // 最大延迟
	BackoffFactor   float64       // 退避因子
	RetryableErrors []string      // 可重试的错误关键字
}

// DefaultRetryConfig 默认重试配置
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxRetries:    3,
		InitialDelay:  100 * time.Millisecond,
		MaxDelay:      5 * time.Second,
		BackoffFactor: 2.0,
		RetryableErrors: []string{
			"timeout",
			"connection refused",
			"temporary failure",
			"too many requests",
			"rate limit",
			"conflict",
			"object has been modified",
		},
	}
}

// ShouldRetry 判断是否应该重试
func (rc *RetryConfig) ShouldRetry(err error, attempt int) bool {
	if err == nil || attempt >= rc.MaxRetries {
		return false
	}

	errStr := err.Error()
	for _, keyword := range rc.RetryableErrors {
		if contains(errStr, keyword) {
			return true
		}
	}

	return false
}

// GetDelay 获取重试延迟时间
func (rc *RetryConfig) GetDelay(attempt int) time.Duration {
	delay := time.Duration(float64(rc.InitialDelay) * pow(rc.BackoffFactor, float64(attempt)))
	if delay > rc.MaxDelay {
		delay = rc.MaxDelay
	}
	return delay
}

// RetryWithBackoff 带重试的操作执行
func RetryWithBackoff(ctx context.Context, config *RetryConfig, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= config.MaxRetries; attempt++ {
		// 检查context是否已取消
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// 执行操作
		err := operation()
		if err == nil {
			return nil // 成功
		}

		lastErr = err

		// 判断是否应该重试
		if !config.ShouldRetry(err, attempt) {
			return err // 不可重试的错误，直接返回
		}

		// 如果还有重试机会，等待后重试
		if attempt < config.MaxRetries {
			delay := config.GetDelay(attempt)
			select {
			case <-time.After(delay):
				// 继续重试
			case <-ctx.Done():
				return ctx.Err()
			}
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxRetries+1, lastErr)
}

// 辅助函数

// contains 检查字符串是否包含子串（不区分大小写）
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			len(s) > 0 && len(substr) > 0 &&
				indexIgnoreCase(s, substr) >= 0)
}

// indexIgnoreCase 不区分大小写查找子串
func indexIgnoreCase(s, substr string) int {
	s = toLower(s)
	substr = toLower(substr)
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}

// toLower 转换为小写
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if 'A' <= c && c <= 'Z' {
			c = c + ('a' - 'A')
		}
		b[i] = c
	}
	return string(b)
}

// pow 计算幂
func pow(base, exp float64) float64 {
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}
