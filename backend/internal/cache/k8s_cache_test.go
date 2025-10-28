package cache

import (
	"context"
	"testing"
	"time"

	"kube-node-manager/pkg/logger"
)

// NodeInfo 测试用的节点信息结构（避免import cycle）
type NodeInfo struct {
	Name   string
	Status string
}

// TestNewK8sCache 测试K8s缓存创建
func TestNewK8sCache(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)

	if cache == nil {
		t.Fatal("NewK8sCache returned nil")
	}

	if cache.listCacheTTL != 30*time.Second {
		t.Errorf("Expected listCacheTTL to be 30s, got %v", cache.listCacheTTL)
	}

	if cache.detailCacheTTL != 5*time.Minute {
		t.Errorf("Expected detailCacheTTL to be 5min, got %v", cache.detailCacheTTL)
	}
}

// TestGetNodeList_CacheHit 测试缓存命中
func TestGetNodeList_CacheHit(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"

	// 模拟节点数据
	mockNodes := []NodeInfo{
		{Name: "node1", Status: "Ready"},
		{Name: "node2", Status: "Ready"},
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNodes, nil
	}

	// 第一次调用 - 缓存未命中
	result1, err := cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected fetch to be called once, got %d", callCount)
	}

	// 第二次调用 - 缓存命中
	result2, err := cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected fetch to still be called once (cache hit), got %d", callCount)
	}

	// 验证数据一致性
	nodes1 := result1.([]NodeInfo)
	nodes2 := result2.([]NodeInfo)

	if len(nodes1) != len(nodes2) {
		t.Errorf("Cache returned different data: %d vs %d", len(nodes1), len(nodes2))
	}
}

// TestGetNodeList_ForceRefresh 测试强制刷新
func TestGetNodeList_ForceRefresh(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"

	mockNodes := []NodeInfo{
		{Name: "node1", Status: "Ready"},
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNodes, nil
	}

	// 第一次调用
	_, err := cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	// 强制刷新
	_, err = cache.GetNodeList(ctx, cluster, true, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList with force refresh failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected fetch to be called twice (force refresh), got %d", callCount)
	}
}

// TestGetNodeList_CacheExpiration 测试缓存过期
func TestGetNodeList_CacheExpiration(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)

	// 设置短TTL用于测试
	cache.listCacheTTL = 100 * time.Millisecond
	cache.staleThreshold = 200 * time.Millisecond

	ctx := context.Background()
	cluster := "test-cluster"

	mockNodes := []NodeInfo{
		{Name: "node1", Status: "Ready"},
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNodes, nil
	}

	// 第一次调用
	_, err := cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	// 等待缓存过期
	time.Sleep(250 * time.Millisecond)

	// 第二次调用 - 缓存过期，应该重新获取
	_, err = cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected fetch to be called twice (cache expired), got %d", callCount)
	}
}

// TestGetNodeDetail 测试节点详情缓存
func TestGetNodeDetail(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"
	nodeName := "test-node"

	mockNode := &NodeInfo{
		Name:   nodeName,
		Status: "Ready",
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNode, nil
	}

	// 第一次调用
	result1, err := cache.GetNodeDetail(ctx, cluster, nodeName, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeDetail failed: %v", err)
	}

	// 第二次调用 - 应该命中缓存
	result2, err := cache.GetNodeDetail(ctx, cluster, nodeName, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeDetail failed: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected fetch to be called once, got %d", callCount)
	}

	node1 := result1.(*NodeInfo)
	node2 := result2.(*NodeInfo)

	if node1.Name != node2.Name {
		t.Errorf("Cache returned different node names: %s vs %s", node1.Name, node2.Name)
	}
}

// TestInvalidateNode 测试缓存失效
func TestInvalidateNode(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"
	nodeName := "test-node"

	mockNode := &NodeInfo{
		Name:   nodeName,
		Status: "Ready",
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNode, nil
	}

	// 第一次调用 - 缓存数据
	_, err := cache.GetNodeDetail(ctx, cluster, nodeName, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeDetail failed: %v", err)
	}

	// 清除缓存
	cache.InvalidateNode(cluster, nodeName)

	// 第二次调用 - 缓存已失效，应该重新获取
	_, err = cache.GetNodeDetail(ctx, cluster, nodeName, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeDetail failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected fetch to be called twice after invalidation, got %d", callCount)
	}
}

// TestInvalidateCluster 测试集群缓存失效
func TestInvalidateCluster(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"

	mockNodes := []NodeInfo{
		{Name: "node1", Status: "Ready"},
	}

	callCount := 0
	fetchFunc := func() (interface{}, error) {
		callCount++
		return mockNodes, nil
	}

	// 缓存数据
	_, err := cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	// 清除集群缓存
	cache.InvalidateCluster(cluster)

	// 重新获取 - 应该重新调用fetch
	_, err = cache.GetNodeList(ctx, cluster, false, fetchFunc)
	if err != nil {
		t.Fatalf("GetNodeList failed: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected fetch to be called twice after cluster invalidation, got %d", callCount)
	}
}

// TestGetCacheStats 测试缓存统计
func TestGetCacheStats(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()

	// 缓存一些数据
	mockNodes := []NodeInfo{{Name: "node1"}}
	fetchFunc := func() (interface{}, error) { return mockNodes, nil }

	cache.GetNodeList(ctx, "cluster1", false, fetchFunc)
	cache.GetNodeList(ctx, "cluster2", false, fetchFunc)

	stats := cache.GetCacheStats()

	if stats["list_cache_count"].(int) != 2 {
		t.Errorf("Expected list_cache_count to be 2, got %v", stats["list_cache_count"])
	}

	if stats["list_cache_ttl"] == nil {
		t.Error("Expected list_cache_ttl to be set")
	}
}

// TestClear 测试清空缓存
func TestClear(t *testing.T) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()

	// 缓存一些数据
	mockNodes := []NodeInfo{{Name: "node1"}}
	fetchFunc := func() (interface{}, error) { return mockNodes, nil }

	cache.GetNodeList(ctx, "cluster1", false, fetchFunc)

	// 清空缓存
	cache.Clear()

	stats := cache.GetCacheStats()
	if stats["list_cache_count"].(int) != 0 {
		t.Errorf("Expected cache to be empty after Clear(), got count=%v", stats["list_cache_count"])
	}
}

// BenchmarkGetNodeList_CacheHit 缓存命中性能测试
func BenchmarkGetNodeList_CacheHit(b *testing.B) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()
	cluster := "test-cluster"

	mockNodes := make([]NodeInfo, 100)
	for i := 0; i < 100; i++ {
		mockNodes[i] = NodeInfo{
			Name:   "node" + string(rune(i)),
			Status: "Ready",
		}
	}

	fetchFunc := func() (interface{}, error) { return mockNodes, nil }

	// 预热缓存
	cache.GetNodeList(ctx, cluster, false, fetchFunc)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.GetNodeList(ctx, cluster, false, fetchFunc)
	}
}

// BenchmarkGetNodeList_CacheMiss 缓存未命中性能测试
func BenchmarkGetNodeList_CacheMiss(b *testing.B) {
	log := logger.NewLogger()
	cache := NewK8sCache(log)
	ctx := context.Background()

	mockNodes := make([]NodeInfo, 100)
	for i := 0; i < 100; i++ {
		mockNodes[i] = NodeInfo{
			Name:   "node" + string(rune(i)),
			Status: "Ready",
		}
	}

	fetchFunc := func() (interface{}, error) { return mockNodes, nil }

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cluster := "cluster-" + string(rune(i))
		_, _ = cache.GetNodeList(ctx, cluster, false, fetchFunc)
	}
}
