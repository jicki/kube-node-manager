package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"strings"
	"time"

	"gorm.io/gorm"
)

// VisualizationService 处理任务执行可视化
type VisualizationService struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewVisualizationService 创建 VisualizationService 实例
func NewVisualizationService(db *gorm.DB, logger *logger.Logger) *VisualizationService {
	return &VisualizationService{
		db:     db,
		logger: logger,
	}
}

// GetTaskVisualization 获取任务执行可视化数据
func (s *VisualizationService) GetTaskVisualization(taskID uint) (*model.TaskExecutionVisualization, error) {
	s.logger.Infof("Fetching visualization data for task %d", taskID)
	
	var task model.AnsibleTask
	if err := s.db.Preload("Inventory").Preload("PreflightChecks").First(&task, taskID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("task not found")
		}
		return nil, fmt.Errorf("failed to get task: %w", err)
	}

	s.logger.Infof("Task %d loaded: name=%s, status=%s, started=%v, finished=%v", 
		task.ID, task.Name, task.Status, task.StartedAt, task.FinishedAt)

	viz := &model.TaskExecutionVisualization{
		TaskID:   task.ID,
		TaskName: task.Name,
		Status:   string(task.Status),
	}

	// 如果有执行时间线，直接使用
	if task.ExecutionTimeline != nil && len(*task.ExecutionTimeline) > 0 {
		s.logger.Infof("Using stored execution timeline with %d events", len(*task.ExecutionTimeline))
		viz.Timeline = *task.ExecutionTimeline
	} else {
		// 否则，根据任务状态生成基本时间线
		s.logger.Infof("Generating basic timeline for task %d", taskID)
		viz.Timeline = s.generateBasicTimeline(&task)
	}

	// 计算总耗时
	if len(viz.Timeline) > 0 {
		first := viz.Timeline[0]
		last := viz.Timeline[len(viz.Timeline)-1]
		viz.TotalDuration = int(last.Timestamp.Sub(first.Timestamp).Milliseconds())
		s.logger.Infof("Total duration calculated from timeline: %dms", viz.TotalDuration)
	} else if task.Duration > 0 {
		viz.TotalDuration = task.Duration * 1000 // 转换为毫秒
		s.logger.Infof("Total duration from task.Duration: %dms", viz.TotalDuration)
	}

	// 计算各阶段耗时分布
	viz.PhaseDistribution = s.calculatePhaseDistribution(viz.Timeline)
	if viz.PhaseDistribution != nil {
		s.logger.Infof("Phase distribution: %v", viz.PhaseDistribution)
	} else {
		s.logger.Warningf("No phase distribution data available for task %d", taskID)
	}

	// 获取主机执行状态
	viz.HostStatuses = s.extractHostStatuses(&task)
	s.logger.Infof("Extracted %d host statuses", len(viz.HostStatuses))

	return viz, nil
}

