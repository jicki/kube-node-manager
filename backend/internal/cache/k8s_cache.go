package cache

import (
	"context"
	"fmt"
	"sync"
	"time"

	"kube-node-manager/pkg/logger"
)

// K8sCache Kubernetes API缓存层
// 提供节点列表和节点详情的多层缓存，减少对K8s API Server的调用
type K8sCache struct {
	// 节点列表缓存: cluster -> CacheEntry[[]NodeInfo]
	nodeListCache *sync.Map

	// 节点详情缓存: "cluster:node" -> CacheEntry[NodeInfo]
	nodeDetailCache *sync.Map

	// Pod数量缓存: cluster -> CacheEntry[map[string]int]
	// 大规模集群优化：Pod数量查询很慢，使用独立缓存（5分钟TTL）
	podCountCache *sync.Map

	// 缓存配置
	listCacheTTL     time.Duration // 列表缓存TTL（默认30s）
	detailCacheTTL   time.Duration // 详情缓存TTL（默认5min）
	podCountCacheTTL time.Duration // Pod数量缓存TTL（默认5min）
	staleThreshold   time.Duration // 过期但可用阈值（默认5min）

	logger *logger.Logger

	// 刷新锁，避免并发刷新
	refreshLocks *sync.Map // cluster -> *sync.Mutex
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Data       interface{} // 缓存的数据
	UpdatedAt  time.Time   // 更新时间
	refreshing bool        // 是否正在刷新
	mu         sync.RWMutex
}

// NewK8sCache 创建K8s缓存实例
// 多副本部署优化：使用较短的TTL减少副本间数据不一致窗口
func NewK8sCache(logger *logger.Logger) *K8sCache {
	return &K8sCache{
		nodeListCache:    &sync.Map{},
		nodeDetailCache:  &sync.Map{},
		podCountCache:    &sync.Map{},
		listCacheTTL:     10 * time.Second, // 列表缓存10秒（多副本环境优化）
		detailCacheTTL:   1 * time.Minute,  // 详情缓存1分钟（多副本环境优化）
		podCountCacheTTL: 5 * time.Minute,  // Pod数量缓存5分钟（大规模集群优化）
		staleThreshold:   2 * time.Minute,  // 过期阈值2分钟（异步刷新窗口）
		logger:           logger,
		refreshLocks:     &sync.Map{},
	}
}

// GetNodeList 获取节点列表（带缓存）
// 缓存策略（多副本环境优化）：
// - <10s: 直接返回缓存（新鲜数据）
// - 10s-2min: 返回缓存并异步刷新（过期但可用）
// - >2min或forceRefresh: 同步刷新（强制更新）
func (c *K8sCache) GetNodeList(ctx context.Context, cluster string, forceRefresh bool, fetchFunc func() (interface{}, error)) (interface{}, error) {
	// 强制刷新，清除缓存
	if forceRefresh {
		c.nodeListCache.Delete(cluster)
		return c.fetchAndCacheNodeList(ctx, cluster, fetchFunc)
	}

	// 尝试从缓存获取
	if cached, ok := c.nodeListCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.RLock()
		age := time.Since(entry.UpdatedAt)
		data := entry.Data
		isRefreshing := entry.refreshing
		entry.mu.RUnlock()

		// 缓存新鲜，直接返回
		if age < c.listCacheTTL {
			// c.logger.Debugf("K8s cache hit: cluster=%s, age=%v", cluster, age)
			return data, nil
		}

		// 缓存过期但在阈值内，返回旧数据并异步刷新
		if age < c.staleThreshold {
			// c.logger.Debugf("K8s cache stale: cluster=%s, age=%v, async refresh", cluster, age)

			// 触发异步刷新（如果未在刷新中）
			if !isRefreshing {
				go c.asyncRefreshNodeList(cluster, fetchFunc)
			}

			return data, nil
		}

		// 缓存过期超过阈值，同步刷新
		// c.logger.Debugf("K8s cache expired: cluster=%s, age=%v, sync refresh", cluster, age)
	}

	// 缓存未命中或过期，同步获取
	return c.fetchAndCacheNodeList(ctx, cluster, fetchFunc)
}

