package feishu

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheItem represents a cached item
type CacheItem struct {
	Value      interface{}
	ExpireTime time.Time
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
	items map[string]*CacheItem
	mu    sync.RWMutex
}

// NewMemoryCache creates a new memory cache
func NewMemoryCache() *MemoryCache {
	cache := &MemoryCache{
		items: make(map[string]*CacheItem),
	}

	// Start cleanup goroutine
	go cache.cleanup()

	return cache
}

// Set sets a cache item with expiration
func (c *MemoryCache) Set(key string, value interface{}, duration time.Duration) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = &CacheItem{
		Value:      value,
		ExpireTime: time.Now().Add(duration),
	}

	return nil
}

// Get gets a cache item
func (c *MemoryCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	item, exists := c.items[key]
	if !exists {
		return nil, false
	}

	// Check expiration
	if time.Now().After(item.ExpireTime) {
		return nil, false
	}

	return item.Value, true
}

// Delete deletes a cache item
func (c *MemoryCache) Delete(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	delete(c.items, key)
}

// Clear clears all cache items
func (c *MemoryCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items = make(map[string]*CacheItem)
}

// cleanup periodically removes expired items
func (c *MemoryCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, item := range c.items {
			if now.After(item.ExpireTime) {
				delete(c.items, key)
			}
		}
		c.mu.Unlock()
	}
}

// CachedService wraps the Service with caching capabilities
type CachedService struct {
	service *Service
	cache   *MemoryCache
}

// NewCachedService creates a new cached service
func NewCachedService(service *Service) *CachedService {
	return &CachedService{
		service: service,
		cache:   NewMemoryCache(),
	}
}

// GetClusterListCached gets cluster list with cache
func (cs *CachedService) GetClusterListCached(userID uint) (interface{}, error) {
	cacheKey := fmt.Sprintf("clusters:user:%d", userID)

	// Try cache first
	if cached, ok := cs.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Cache miss, fetch from service
	clusters, err := cs.service.clusterService.List(nil, userID)
	if err != nil {
		return nil, err
	}

	// Cache for 5 minutes
	_ = cs.cache.Set(cacheKey, clusters, 5*time.Minute)

	return clusters, nil
}

// GetNodeListCached gets node list with cache
func (cs *CachedService) GetNodeListCached(clusterName string, userID uint) (interface{}, error) {
	cacheKey := fmt.Sprintf("nodes:cluster:%s:user:%d", clusterName, userID)

	// Try cache first
	if cached, ok := cs.cache.Get(cacheKey); ok {
		return cached, nil
	}

	// Cache miss, fetch from service
	nodes, err := cs.service.nodeService.List(map[string]interface{}{
		"ClusterName": clusterName,
	}, userID)
	if err != nil {
		return nil, err
	}

	// Cache for 2 minutes
	_ = cs.cache.Set(cacheKey, nodes, 2*time.Minute)

	return nodes, nil
}

// InvalidateClusterCache invalidates cluster cache
func (cs *CachedService) InvalidateClusterCache(userID uint) {
	cacheKey := fmt.Sprintf("clusters:user:%d", userID)
	cs.cache.Delete(cacheKey)
}

// InvalidateNodeCache invalidates node cache
func (cs *CachedService) InvalidateNodeCache(clusterName string, userID uint) {
	cacheKey := fmt.Sprintf("nodes:cluster:%s:user:%d", clusterName, userID)
	cs.cache.Delete(cacheKey)
}

// SessionCache manages user session cache
type SessionCache struct {
	cache *MemoryCache
}

// NewSessionCache creates a new session cache
func NewSessionCache() *SessionCache {
	return &SessionCache{
		cache: NewMemoryCache(),
	}
}

// GetSession gets user session from cache
func (sc *SessionCache) GetSession(feishuUserID string) (interface{}, bool) {
	cacheKey := fmt.Sprintf("session:%s", feishuUserID)
	return sc.cache.Get(cacheKey)
}

// SetSession sets user session in cache
func (sc *SessionCache) SetSession(feishuUserID string, session interface{}) {
	cacheKey := fmt.Sprintf("session:%s", feishuUserID)
	// Cache for 30 minutes
	_ = sc.cache.Set(cacheKey, session, 30*time.Minute)
}

// InvalidateSession invalidates user session
func (sc *SessionCache) InvalidateSession(feishuUserID string) {
	cacheKey := fmt.Sprintf("session:%s", feishuUserID)
	sc.cache.Delete(cacheKey)
}

// CommandCache manages command result cache
type CommandCache struct {
	cache *MemoryCache
}