// generateBasicTimeline 为没有详细时间线的任务生成基本时间线
func (s *VisualizationService) generateBasicTimeline(task *model.AnsibleTask) model.TaskExecutionTimeline {
	timeline := make(model.TaskExecutionTimeline, 0)

	// 入队事件
	if task.QueuedAt != nil {
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhaseQueued,
			Message:   "任务已入队",
			Timestamp: *task.QueuedAt,
		})
	} else {
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhaseQueued,
			Message:   "任务已创建",
			Timestamp: task.CreatedAt,
		})
	}

	// 前置检查事件
	if task.PreflightChecks != nil {
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhasePreflightCheck,
			Message:   fmt.Sprintf("前置检查: %s", task.PreflightChecks.Status),
			Timestamp: task.PreflightChecks.CheckedAt,
			Duration:  task.PreflightChecks.Duration,
		})
	}

	// 执行开始事件（仅在任务已完成时添加）
	if task.StartedAt != nil && task.FinishedAt != nil {
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhaseExecuting,
			Message:   "任务开始执行",
			Timestamp: *task.StartedAt,
			HostCount: task.HostsTotal,
		})
	}

	// 完成事件
	if task.FinishedAt != nil {
		var phase model.ExecutionPhase
		var message string

		if task.IsTimedOut {
			phase = model.PhaseTimeout
			message = "任务执行超时"
		} else if task.Status == model.AnsibleTaskStatusCancelled {
			phase = model.PhaseCancelled
			message = "任务已取消"
		} else if task.Status == model.AnsibleTaskStatusFailed {
			phase = model.PhaseFailed
			message = "任务执行失败"
		} else {
			phase = model.PhaseCompleted
			message = "任务执行成功"
		}

		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:        phase,
			Message:      message,
			Timestamp:    *task.FinishedAt,
			HostCount:    task.HostsTotal,
			SuccessCount: task.HostsOk,
			FailCount:    task.HostsFailed,
		})
	} else if task.StartedAt != nil {
		// 任务还在执行中
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhaseExecuting,
			Message:   "任务执行中",
			Timestamp: *task.StartedAt,
			HostCount: task.HostsTotal,
		})
	}

	// 计算每个事件的耗时
	for i := 0; i < len(timeline); i++ {
		if timeline[i].Duration == 0 {
			// 如果不是最后一个事件，用下一个事件的时间戳计算
			if i < len(timeline)-1 {
				timeline[i].Duration = int(timeline[i+1].Timestamp.Sub(timeline[i].Timestamp).Milliseconds())
			} else if task.FinishedAt != nil && task.StartedAt != nil {
				// 最后一个事件，如果是完成事件，计算从开始到完成的耗时
				timeline[i].Duration = int(task.FinishedAt.Sub(*task.StartedAt).Milliseconds())
			}
		}
	}

	s.logger.Infof("Generated basic timeline for task %d with %d events", task.ID, len(timeline))
	for i, event := range timeline {
		s.logger.Debugf("Event %d: phase=%s, duration=%dms, timestamp=%v", 
			i, event.Phase, event.Duration, event.Timestamp)
	}

	return timeline
}

// calculatePhaseDistribution 计算各阶段耗时分布
func (s *VisualizationService) calculatePhaseDistribution(timeline model.TaskExecutionTimeline) map[string]int {
	distribution := make(map[string]int)
	
	for _, event := range timeline {
		// 只统计有耗时的事件，跳过 Duration 为 0 的事件
		if event.Duration > 0 {
			phase := string(event.Phase)
			distribution[phase] += event.Duration
		}
	}
	
	// 如果没有任何耗时数据，返回 nil 而不是空 map
	if len(distribution) == 0 {
		s.logger.Infof("No phase distribution data for timeline with %d events", len(timeline))
		return nil
	}
	
	s.logger.Debugf("Phase distribution calculated: %v", distribution)
	return distribution
}

