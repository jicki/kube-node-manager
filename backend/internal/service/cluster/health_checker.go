package cluster

import (
	"sync"
	"time"
)

// ClusterHealth 集群健康状态
type ClusterHealth struct {
	Name          string    // 集群名称
	IsHealthy     bool      // 是否健康
	LastCheckTime time.Time // 上次检查时间
	FailureCount  int       // 连续失败次数
	LastError     error     // 最后一次错误
	CircuitOpen   bool      // 断路器是否打开
}

// HealthChecker 健康检查器
// 实现断路器模式（Circuit Breaker Pattern）：
// - 连续失败达到阈值后打开断路器，跳过后续请求
// - 经过恢复时间后尝试半开状态（Half-Open）
// - 成功后关闭断路器，恢复正常
// 智能重试策略（指数退避）：
// - 首次失败：立即重试
// - 第2次失败：等待 baseRetryDelay（2秒）
// - 第N次失败：等待 baseRetryDelay * 2^(N-2)，最多 maxRetryDelay（5分钟）
type HealthChecker struct {
	healthMap        map[string]*ClusterHealth
	mu               sync.RWMutex
	failureThreshold int           // 失败阈值（默认3次）
	recoveryTime     time.Duration // 恢复时间（默认5分钟）
	baseRetryDelay   time.Duration // 基础重试延迟（默认2秒）
	maxRetryDelay    time.Duration // 最大重试延迟（默认5分钟）
}

// NewHealthChecker 创建健康检查器
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		healthMap:        make(map[string]*ClusterHealth),
		failureThreshold: 3,              // 连续失败3次打开断路器
		recoveryTime:     5 * time.Minute, // 5分钟后尝试恢复
		baseRetryDelay:   2 * time.Second, // 基础重试延迟2秒
		maxRetryDelay:    5 * time.Minute, // 最大重试延迟5分钟
	}
}

// RecordSuccess 记录成功
// 重置失败计数器，关闭断路器
func (h *HealthChecker) RecordSuccess(clusterName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.healthMap[clusterName]
	if !exists {
		health = &ClusterHealth{
			Name: clusterName,
		}
		h.healthMap[clusterName] = health
	}

	health.IsHealthy = true
	health.FailureCount = 0
	health.LastCheckTime = time.Now()
	health.CircuitOpen = false
	health.LastError = nil
}

// RecordFailure 记录失败
// 增加失败计数，达到阈值后打开断路器
func (h *HealthChecker) RecordFailure(clusterName string, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.healthMap[clusterName]
	if !exists {
		health = &ClusterHealth{
			Name: clusterName,
		}
		h.healthMap[clusterName] = health
	}

	health.IsHealthy = false
	health.FailureCount++
	health.LastError = err
	health.LastCheckTime = time.Now()

	// 达到失败阈值，打开断路器
	if health.FailureCount >= h.failureThreshold {
		health.CircuitOpen = true
	}
}

// ShouldSkip 判断是否应该跳过（断路器打开且未到恢复时间）
// 返回 true 表示应该跳过该集群的操作
// 实现指数退避策略
func (h *HealthChecker) ShouldSkip(clusterName string) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health, exists := h.healthMap[clusterName]
	if !exists {
		return false // 首次尝试，不跳过
	}

	// 计算指数退避延迟
	retryDelay := h.calculateRetryDelay(health.FailureCount)

	// 断路器打开
	if health.CircuitOpen {
		// 检查是否到达恢复时间（使用指数退避）
		if time.Since(health.LastCheckTime) < retryDelay {
			return true // 跳过，还未到重试时间
		}
		// 到达恢复时间，进入半开状态（Half-Open），允许尝试
		// 如果这次尝试成功，会通过 RecordSuccess 关闭断路器
		// 如果失败，会继续打开断路器
	} else if health.FailureCount > 0 {
		// 断路器未打开，但有失败记录，使用指数退避
		if time.Since(health.LastCheckTime) < retryDelay {
			return true // 跳过，还未到重试时间
		}
	}

	return false
}

// calculateRetryDelay 计算重试延迟（指数退避）
// 策略：
// - 首次失败（failureCount=1）：立即重试（0延迟）
// - 第2次失败：baseRetryDelay（2秒）
// - 第3次失败：baseRetryDelay * 2（4秒）
// - 第N次失败：baseRetryDelay * 2^(N-2)
// - 最大延迟：maxRetryDelay（5分钟）
func (h *HealthChecker) calculateRetryDelay(failureCount int) time.Duration {
	if failureCount <= 1 {
		return 0 // 首次失败，立即重试
	}

	// 计算指数退避：baseRetryDelay * 2^(failureCount-2)
	exponent := failureCount - 2
	delay := h.baseRetryDelay * time.Duration(1<<uint(exponent))

	// 限制最大延迟
	if delay > h.maxRetryDelay {
		return h.maxRetryDelay
	}

	return delay
}

// GetHealth 获取集群健康状态
func (h *HealthChecker) GetHealth(clusterName string) *ClusterHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	health, exists := h.healthMap[clusterName]
	if !exists {
		return &ClusterHealth{
			Name:      clusterName,
			IsHealthy: true, // 默认健康
		}
	}

	// 返回副本，避免外部修改
	return &ClusterHealth{
		Name:          health.Name,
		IsHealthy:     health.IsHealthy,
		LastCheckTime: health.LastCheckTime,
		FailureCount:  health.FailureCount,
		LastError:     health.LastError,
		CircuitOpen:   health.CircuitOpen,
	}
}

// GetAllHealth 获取所有集群的健康状态
func (h *HealthChecker) GetAllHealth() map[string]*ClusterHealth {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[string]*ClusterHealth)
	for name, health := range h.healthMap {
		result[name] = &ClusterHealth{
			Name:          health.Name,
			IsHealthy:     health.IsHealthy,
			LastCheckTime: health.LastCheckTime,
			FailureCount:  health.FailureCount,
			LastError:     health.LastError,
			CircuitOpen:   health.CircuitOpen,
		}
	}
	return result
}

// ResetCircuitBreaker 重置断路器（手动恢复）
// 用于管理员手动重置失败的集群
func (h *HealthChecker) ResetCircuitBreaker(clusterName string) {
	h.mu.Lock()
	defer h.mu.Unlock()

	health, exists := h.healthMap[clusterName]
	if exists {
		health.CircuitOpen = false
		health.FailureCount = 0
		health.LastCheckTime = time.Now()
	}
}

// SetFailureThreshold 设置失败阈值
func (h *HealthChecker) SetFailureThreshold(threshold int) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.failureThreshold = threshold
}

// SetRecoveryTime 设置恢复时间
func (h *HealthChecker) SetRecoveryTime(duration time.Duration) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recoveryTime = duration
}

