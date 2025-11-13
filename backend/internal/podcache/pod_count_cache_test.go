package podcache

import (
	"testing"
	"time"

	"kube-node-manager/pkg/logger"
)

// TestMarkSynced 测试 MarkSynced 方法
func TestMarkSynced(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"

	// 初始状态：缓存未就绪
	if cache.IsReady(clusterName) {
		t.Error("Cache should not be ready before MarkSynced is called")
	}

	// 调用 MarkSynced
	cache.MarkSynced(clusterName)

	// 缓存应该已就绪
	if !cache.IsReady(clusterName) {
		t.Error("Cache should be ready after MarkSynced is called")
	}
}

// TestShouldLogNotReady 测试日志限速机制
func TestShouldLogNotReady(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"

	// 第一次调用应该返回 true
	if !cache.ShouldLogNotReady(clusterName) {
		t.Error("First call should return true")
	}

	// 短时间内再次调用应该返回 false（限速）
	if cache.ShouldLogNotReady(clusterName) {
		t.Error("Second call within 60s should return false")
	}

	// 等待 1 秒后再次调用仍然应该返回 false
	time.Sleep(1 * time.Second)
	if cache.ShouldLogNotReady(clusterName) {
		t.Error("Call within 60s should return false")
	}
}

// TestIsReadyWithData 测试带数据的 IsReady 检查（兼容性）
func TestIsReadyWithData(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"

	// 初始状态：缓存未就绪
	if cache.IsReady(clusterName) {
		t.Error("Cache should not be ready when empty")
	}

	// 添加一个节点的 Pod 计数
	cache.incrementPodCount(clusterName, "node-1")

	// 即使没有调用 MarkSynced，有数据时也应该返回 ready（兼容性）
	if !cache.IsReady(clusterName) {
		t.Error("Cache should be ready when it has data (compatibility fallback)")
	}
}

// TestPodCountOperations 测试 Pod 计数的增减操作
func TestPodCountOperations(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"
	nodeName := "node-1"

	// 初始计数应该是 0
	if count := cache.GetNodePodCount(clusterName, nodeName); count != 0 {
		t.Errorf("Initial count should be 0, got %d", count)
	}

	// 递增计数
	cache.incrementPodCount(clusterName, nodeName)
	if count := cache.GetNodePodCount(clusterName, nodeName); count != 1 {
		t.Errorf("Count should be 1 after increment, got %d", count)
	}

	// 再次递增
	cache.incrementPodCount(clusterName, nodeName)
	if count := cache.GetNodePodCount(clusterName, nodeName); count != 2 {
		t.Errorf("Count should be 2 after second increment, got %d", count)
	}

	// 递减计数
	cache.decrementPodCount(clusterName, nodeName)
	if count := cache.GetNodePodCount(clusterName, nodeName); count != 1 {
		t.Errorf("Count should be 1 after decrement, got %d", count)
	}

	// 递减到 0
	cache.decrementPodCount(clusterName, nodeName)
	if count := cache.GetNodePodCount(clusterName, nodeName); count != 0 {
		t.Errorf("Count should be 0 after second decrement, got %d", count)
	}
}

// TestGetAllNodePodCounts 测试批量获取 Pod 计数
func TestGetAllNodePodCounts(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"

	// 为多个节点设置计数
	cache.incrementPodCount(clusterName, "node-1")
	cache.incrementPodCount(clusterName, "node-1")
	cache.incrementPodCount(clusterName, "node-2")

	// 获取所有节点的计数
	counts := cache.GetAllNodePodCounts(clusterName)

	if len(counts) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(counts))
	}

	if counts["node-1"] != 2 {
		t.Errorf("Expected node-1 count to be 2, got %d", counts["node-1"])
	}

	if counts["node-2"] != 1 {
		t.Errorf("Expected node-2 count to be 1, got %d", counts["node-2"])
	}
}

// TestInvalidateCluster 测试集群缓存清除
func TestInvalidateCluster(t *testing.T) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"

	// 添加一些数据
	cache.incrementPodCount(clusterName, "node-1")
	cache.MarkSynced(clusterName)

	// 验证缓存已就绪且有数据
	if !cache.IsReady(clusterName) {
		t.Error("Cache should be ready")
	}
	if cache.GetNodePodCount(clusterName, "node-1") != 1 {
		t.Error("Node should have 1 pod")
	}

	// 清除集群缓存
	cache.InvalidateCluster(clusterName)

	// 验证缓存已清除
	if cache.GetNodePodCount(clusterName, "node-1") != 0 {
		t.Error("Count should be 0 after invalidation")
	}
	
	// 注意：InvalidateCluster 不会清除 clusterSynced 标记
	// 这是有意为之，因为我们只想清除数据，不想影响同步状态
}

// BenchmarkIncrementPodCount 性能测试：Pod 计数递增
func BenchmarkIncrementPodCount(b *testing.B) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"
	nodeName := "node-1"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.incrementPodCount(clusterName, nodeName)
	}
}

// BenchmarkGetNodePodCount 性能测试：获取 Pod 计数
func BenchmarkGetNodePodCount(b *testing.B) {
	log := logger.NewLogger()
	cache := NewPodCountCache(log)

	clusterName := "test-cluster"
	nodeName := "node-1"
	cache.incrementPodCount(clusterName, nodeName)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.GetNodePodCount(clusterName, nodeName)
	}
}

