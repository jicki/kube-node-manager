package k8s

import (
	"sync"
	"time"
)

// ConnectionPool 连接池统计和管理
// 用于监控和限制Kubernetes客户端连接
type ConnectionPool struct {
	// 连接统计
	stats map[string]*ConnectionStats
	mu    sync.RWMutex

	// 连接限制
	maxConnections     int           // 最大连接数
	connectionTimeout  time.Duration // 连接超时时间
	idleTimeout        time.Duration // 空闲超时时间（未使用）
	healthCheckEnabled bool          // 是否启用健康检查
}

// ConnectionStats 连接统计信息
type ConnectionStats struct {
	ClusterName      string    // 集群名称
	CreatedAt        time.Time // 创建时间
	LastUsedAt       time.Time // 最后使用时间
	RequestCount     int64     // 请求次数
	SuccessCount     int64     // 成功次数
	FailureCount     int64     // 失败次数
	AverageLatencyMs float64   // 平均延迟（毫秒）
	IsHealthy        bool      // 是否健康
}

// NewConnectionPool 创建连接池
func NewConnectionPool() *ConnectionPool {
	return &ConnectionPool{
		stats:              make(map[string]*ConnectionStats),
		maxConnections:     100,          // 默认最大100个连接
		connectionTimeout:  15 * time.Second, // 连接超时15秒
		idleTimeout:        30 * time.Minute, // 空闲30分钟
		healthCheckEnabled: true,
	}
}

// RegisterConnection 注册连接
func (p *ConnectionPool) RegisterConnection(clusterName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if _, exists := p.stats[clusterName]; !exists {
		p.stats[clusterName] = &ConnectionStats{
			ClusterName: clusterName,
			CreatedAt:   time.Now(),
			LastUsedAt:  time.Now(),
			IsHealthy:   true,
		}
	}
}

// UnregisterConnection 注销连接
func (p *ConnectionPool) UnregisterConnection(clusterName string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	delete(p.stats, clusterName)
}

// RecordRequest 记录请求
func (p *ConnectionPool) RecordRequest(clusterName string, success bool, latencyMs float64) {
	p.mu.Lock()
	defer p.mu.Unlock()

	stats, exists := p.stats[clusterName]
	if !exists {
		// 如果连接不存在，创建它
		stats = &ConnectionStats{
			ClusterName: clusterName,
			CreatedAt:   time.Now(),
			IsHealthy:   true,
		}
		p.stats[clusterName] = stats
	}

	stats.LastUsedAt = time.Now()
	stats.RequestCount++

	if success {
		stats.SuccessCount++
		stats.IsHealthy = true
	} else {
		stats.FailureCount++
		// 连续失败超过3次，标记为不健康
		if stats.FailureCount >= 3 && stats.SuccessCount == 0 {
			stats.IsHealthy = false
		}
	}

	// 计算平均延迟（滑动平均）
	if stats.AverageLatencyMs == 0 {
		stats.AverageLatencyMs = latencyMs
	} else {
		// 使用指数移动平均（EMA）
		alpha := 0.3 // 平滑系数
		stats.AverageLatencyMs = alpha*latencyMs + (1-alpha)*stats.AverageLatencyMs
	}
}

// GetStats 获取连接统计
func (p *ConnectionPool) GetStats(clusterName string) *ConnectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	stats, exists := p.stats[clusterName]
	if !exists {
		return nil
	}

	// 返回副本
	return &ConnectionStats{
		ClusterName:      stats.ClusterName,
		CreatedAt:        stats.CreatedAt,
		LastUsedAt:       stats.LastUsedAt,
		RequestCount:     stats.RequestCount,
		SuccessCount:     stats.SuccessCount,
		FailureCount:     stats.FailureCount,
		AverageLatencyMs: stats.AverageLatencyMs,
		IsHealthy:        stats.IsHealthy,
	}
}

// GetAllStats 获取所有连接统计
func (p *ConnectionPool) GetAllStats() map[string]*ConnectionStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	result := make(map[string]*ConnectionStats, len(p.stats))
	for name, stats := range p.stats {
		result[name] = &ConnectionStats{
			ClusterName:      stats.ClusterName,
			CreatedAt:        stats.CreatedAt,
			LastUsedAt:       stats.LastUsedAt,
			RequestCount:     stats.RequestCount,
			SuccessCount:     stats.SuccessCount,
			FailureCount:     stats.FailureCount,
			AverageLatencyMs: stats.AverageLatencyMs,
			IsHealthy:        stats.IsHealthy,
		}
	}
	return result
}

// GetConnectionCount 获取当前连接数
func (p *ConnectionPool) GetConnectionCount() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.stats)
}

// CanAcceptConnection 检查是否可以接受新连接
func (p *ConnectionPool) CanAcceptConnection() bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.stats) < p.maxConnections
}

// SetMaxConnections 设置最大连接数
func (p *ConnectionPool) SetMaxConnections(max int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.maxConnections = max
}

// GetMaxConnections 获取最大连接数
func (p *ConnectionPool) GetMaxConnections() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.maxConnections
}

// CleanupIdleConnections 清理空闲连接（未使用超过idleTimeout的连接）
func (p *ConnectionPool) CleanupIdleConnections() []string {
	p.mu.Lock()
	defer p.mu.Unlock()

	idleClusters := make([]string, 0)
	now := time.Now()

	for name, stats := range p.stats {
		if now.Sub(stats.LastUsedAt) > p.idleTimeout {
			idleClusters = append(idleClusters, name)
			delete(p.stats, name)
		}
	}

	return idleClusters
}

// GetHealthySummary 获取健康状态摘要
func (p *ConnectionPool) GetHealthySummary() (healthy, unhealthy, total int) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	total = len(p.stats)
	for _, stats := range p.stats {
		if stats.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
	}
	return
}