// NewCommandCache creates a new command cache
func NewCommandCache() *CommandCache {
	return &CommandCache{
		cache: NewMemoryCache(),
	}
}

// GetCommandResult gets cached command result
func (cc *CommandCache) GetCommandResult(commandHash string) (string, bool) {
	if cached, ok := cc.cache.Get(commandHash); ok {
		if result, ok := cached.(string); ok {
			return result, true
		}
	}
	return "", false
}

// SetCommandResult caches command result
func (cc *CommandCache) SetCommandResult(commandHash, result string, duration time.Duration) {
	_ = cc.cache.Set(commandHash, result, duration)
}

// GenerateCommandHash generates a hash for command caching
func GenerateCommandHash(command, userID string) string {
	data := fmt.Sprintf("%s:%s:%d", command, userID, time.Now().Unix()/60) // Per minute
	return fmt.Sprintf("cmd:%x", []byte(data))
}

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	requests map[string][]time.Time
	mu       sync.RWMutex
	limit    int           // requests per window
	window   time.Duration // time window
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	rl := &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}

	// Start cleanup goroutine
	go rl.cleanup()

	return rl
}

// Allow checks if a request is allowed
func (rl *RateLimiter) Allow(key string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	windowStart := now.Add(-rl.window)

	// Get existing requests
	requests, exists := rl.requests[key]
	if !exists {
		rl.requests[key] = []time.Time{now}
		return true
	}

	// Filter out old requests
	validRequests := []time.Time{}
	for _, t := range requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// Check limit
	if len(validRequests) >= rl.limit {
		rl.requests[key] = validRequests
		return false
	}

	// Add new request
	validRequests = append(validRequests, now)
	rl.requests[key] = validRequests
	return true
}

// cleanup periodically removes old entries
func (rl *RateLimiter) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		windowStart := now.Add(-rl.window)

		for key, requests := range rl.requests {
			validRequests := []time.Time{}
			for _, t := range requests {
				if t.After(windowStart) {
					validRequests = append(validRequests, t)
				}
			}

			if len(validRequests) == 0 {
				delete(rl.requests, key)
			} else {
				rl.requests[key] = validRequests
			}
		}
		rl.mu.Unlock()
	}
}

// AsyncOperationManager manages async operations
type AsyncOperationManager struct {
	operations map[string]*AsyncOperation
	mu         sync.RWMutex
}

// AsyncOperation represents an async operation
type AsyncOperation struct {
	ID         string
	Status     string // pending, running, completed, failed
	Progress   int    // 0-100
	Result     interface{}
	Error      error
	StartTime  time.Time
	UpdateTime time.Time
}

// NewAsyncOperationManager creates a new async operation manager
func NewAsyncOperationManager() *AsyncOperationManager {
	return &AsyncOperationManager{
		operations: make(map[string]*AsyncOperation),
	}
}

// CreateOperation creates a new async operation
func (aom *AsyncOperationManager) CreateOperation(id string) *AsyncOperation {
	aom.mu.Lock()
	defer aom.mu.Unlock()

	op := &AsyncOperation{
		ID:         id,
		Status:     "pending",
		Progress:   0,
		StartTime:  time.Now(),
		UpdateTime: time.Now(),
	}

	aom.operations[id] = op
	return op
}

// GetOperation gets an async operation
func (aom *AsyncOperationManager) GetOperation(id string) (*AsyncOperation, bool) {
	aom.mu.RLock()
	defer aom.mu.RUnlock()

	op, exists := aom.operations[id]
	return op, exists
}

// UpdateOperation updates an async operation
func (aom *AsyncOperationManager) UpdateOperation(id string, status string, progress int) {
	aom.mu.Lock()
	defer aom.mu.Unlock()

	if op, exists := aom.operations[id]; exists {
		op.Status = status
		op.Progress = progress
		op.UpdateTime = time.Now()
	}
}

// CompleteOperation completes an async operation
func (aom *AsyncOperationManager) CompleteOperation(id string, result interface{}, err error) {
	aom.mu.Lock()
	defer aom.mu.Unlock()

	if op, exists := aom.operations[id]; exists {
		if err != nil {
			op.Status = "failed"
			op.Error = err
		} else {
			op.Status = "completed"
			op.Result = result
		}
		op.Progress = 100
		op.UpdateTime = time.Now()
	}
}

// SerializeForCache serializes data for caching
func SerializeForCache(data interface{}) (string, error) {
	bytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// DeserializeFromCache deserializes data from cache
func DeserializeFromCache(cached string, target interface{}) error {
	return json.Unmarshal([]byte(cached), target)
}
