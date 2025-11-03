package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"math"
	"time"

	"gorm.io/gorm"
)

// EstimationService 任务执行预估服务
type EstimationService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewEstimationService 创建预估服务实例
func NewEstimationService(db *gorm.DB, logger *logger.Logger) *EstimationService {
	return &EstimationService{
		db:     db,
		logger: logger,
	}
}

// TaskEstimation 任务执行预估结果
type TaskEstimation struct {
	MinDuration      int     `json:"min_duration"`       // 最短时长（秒）
	MaxDuration      int     `json:"max_duration"`       // 最长时长（秒）
	AvgDuration      float64 `json:"avg_duration"`       // 平均时长（秒）
	MedianDuration   int     `json:"median_duration"`    // 中位数时长（秒）
	SuccessRate      float64 `json:"success_rate"`       // 成功率（0-100）
	SampleSize       int     `json:"sample_size"`        // 样本数量
	LastExecutedAt   *time.Time `json:"last_executed_at"` // 最后执行时间
	EstimatedRange   string  `json:"estimated_range"`    // 预估范围描述
	Confidence       string  `json:"confidence"`         // 置信度（high/medium/low）
}

// EstimateByTemplate 基于模板预估任务执行时间
func (s *EstimationService) EstimateByTemplate(templateID uint) (*TaskEstimation, error) {
	var tasks []model.AnsibleTask
	
	// 查询最近30天使用该模板的成功任务
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := s.db.Where("template_id = ? AND status IN (?, ?) AND created_at > ?", 
		templateID, model.AnsibleTaskStatusSuccess, model.AnsibleTaskStatusFailed, thirtyDaysAgo).
		Order("created_at DESC").
		Limit(100).
		Find(&tasks).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no historical data found")
	}
	
	return s.calculateEstimation(tasks)
}

// EstimateByInventory 基于清单预估任务执行时间
func (s *EstimationService) EstimateByInventory(inventoryID uint) (*TaskEstimation, error) {
	var tasks []model.AnsibleTask
	
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := s.db.Where("inventory_id = ? AND status IN (?, ?) AND created_at > ?", 
		inventoryID, model.AnsibleTaskStatusSuccess, model.AnsibleTaskStatusFailed, thirtyDaysAgo).
		Order("created_at DESC").
		Limit(100).
		Find(&tasks).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no historical data found")
	}
	
	return s.calculateEstimation(tasks)
}

// EstimateByTemplateAndInventory 基于模板和清单组合预估
func (s *EstimationService) EstimateByTemplateAndInventory(templateID, inventoryID uint) (*TaskEstimation, error) {
	var tasks []model.AnsibleTask
	
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	err := s.db.Where("template_id = ? AND inventory_id = ? AND status IN (?, ?) AND created_at > ?", 
		templateID, inventoryID, model.AnsibleTaskStatusSuccess, model.AnsibleTaskStatusFailed, thirtyDaysAgo).
		Order("created_at DESC").
		Limit(100).
		Find(&tasks).Error
	
	if err != nil {
		return nil, fmt.Errorf("failed to query tasks: %w", err)
	}
	
	// 如果组合查询没有数据，尝试只用模板查询
	if len(tasks) == 0 {
		s.logger.Infof("No data for template+inventory, falling back to template only")
		return s.EstimateByTemplate(templateID)
	}
	
	return s.calculateEstimation(tasks)
}

// calculateEstimation 计算预估结果
func (s *EstimationService) calculateEstimation(tasks []model.AnsibleTask) (*TaskEstimation, error) {
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no tasks to estimate")
	}
	
	// 提取所有时长
	var durations []int
	var successCount int
	var lastExecutedAt *time.Time
	
	for _, task := range tasks {
		if task.Duration > 0 {
			durations = append(durations, task.Duration)
		}
		if task.Status == model.AnsibleTaskStatusSuccess {
			successCount++
		}
		if lastExecutedAt == nil || task.CreatedAt.After(*lastExecutedAt) {
			lastExecutedAt = &task.CreatedAt
		}
	}
	
	if len(durations) == 0 {
		return nil, fmt.Errorf("no valid duration data")
	}
	
	// 计算统计数据
	minDuration, maxDuration := minMax(durations)
	avgDuration := average(durations)
	medianDuration := median(durations)
	successRate := float64(successCount) / float64(len(tasks)) * 100
	
	// 生成预估范围描述
	estimatedRange := s.formatDurationRange(int(avgDuration * 0.8), int(avgDuration * 1.2))
	
	// 计算置信度
	confidence := s.calculateConfidence(len(durations), successRate)
	
	return &TaskEstimation{
		MinDuration:    minDuration,
		MaxDuration:    maxDuration,
		AvgDuration:    avgDuration,
		MedianDuration: medianDuration,
		SuccessRate:    successRate,
		SampleSize:     len(tasks),
		LastExecutedAt: lastExecutedAt,
		EstimatedRange: estimatedRange,
		Confidence:     confidence,
	}, nil
}

// minMax 计算最小值和最大值
func minMax(numbers []int) (int, int) {
	if len(numbers) == 0 {
		return 0, 0
	}
	min, max := numbers[0], numbers[0]
	for _, n := range numbers {
		if n < min {
			min = n
		}
		if n > max {
			max = n
		}
	}
	return min, max
}

// average 计算平均值
func average(numbers []int) float64 {
	if len(numbers) == 0 {
		return 0
	}
	sum := 0
	for _, n := range numbers {
		sum += n
	}
	return float64(sum) / float64(len(numbers))
}

// median 计算中位数
func median(numbers []int) int {
	if len(numbers) == 0 {
		return 0
	}
	
	// 简单排序（冒泡排序）
	sorted := make([]int, len(numbers))
	copy(sorted, numbers)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i] > sorted[j] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	
	// 返回中位数
	mid := len(sorted) / 2
	if len(sorted)%2 == 0 {
		return (sorted[mid-1] + sorted[mid]) / 2
	}
	return sorted[mid]
}

// formatDurationRange 格式化时长范围
func (s *EstimationService) formatDurationRange(minSec, maxSec int) string {
	formatDuration := func(sec int) string {
		if sec < 60 {
			return fmt.Sprintf("%d秒", sec)
		} else if sec < 3600 {
			return fmt.Sprintf("%d分钟", int(math.Round(float64(sec)/60)))
		} else {
			hours := sec / 3600
			minutes := (sec % 3600) / 60
			if minutes > 0 {
				return fmt.Sprintf("%d小时%d分钟", hours, minutes)
			}
			return fmt.Sprintf("%d小时", hours)
		}
	}
	
	return fmt.Sprintf("%s - %s", formatDuration(minSec), formatDuration(maxSec))
}

// calculateConfidence 计算置信度
func (s *EstimationService) calculateConfidence(sampleSize int, successRate float64) string {
	// 基于样本量和成功率判断置信度
	if sampleSize >= 20 && successRate >= 80 {
		return "high"
	} else if sampleSize >= 10 && successRate >= 60 {
		return "medium"
	}
	return "low"
}

// GetEstimationSummary 获取预估摘要（用于UI显示）
func (s *EstimationService) GetEstimationSummary(estimation *TaskEstimation) string {
	if estimation == nil {
		return "无历史数据"
	}
	
	confidenceText := map[string]string{
		"high":   "高",
		"medium": "中",
		"low":    "低",
	}
	
	return fmt.Sprintf("预估: %s (基于%d次历史执行, 成功率%.1f%%, 置信度: %s)", 
		estimation.EstimatedRange, 
		estimation.SampleSize, 
		estimation.SuccessRate,
		confidenceText[estimation.Confidence])
}