// fetchAndCacheNodeList 获取并缓存节点列表
func (c *K8sCache) fetchAndCacheNodeList(ctx context.Context, cluster string, fetchFunc func() (interface{}, error)) (interface{}, error) {
	// 获取或创建刷新锁
	lockVal, _ := c.refreshLocks.LoadOrStore(cluster, &sync.Mutex{})
	lock := lockVal.(*sync.Mutex)

	lock.Lock()
	defer lock.Unlock()

	// 双重检查：可能其他goroutine已经刷新了
	if cached, ok := c.nodeListCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.RLock()
		age := time.Since(entry.UpdatedAt)
		entry.mu.RUnlock()

		if age < c.listCacheTTL {
			entry.mu.RLock()
			data := entry.Data
			entry.mu.RUnlock()
			return data, nil
		}
	}

	// 执行获取
	nodes, err := fetchFunc()
	if err != nil {
		c.logger.Errorf("Failed to fetch node list for cluster %s: %v", cluster, err)
		return nil, err
	}

	// 缓存结果
	entry := &CacheEntry{
		Data:      nodes,
		UpdatedAt: time.Now(),
	}
	c.nodeListCache.Store(cluster, entry)

	c.logger.Infof("K8s cache updated: cluster=%s", cluster)
	return nodes, nil
}

// asyncRefreshNodeList 异步刷新节点列表
func (c *K8sCache) asyncRefreshNodeList(cluster string, fetchFunc func() (interface{}, error)) {
	// 标记正在刷新
	if cached, ok := c.nodeListCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.Lock()
		if entry.refreshing {
			entry.mu.Unlock()
			return // 已经在刷新中
		}
		entry.refreshing = true
		entry.mu.Unlock()
	}

	// 执行刷新
	nodes, err := fetchFunc()

	// 更新缓存
	if cached, ok := c.nodeListCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()

		if err == nil {
			entry.Data = nodes
			entry.UpdatedAt = time.Now()
			c.logger.Infof("K8s cache async refreshed: cluster=%s", cluster)
		} else {
			c.logger.Warningf("K8s cache async refresh failed: cluster=%s, error=%v", cluster, err)
		}
		entry.refreshing = false
	}
}

// GetNodeDetail 获取节点详情（带缓存）
func (c *K8sCache) GetNodeDetail(ctx context.Context, cluster, nodeName string, forceRefresh bool, fetchFunc func() (interface{}, error)) (interface{}, error) {
	key := fmt.Sprintf("%s:%s", cluster, nodeName)

	// 强制刷新
	if forceRefresh {
		c.nodeDetailCache.Delete(key)
		return c.fetchAndCacheNodeDetail(ctx, key, fetchFunc)
	}

	// 尝试从缓存获取
	if cached, ok := c.nodeDetailCache.Load(key); ok {
		entry := cached.(*CacheEntry)
		entry.mu.RLock()
		age := time.Since(entry.UpdatedAt)
		data := entry.Data
		entry.mu.RUnlock()

		// 缓存有效
		if age < c.detailCacheTTL {
			// c.logger.Debugf("K8s node detail cache hit: %s, age=%v", key, age)
			return data, nil
		}

		// c.logger.Debugf("K8s node detail cache expired: %s, age=%v", key, age)
	}

	// 缓存未命中或过期
	return c.fetchAndCacheNodeDetail(ctx, key, fetchFunc)
}

// fetchAndCacheNodeDetail 获取并缓存节点详情
func (c *K8sCache) fetchAndCacheNodeDetail(ctx context.Context, key string, fetchFunc func() (interface{}, error)) (interface{}, error) {
	node, err := fetchFunc()
	if err != nil {
		return nil, err
	}

	// 缓存结果
	entry := &CacheEntry{
		Data:      node,
		UpdatedAt: time.Now(),
	}
	c.nodeDetailCache.Store(key, entry)

	return node, nil
}