// extractHostStatuses 从任务日志中提取主机执行状态
func (s *VisualizationService) extractHostStatuses(task *model.AnsibleTask) []model.HostExecutionStatus {
	statuses := make([]model.HostExecutionStatus, 0)
	
	// 从任务中获取主机列表
	if task.Inventory == nil {
		return statuses
	}

	// 查询任务日志以提取主机状态信息
	var logs []model.AnsibleLog
	if err := s.db.Where("task_id = ?", task.ID).
		Order("created_at ASC").
		Find(&logs).Error; err != nil {
		s.logger.Errorf("Failed to get task logs: %v", err)
		return statuses
	}

	// 解析日志，提取主机状态
	hostMap := make(map[string]*model.HostExecutionStatus)
	
	for _, log := range logs {
		// 尝试从日志中解析主机名和状态
		// Ansible 输出格式通常是: "ok: [hostname]" 或 "failed: [hostname]"
		content := log.Content
		
		// 检测 Ansible 的标准输出格式
		if strings.Contains(content, "ok: [") {
			hostname := extractHostname(content, "ok")
			if hostname != "" {
				if _, exists := hostMap[hostname]; !exists {
					hostMap[hostname] = &model.HostExecutionStatus{
						HostName:  hostname,
						Status:    "ok",
						StartTime: log.CreatedAt,
					}
				}
				hostMap[hostname].TasksOk++
				hostMap[hostname].EndTime = log.CreatedAt
			}
		} else if strings.Contains(content, "failed: [") {
			hostname := extractHostname(content, "failed")
			if hostname != "" {
				if _, exists := hostMap[hostname]; !exists {
					hostMap[hostname] = &model.HostExecutionStatus{
						HostName:  hostname,
						Status:    "failed",
						StartTime: log.CreatedAt,
					}
				} else {
					hostMap[hostname].Status = "failed" // 更新状态为失败
				}
				hostMap[hostname].TasksFailed++
				hostMap[hostname].EndTime = log.CreatedAt
			}
		} else if strings.Contains(content, "skipped: [") {
			hostname := extractHostname(content, "skipped")
			if hostname != "" {
				if _, exists := hostMap[hostname]; !exists {
					hostMap[hostname] = &model.HostExecutionStatus{
						HostName:  hostname,
						Status:    "skipped",
						StartTime: log.CreatedAt,
					}
				}
				hostMap[hostname].TasksSkipped++
				hostMap[hostname].EndTime = log.CreatedAt
			}
		} else if strings.Contains(content, "changed: [") {
			hostname := extractHostname(content, "changed")
			if hostname != "" {
				if _, exists := hostMap[hostname]; !exists {
					hostMap[hostname] = &model.HostExecutionStatus{
						HostName:  hostname,
						Status:    "ok",
						StartTime: log.CreatedAt,
					}
				}
				hostMap[hostname].Changed = true
				hostMap[hostname].TasksOk++
				hostMap[hostname].EndTime = log.CreatedAt
			}
		}
	}

	// 计算每个主机的执行时长
	for _, status := range hostMap {
		status.Duration = int(status.EndTime.Sub(status.StartTime).Milliseconds())
		statuses = append(statuses, *status)
	}

	return statuses
}

// extractHostname 从 Ansible 日志行中提取主机名
func extractHostname(content, prefix string) string {
	// 格式: "ok: [hostname]" 或 "failed: [hostname]"
	start := strings.Index(content, prefix+": [")
	if start == -1 {
		return ""
	}
	start += len(prefix) + 3 // 跳过 "prefix: ["
	
	end := strings.Index(content[start:], "]")
	if end == -1 {
		return ""
	}
	
	return content[start : start+end]
}

// GetTaskTimelineSummary 获取任务时间线摘要
func (s *VisualizationService) GetTaskTimelineSummary(taskID uint) (map[string]interface{}, error) {
	viz, err := s.GetTaskVisualization(taskID)
	if err != nil {
		return nil, err
	}

	summary := make(map[string]interface{})
	summary["task_id"] = viz.TaskID
	summary["task_name"] = viz.TaskName
	summary["status"] = viz.Status
	summary["total_duration_ms"] = viz.TotalDuration
	summary["total_duration_readable"] = formatDuration(viz.TotalDuration)
	summary["phase_count"] = len(viz.Timeline)
	summary["host_count"] = len(viz.HostStatuses)
	
	// 统计各阶段
	phaseStats := make(map[string]int)
	for _, event := range viz.Timeline {
		phaseStats[string(event.Phase)]++
	}
	summary["phase_stats"] = phaseStats
	
	// 统计主机状态
	hostStatusStats := make(map[string]int)
	for _, host := range viz.HostStatuses {
		hostStatusStats[host.Status]++
	}
	summary["host_status_stats"] = hostStatusStats
	
	return summary, nil
}

// formatDuration 格式化时长
func formatDuration(ms int) string {
	d := time.Duration(ms) * time.Millisecond
	
	if d < time.Second {
		return fmt.Sprintf("%dms", ms)
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}
	
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

