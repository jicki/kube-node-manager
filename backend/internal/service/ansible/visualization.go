package ansible

import (
	"fmt"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"regexp"
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
	if err := s.db.Preload("Inventory").First(&task, taskID).Error; err != nil {
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

	// 如果有执行时间线，使用它但需要重新计算 Duration
	if task.ExecutionTimeline != nil && len(*task.ExecutionTimeline) > 0 {
		s.logger.Infof("Using stored execution timeline with %d events", len(*task.ExecutionTimeline))
		viz.Timeline = *task.ExecutionTimeline
		
		// 重新计算每个事件的耗时，确保 Duration 正确
		s.recalculateTimelineDurations(&viz.Timeline, &task)
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

	// 解析日志中的 TASK 阶段并添加到时间线（如果有日志）
	if task.FullLog != "" && len(viz.Timeline) > 0 {
		s.parseAndEnrichTimeline(&viz.Timeline, &task)
	}

	// 获取主机执行状态
	viz.HostStatuses = s.extractHostStatuses(&task)
	s.logger.Infof("Extracted %d host statuses", len(viz.HostStatuses))

	return viz, nil
}

// recalculateTimelineDurations 重新计算时间线中每个事件的耗时
func (s *VisualizationService) recalculateTimelineDurations(timeline *model.TaskExecutionTimeline, task *model.AnsibleTask) {
	if timeline == nil || len(*timeline) == 0 {
		return
	}
	
	s.logger.Infof("Recalculating durations for %d timeline events", len(*timeline))
	
	// 重新计算每个事件的耗时
	for i := 0; i < len(*timeline); i++ {
		if i < len(*timeline)-1 {
			// 不是最后一个事件，用下一个事件的时间戳计算
			duration := int((*timeline)[i+1].Timestamp.Sub((*timeline)[i].Timestamp).Milliseconds())
			if duration > 0 {
				(*timeline)[i].Duration = duration
			}
		} else {
			// 最后一个事件，使用任务总耗时计算剩余时间
			if task.Duration > 0 && len(*timeline) > 1 {
				// 计算已有事件的总耗时
				totalDuration := 0
				for j := 0; j < i; j++ {
					totalDuration += (*timeline)[j].Duration
				}
				
				// 剩余时间分配给最后一个事件
				taskDurationMs := task.Duration * 1000
				remainingDuration := taskDurationMs - totalDuration
				
				if remainingDuration > 0 {
					(*timeline)[i].Duration = remainingDuration
					s.logger.Debugf("Assigned remaining duration %dms to final event", remainingDuration)
				} else if totalDuration == 0 {
					// 如果前面所有事件都没有耗时，把整个任务耗时分配给最后一个事件
					(*timeline)[i].Duration = taskDurationMs
					s.logger.Debugf("Assigned full task duration %dms to final event", taskDurationMs)
				}
			}
		}
	}
	
	// 打印调试信息
	for i, event := range *timeline {
		s.logger.Debugf("Event %d after recalc: phase=%s, duration=%dms", i, event.Phase, event.Duration)
	}
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

	// 执行开始事件
	if task.StartedAt != nil {
		timeline = append(timeline, model.TaskExecutionEvent{
			Phase:     model.PhaseExecuting,
			Message:   "任务开始执行",
			Timestamp: *task.StartedAt,
			HostCount: task.HostsTotal,
		})
	}

	// 完成事件（仅在任务已完成时添加）
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
	}

	// 计算每个事件的耗时
	// 耗时表示从当前事件到下一个事件之间的时间间隔
	for i := 0; i < len(timeline); i++ {
		if timeline[i].Duration == 0 {
			if i < len(timeline)-1 {
				// 不是最后一个事件，用下一个事件的时间戳计算
				duration := int(timeline[i+1].Timestamp.Sub(timeline[i].Timestamp).Milliseconds())
				if duration > 0 {
					timeline[i].Duration = duration
				}
			} else {
				// 最后一个事件（完成/失败事件）的耗时处理
				// 如果只有一个事件，或者前面所有事件耗时都为0，使用任务总耗时
				if task.Duration > 0 && len(timeline) > 1 {
					// 计算已有事件的总耗时
					totalDuration := 0
					for j := 0; j < i; j++ {
						totalDuration += timeline[j].Duration
					}
					
					// 剩余时间分配给最后一个事件
					taskDurationMs := task.Duration * 1000 // 转换为毫秒
					remainingDuration := taskDurationMs - totalDuration
					
					if remainingDuration > 0 {
						timeline[i].Duration = remainingDuration
						s.logger.Debugf("Assigned remaining duration %dms to final event", remainingDuration)
					} else if totalDuration == 0 {
						// 如果前面所有事件都没有耗时，把整个任务耗时分配给最后一个事件
						timeline[i].Duration = taskDurationMs
						s.logger.Debugf("Assigned full task duration %dms to final event", taskDurationMs)
					}
				}
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
	if len(timeline) == 0 {
		s.logger.Infof("No timeline events to calculate distribution")
		return nil
	}
	
	distribution := make(map[string]int)
	
	for i, event := range timeline {
		// 统计有耗时的事件
		if event.Duration > 0 {
			phase := string(event.Phase)
			distribution[phase] += event.Duration
			s.logger.Debugf("Event %d: phase=%s, duration=%dms", i, phase, event.Duration)
		} else {
			s.logger.Debugf("Event %d: phase=%s, duration=0 (skipped)", i, event.Phase)
		}
	}
	
	// 如果没有任何耗时数据，返回 nil 而不是空 map
	if len(distribution) == 0 {
		s.logger.Warningf("No phase distribution data for timeline with %d events (all durations are 0)", len(timeline))
		return nil
	}
	
	s.logger.Infof("Phase distribution calculated: %v (total phases: %d)", distribution, len(distribution))
	return distribution
}

// parseAndEnrichTimeline 解析日志中的 TASK 阶段信息并丰富时间线
func (s *VisualizationService) parseAndEnrichTimeline(timeline *model.TaskExecutionTimeline, task *model.AnsibleTask) {
	if timeline == nil || task.FullLog == "" {
		return
	}
	
	s.logger.Infof("Parsing TASK phases from log for task %d", task.ID)
	
	// 查找所有 TASK 行
	// 格式: TASK [task name] ******************
	taskRegex := regexp.MustCompile(`^TASK\s+\[([^\]]+)\]`)
	
	lines := strings.Split(task.FullLog, "\n")
	tasks := make([]struct {
		name      string
		lineIndex int
	}, 0)
	
	for i, line := range lines {
		if matches := taskRegex.FindStringSubmatch(strings.TrimSpace(line)); len(matches) > 1 {
			taskName := matches[1]
			tasks = append(tasks, struct {
				name      string
				lineIndex int
			}{
				name:      taskName,
				lineIndex: i,
			})
		}
	}
	
	if len(tasks) == 0 {
		s.logger.Infof("No TASK phases found in log")
		return
	}
	
	s.logger.Infof("Found %d TASK phases in log", len(tasks))
	
	// 找到 "executing" 阶段的时间窗口
	var executingStartTime time.Time
	var executingEndTime time.Time
	var executingIndex int = -1
	
	for i, event := range *timeline {
		if event.Phase == model.PhaseExecuting {
			executingStartTime = event.Timestamp
			executingIndex = i
			
			// 找到下一个事件的时间作为执行结束时间
			if i < len(*timeline)-1 {
				executingEndTime = (*timeline)[i+1].Timestamp
			} else {
				// 如果是最后一个事件，使用任务结束时间
				if task.FinishedAt != nil {
					executingEndTime = *task.FinishedAt
				} else {
					executingEndTime = executingStartTime.Add(time.Duration(task.Duration) * time.Second)
				}
			}
			break
		}
	}
	
	if executingIndex == -1 {
		s.logger.Warningf("No executing phase found in timeline")
		return
	}
	
	// 计算执行总时长（毫秒）
	totalExecutionMs := float64(executingEndTime.Sub(executingStartTime).Milliseconds())
	if totalExecutionMs <= 0 {
		s.logger.Warningf("Invalid execution time window")
		return
	}
	
	// 根据 TASK 在日志中的位置，估算每个 TASK 的时间戳
	// 简单策略：均匀分布
	totalLines := float64(len(lines))
	newEvents := make(model.TaskExecutionTimeline, 0, len(tasks))
	
	for i, taskInfo := range tasks {
		// 根据日志行位置计算时间偏移
		progress := float64(taskInfo.lineIndex) / totalLines
		timeOffset := time.Duration(progress * totalExecutionMs) * time.Millisecond
		timestamp := executingStartTime.Add(timeOffset)
		
		// 计算持续时间（到下一个 TASK 或执行结束）
		var duration int
		if i < len(tasks)-1 {
			nextProgress := float64(tasks[i+1].lineIndex) / totalLines
			nextOffset := time.Duration(nextProgress * totalExecutionMs) * time.Millisecond
			duration = int((nextOffset - timeOffset).Milliseconds())
		} else {
			// 最后一个 TASK，持续到执行结束
			duration = int(executingEndTime.Sub(timestamp).Milliseconds())
		}
		
		newEvents = append(newEvents, model.TaskExecutionEvent{
			Phase:     "task_execution", // 自定义阶段
			Message:   fmt.Sprintf("执行任务: %s", taskInfo.name),
			Timestamp: timestamp,
			Duration:  duration,
			Details: map[string]interface{}{
				"task_name": taskInfo.name,
			},
		})
	}
	
	if len(newEvents) == 0 {
		return
	}
	
	// 将新事件插入到时间线中（在 executing 事件之后）
	// 重新构建时间线
	newTimeline := make(model.TaskExecutionTimeline, 0, len(*timeline)+len(newEvents))
	
	// 添加 executing 之前的事件
	for i := 0; i <= executingIndex; i++ {
		newTimeline = append(newTimeline, (*timeline)[i])
	}
	
	// 添加解析出的 TASK 事件
	newTimeline = append(newTimeline, newEvents...)
	
	// 添加 executing 之后的事件
	for i := executingIndex + 1; i < len(*timeline); i++ {
		newTimeline = append(newTimeline, (*timeline)[i])
	}
	
	// 更新原时间线
	*timeline = newTimeline
	
	// 重新计算耗时
	s.recalculateTimelineDurations(timeline, task)
	
	s.logger.Infof("Enriched timeline with %d TASK events, total events: %d", len(newEvents), len(*timeline))
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