// PrefetchNodeDetails 预取节点详情
// 用于优化用户体验，在列表加载后预取前N个节点的详情
func (c *K8sCache) PrefetchNodeDetails(cluster string, nodeNames []string, limit int, fetchFunc func(nodeName string) (interface{}, error)) {
	if limit <= 0 || limit > len(nodeNames) {
		limit = len(nodeNames)
	}

	c.logger.Infof("Prefetching %d node details for cluster %s", limit, cluster)

	// 限制并发预取数量
	sem := make(chan struct{}, 5) // 最多5个并发
	var wg sync.WaitGroup

	for i := 0; i < limit && i < len(nodeNames); i++ {
		nodeName := nodeNames[i]
		key := fmt.Sprintf("%s:%s", cluster, nodeName)

		// 检查是否已缓存
		if cached, ok := c.nodeDetailCache.Load(key); ok {
			entry := cached.(*CacheEntry)
			entry.mu.RLock()
			age := time.Since(entry.UpdatedAt)
			entry.mu.RUnlock()

			if age < c.detailCacheTTL {
				continue // 已有有效缓存，跳过
			}
		}

		wg.Add(1)
		go func(name string) {
			defer wg.Done()

			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量

			node, err := fetchFunc(name)
			if err != nil {
				// c.logger.Debugf("Prefetch failed for node %s: %v", name, err)
				return
			}

			// 缓存预取结果
			cacheKey := fmt.Sprintf("%s:%s", cluster, name)
			entry := &CacheEntry{
				Data:      node,
				UpdatedAt: time.Now(),
			}
			c.nodeDetailCache.Store(cacheKey, entry)
		}(nodeName)
	}

	wg.Wait()
	c.logger.Infof("Prefetch completed for cluster %s", cluster)
}

// InvalidateCluster 清除指定集群的所有缓存
func (c *K8sCache) InvalidateCluster(cluster string) {
	c.nodeListCache.Delete(cluster)

	// 清除该集群的所有节点详情缓存
	c.nodeDetailCache.Range(func(key, value interface{}) bool {
		keyStr := key.(string)
		if len(keyStr) > len(cluster) && keyStr[:len(cluster)] == cluster && keyStr[len(cluster)] == ':' {
			c.nodeDetailCache.Delete(key)
		}
		return true
	})

	c.logger.Infof("Invalidated K8s cache for cluster: %s", cluster)
}

// InvalidateNode 清除指定节点的缓存
func (c *K8sCache) InvalidateNode(cluster, nodeName string) {
	key := fmt.Sprintf("%s:%s", cluster, nodeName)
	c.nodeDetailCache.Delete(key)

	// 同时清除列表缓存，确保一致性
	c.nodeListCache.Delete(cluster)

	// c.logger.Debugf("Invalidated K8s cache for node: %s", key)
}

// GetCacheStats 获取缓存统计信息
func (c *K8sCache) GetCacheStats() map[string]interface{} {
	stats := map[string]interface{}{
		"list_cache_ttl":   c.listCacheTTL.String(),
		"detail_cache_ttl": c.detailCacheTTL.String(),
		"stale_threshold":  c.staleThreshold.String(),
	}

	// 统计列表缓存
	listCount := 0
	c.nodeListCache.Range(func(_, _ interface{}) bool {
		listCount++
		return true
	})
	stats["list_cache_count"] = listCount

	// 统计详情缓存
	detailCount := 0
	c.nodeDetailCache.Range(func(_, _ interface{}) bool {
		detailCount++
		return true
	})
	stats["detail_cache_count"] = detailCount

	return stats
}

