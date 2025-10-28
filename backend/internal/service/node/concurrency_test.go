package node

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewConcurrencyController 测试并发控制器创建
func TestNewConcurrencyController(t *testing.T) {
	ctrl := NewConcurrencyController()

	if ctrl == nil {
		t.Fatal("NewConcurrencyController returned nil")
	}

	// 验证操作配置
	if ctrl.operationConfigs == nil {
		t.Fatal("operationConfigs is nil")
	}

	// 验证cordon配置
	cordonConfig, exists := ctrl.operationConfigs["cordon"]
	if !exists {
		t.Fatal("cordon config not found")
	}

	if cordonConfig.BaseLimit != 15 {
		t.Errorf("Expected cordon BaseLimit to be 15, got %d", cordonConfig.BaseLimit)
	}
}

// TestCalculate_SmallCluster 测试小集群并发计算
func TestCalculate_SmallCluster(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 小集群（<50节点）
	concurrency := ctrl.Calculate("cordon", 30, 0)

	if concurrency != 15 {
		t.Errorf("Expected concurrency for small cluster to be 15, got %d", concurrency)
	}
}

// TestCalculate_MediumCluster 测试中等集群并发计算
func TestCalculate_MediumCluster(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 中等集群（50-200节点）
	concurrency := ctrl.Calculate("cordon", 100, 0)

	expected := int(float64(15) * 1.2) // 15 * 1.2 = 18
	if concurrency != expected {
		t.Errorf("Expected concurrency for medium cluster to be %d, got %d", expected, concurrency)
	}
}

// TestCalculate_LargeCluster 测试大集群并发计算
func TestCalculate_LargeCluster(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 大集群（>200节点）
	concurrency := ctrl.Calculate("cordon", 300, 0)

	// 15 * 1.5 = 22.5, 向下取整为22
	// 但不超过maxLimit 20
	expectedCapped := 20

	if concurrency != expectedCapped {
		t.Errorf("Expected concurrency for large cluster to be %d (capped), got %d", expectedCapped, concurrency)
	}
}

// TestCalculate_HighLatency 测试高延迟场景
func TestCalculate_HighLatency(t *testing.T) {
	ctrl := NewConcurrencyController()

	tests := []struct {
		name     string
		latency  time.Duration
		expected float64 // 期望的调整因子
	}{
		{"very high latency", 6 * time.Second, 0.4},
		{"high latency", 4 * time.Second, 0.6},
		{"medium latency", 2500 * time.Millisecond, 0.8},
		{"low latency", 400 * time.Millisecond, 1.2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			concurrency := ctrl.Calculate("cordon", 50, tt.latency)

			// 验证调整后的并发数在合理范围内
			if concurrency < 5 || concurrency > 20 {
				t.Errorf("Concurrency %d out of expected range [5, 20] for %s",
					concurrency, tt.name)
			}
		})
	}
}

// TestCalculate_DrainOperation 测试Drain操作（重量级）
func TestCalculate_DrainOperation(t *testing.T) {
	ctrl := NewConcurrencyController()

	// Drain操作应该有较低的并发数
	concurrency := ctrl.Calculate("drain", 100, 0)

	// Drain基础并发数是5，中等集群调整为6
	expected := int(float64(5) * 1.2)
	if concurrency != expected {
		t.Errorf("Expected drain concurrency to be %d, got %d", expected, concurrency)
	}
}

// TestCalculate_UnknownOperation 测试未知操作类型
func TestCalculate_UnknownOperation(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 未知操作应该使用默认配置
	concurrency := ctrl.Calculate("unknown", 100, 0)

	// 默认配置应该返回合理的值
	if concurrency < 5 || concurrency > 20 {
		t.Errorf("Concurrency for unknown operation out of range: %d", concurrency)
	}
}

// TestRecordLatency 测试延迟记录
func TestRecordLatency(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 记录一些延迟
	latencies := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		300 * time.Millisecond,
	}

	for _, latency := range latencies {
		ctrl.RecordLatency(latency)
	}

	// 获取平均延迟
	avgLatency := ctrl.GetAverageLatency()

	// 验证平均值
	expected := 200 * time.Millisecond
	if avgLatency != expected {
		t.Errorf("Expected average latency to be %v, got %v", expected, avgLatency)
	}
}

// TestRecordLatency_MaxSamples 测试最大样本限制
func TestRecordLatency_MaxSamples(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 记录超过最大样本数的延迟
	for i := 0; i < 150; i++ {
		ctrl.RecordLatency(time.Duration(i) * time.Millisecond)
	}

	stats := ctrl.GetStats()
	sampleCount := stats["sample_count"].(int)

	// 应该只保留最近100个样本
	if sampleCount != 100 {
		t.Errorf("Expected sample count to be 100, got %d", sampleCount)
	}
}

// TestResetLatencies 测试重置延迟统计
func TestResetLatencies(t *testing.T) {
	ctrl := NewConcurrencyController()

	// 记录一些延迟
	ctrl.RecordLatency(100 * time.Millisecond)
	ctrl.RecordLatency(200 * time.Millisecond)

	// 重置
	ctrl.ResetLatencies()

	// 验证已清空
	avgLatency := ctrl.GetAverageLatency()
	if avgLatency != 0 {
		t.Errorf("Expected average latency to be 0 after reset, got %v", avgLatency)
	}

	stats := ctrl.GetStats()
	if stats["sample_count"].(int) != 0 {
		t.Errorf("Expected sample count to be 0 after reset, got %v", stats["sample_count"])
	}
}

// TestDefaultRetryConfig 测试默认重试配置
func TestDefaultRetryConfig(t *testing.T) {
	config := DefaultRetryConfig()

	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries to be 3, got %d", config.MaxRetries)
	}

	if config.InitialDelay != 100*time.Millisecond {
		t.Errorf("Expected InitialDelay to be 100ms, got %v", config.InitialDelay)
	}

	if config.BackoffFactor != 2.0 {
		t.Errorf("Expected BackoffFactor to be 2.0, got %f", config.BackoffFactor)
	}
}

// TestShouldRetry 测试重试判断
func TestShouldRetry(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		name        string
		err         error
		attempt     int
		shouldRetry bool
	}{
		{"timeout error", errors.New("connection timeout"), 0, true},
		{"connection refused", errors.New("connection refused"), 1, true},
		{"conflict error", errors.New("object has been modified"), 2, true},
		{"max retries reached", errors.New("timeout"), 3, false},
		{"non-retryable error", errors.New("invalid input"), 0, false},
		{"nil error", nil, 0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := config.ShouldRetry(tt.err, tt.attempt)
			if result != tt.shouldRetry {
				t.Errorf("Expected ShouldRetry to be %v, got %v", tt.shouldRetry, result)
			}
		})
	}
}

// TestGetDelay 测试重试延迟计算
func TestGetDelay(t *testing.T) {
	config := DefaultRetryConfig()

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 100 * time.Millisecond}, // 100ms * 2^0 = 100ms
		{1, 200 * time.Millisecond}, // 100ms * 2^1 = 200ms
		{2, 400 * time.Millisecond}, // 100ms * 2^2 = 400ms
		{3, 800 * time.Millisecond}, // 100ms * 2^3 = 800ms
		{10, 5 * time.Second},       // 超过最大延迟，返回5s
	}

	for _, tt := range tests {
		t.Run("attempt_"+string(rune(tt.attempt)), func(t *testing.T) {
			delay := config.GetDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("Expected delay for attempt %d to be %v, got %v",
					tt.attempt, tt.expected, delay)
			}
		})
	}
}

// TestRetryWithBackoff_Success 测试重试成功
func TestRetryWithBackoff_Success(t *testing.T) {
	config := DefaultRetryConfig()
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 2 {
			return errors.New("temporary failure")
		}
		return nil
	}

	err := RetryWithBackoff(ctx, config, operation)

	if err != nil {
		t.Errorf("Expected retry to succeed, got error: %v", err)
	}

	if callCount != 2 {
		t.Errorf("Expected operation to be called 2 times, got %d", callCount)
	}
}

// TestRetryWithBackoff_MaxRetriesExceeded 测试达到最大重试次数
func TestRetryWithBackoff_MaxRetriesExceeded(t *testing.T) {
	config := DefaultRetryConfig()
	config.MaxRetries = 2
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("timeout")
	}

	err := RetryWithBackoff(ctx, config, operation)

	if err == nil {
		t.Error("Expected retry to fail after max retries")
	}

	// MaxRetries=2 means 3 total attempts (initial + 2 retries)
	if callCount != 3 {
		t.Errorf("Expected operation to be called 3 times, got %d", callCount)
	}
}

// TestRetryWithBackoff_NonRetryableError 测试不可重试错误
func TestRetryWithBackoff_NonRetryableError(t *testing.T) {
	config := DefaultRetryConfig()
	ctx := context.Background()

	callCount := 0
	operation := func() error {
		callCount++
		return errors.New("invalid input")
	}

	err := RetryWithBackoff(ctx, config, operation)

	if err == nil {
		t.Error("Expected operation to fail")
	}

	// 不可重试错误应该只调用一次
	if callCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", callCount)
	}
}

// TestRetryWithBackoff_ContextCanceled 测试上下文取消
func TestRetryWithBackoff_ContextCanceled(t *testing.T) {
	config := DefaultRetryConfig()
	ctx, cancel := context.WithCancel(context.Background())

	callCount := 0
	operation := func() error {
		callCount++
		if callCount == 2 {
			cancel() // 第二次调用时取消上下文
		}
		return errors.New("timeout")
	}

	err := RetryWithBackoff(ctx, config, operation)

	if err != context.Canceled {
		t.Errorf("Expected context.Canceled error, got %v", err)
	}
}

// BenchmarkCalculate 并发计算性能测试
func BenchmarkCalculate(b *testing.B) {
	ctrl := NewConcurrencyController()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrl.Calculate("cordon", 100, 500*time.Millisecond)
	}
}

// BenchmarkRecordLatency 延迟记录性能测试
func BenchmarkRecordLatency(b *testing.B) {
	ctrl := NewConcurrencyController()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctrl.RecordLatency(100 * time.Millisecond)
	}
}