// GetPodCounts 获取Pod数量缓存（带异步刷新）
// 大规模集群优化：
// - <5min: 直接返回缓存（新鲜数据）
// - 5min-10min: 返回缓存并异步刷新（过期但可用）
// - >10min或无缓存: 返回空map并异步加载
func (c *K8sCache) GetPodCounts(cluster string, nodeNames []string, fetchFunc func() map[string]int) map[string]int {
	// 尝试从缓存获取
	if cached, ok := c.podCountCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.RLock()
		age := time.Since(entry.UpdatedAt)
		data := entry.Data
		isRefreshing := entry.refreshing
		entry.mu.RUnlock()

		// 缓存新鲜，直接返回
		if age < c.podCountCacheTTL {
			if podCounts, ok := data.(map[string]int); ok {
				return podCounts
			}
		}

		// 缓存过期但在10分钟内，返回旧数据并异步刷新
		if age < c.podCountCacheTTL*2 {
			if podCounts, ok := data.(map[string]int); ok {
				// 触发异步刷新（如果未在刷新中）
				if !isRefreshing {
					go c.asyncRefreshPodCounts(cluster, fetchFunc)
				}

				return podCounts
			}
		}
	}

	// 缓存未命中或过期太久，异步加载并返回空map
	c.logger.Infof("Pod count cache miss: cluster=%s, triggering async load", cluster)
	go c.asyncRefreshPodCounts(cluster, fetchFunc)

	// 返回空map，不阻塞节点列表查询
	result := make(map[string]int)
	for _, nodeName := range nodeNames {
		result[nodeName] = 0
	}
	return result
}

// SetPodCounts 设置Pod数量缓存
func (c *K8sCache) SetPodCounts(cluster string, podCounts map[string]int) {
	entry := &CacheEntry{
		Data:      podCounts,
		UpdatedAt: time.Now(),
	}
	c.podCountCache.Store(cluster, entry)
	c.logger.Infof("Pod count cache updated: cluster=%s, nodes=%d", cluster, len(podCounts))
}

// asyncRefreshPodCounts 异步刷新Pod数量缓存
func (c *K8sCache) asyncRefreshPodCounts(cluster string, fetchFunc func() map[string]int) {
	// 标记正在刷新
	if cached, ok := c.podCountCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.Lock()
		if entry.refreshing {
			entry.mu.Unlock()
			return // 已经在刷新中
		}
		entry.refreshing = true
		entry.mu.Unlock()
	} else {
		// 创建新条目并标记为刷新中
		entry := &CacheEntry{
			Data:       make(map[string]int),
			UpdatedAt:  time.Now(),
			refreshing: true,
		}
		c.podCountCache.Store(cluster, entry)
	}

	// 执行刷新
	c.logger.Infof("Starting async pod count refresh for cluster: %s", cluster)
	podCounts := fetchFunc()

	// 更新缓存
	if cached, ok := c.podCountCache.Load(cluster); ok {
		entry := cached.(*CacheEntry)
		entry.mu.Lock()
		defer entry.mu.Unlock()

		if len(podCounts) > 0 {
			entry.Data = podCounts
			entry.UpdatedAt = time.Now()
			c.logger.Infof("Pod count cache async refreshed: cluster=%s, nodes=%d", cluster, len(podCounts))
		} else {
			c.logger.Warningf("Pod count cache async refresh returned empty: cluster=%s", cluster)
		}
		entry.refreshing = false
	}
}

// InvalidatePodCounts 清除指定集群的Pod数量缓存
func (c *K8sCache) InvalidatePodCounts(cluster string) {
	c.podCountCache.Delete(cluster)
	c.logger.Infof("Invalidated pod count cache for cluster: %s", cluster)
}

// Clear 清空所有缓存
func (c *K8sCache) Clear() {
	c.nodeListCache = &sync.Map{}
	c.nodeDetailCache = &sync.Map{}
	c.podCountCache = &sync.Map{}
	c.logger.Info("K8s cache cleared")
}
