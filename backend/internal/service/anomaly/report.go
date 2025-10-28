package anomaly

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"kube-node-manager/internal/model"
	"kube-node-manager/pkg/logger"
	"net/http"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
	"gorm.io/gorm"
)

// ReportService 报告生成服务
type ReportService struct {
	db         *gorm.DB
	logger     *logger.Logger
	anomalySvc *Service
	scheduler  *cron.Cron
	mu         sync.RWMutex
	jobMap     map[uint]cron.EntryID // config ID -> cron entry ID
	enabled    bool
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewReportService 创建报告服务实例
func NewReportService(db *gorm.DB, logger *logger.Logger, anomalySvc *Service, enabled bool) *ReportService {
	ctx, cancel := context.WithCancel(context.Background())
	return &ReportService{
		db:         db,
		logger:     logger,
		anomalySvc: anomalySvc,
		scheduler:  cron.New(), // 使用标准的 5 字段 Cron 表达式（分 时 日 月 周）
		jobMap:     make(map[uint]cron.EntryID),
		enabled:    enabled,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// StartScheduler 启动定时任务调度器
func (rs *ReportService) StartScheduler() {
	if !rs.enabled {
		rs.logger.Info("Anomaly report scheduler is disabled")
		return
	}

	rs.logger.Info("Starting anomaly report scheduler...")

	// 从数据库加载所有启用的报告配置
	if err := rs.SyncSchedulerJobs(); err != nil {
		rs.logger.Errorf("Failed to sync scheduler jobs: %v", err)
	}

	// 启动调度器
	rs.scheduler.Start()
	rs.logger.Info("Anomaly report scheduler started successfully")
}

// StopScheduler 停止调度器
func (rs *ReportService) StopScheduler() {
	if !rs.enabled {
		return
	}

	rs.logger.Info("Stopping anomaly report scheduler...")
	rs.cancel()

	// 停止调度器并等待所有任务完成
	ctx := rs.scheduler.Stop()
	<-ctx.Done()

	rs.logger.Info("Anomaly report scheduler stopped successfully")
}

// SyncSchedulerJobs 同步数据库配置到调度器
func (rs *ReportService) SyncSchedulerJobs() error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	// 清除所有现有任务
	for _, entryID := range rs.jobMap {
		rs.scheduler.Remove(entryID)
	}
	rs.jobMap = make(map[uint]cron.EntryID)

	// 从数据库加载所有启用的配置
	var configs []model.AnomalyReportConfig
	if err := rs.db.Where("enabled = ?", true).Find(&configs).Error; err != nil {
		return fmt.Errorf("failed to load report configs: %w", err)
	}

	// 为每个配置注册调度任务
	for _, config := range configs {
		if err := rs.addSchedulerJob(config); err != nil {
			rs.logger.Errorf("Failed to add scheduler job for config %d: %v", config.ID, err)
			continue
		}
	}

	rs.logger.Infof("Synced %d report scheduler jobs", len(configs))
	return nil
}

// addSchedulerJob 添加单个调度任务
func (rs *ReportService) addSchedulerJob(config model.AnomalyReportConfig) error {
	if config.Schedule == "" {
		return fmt.Errorf("empty schedule for config %d", config.ID)
	}

	// 创建任务函数
	job := func() {
		rs.logger.Infof("Executing scheduled report: %s (ID: %d)", config.ReportName, config.ID)
		if err := rs.ExecuteReport(config.ID); err != nil {
			rs.logger.Errorf("Failed to execute report %d: %v", config.ID, err)
		}
	}

	// 注册到调度器
	entryID, err := rs.scheduler.AddFunc(config.Schedule, job)
	if err != nil {
		return fmt.Errorf("failed to add cron job: %w", err)
	}

	rs.jobMap[config.ID] = entryID

	// 计算下次执行时间
	entry := rs.scheduler.Entry(entryID)
	nextTime := entry.Next

	// 更新数据库中的下次执行时间（使用 Where 和 Updates 确保更新生效）
	if err := rs.db.Model(&model.AnomalyReportConfig{}).
		Where("id = ?", config.ID).
		Update("next_run_time", nextTime).Error; err != nil {
		rs.logger.Warningf("Failed to update next run time for config %d: %v", config.ID, err)
	}

	rs.logger.Infof("Added scheduler job for report '%s' (ID: %d), next run: %v", config.ReportName, config.ID, nextTime)
	return nil
}

// GetReportConfigs 获取所有报告配置
func (rs *ReportService) GetReportConfigs() ([]model.AnomalyReportConfig, error) {
	configs := make([]model.AnomalyReportConfig, 0) // 初始化为空数组而不是 nil
	if err := rs.db.Order("id DESC").Find(&configs).Error; err != nil {
		return nil, fmt.Errorf("failed to get report configs: %w", err)
	}
	return configs, nil
}

// GetReportConfig 获取单个报告配置
func (rs *ReportService) GetReportConfig(id uint) (*model.AnomalyReportConfig, error) {
	var config model.AnomalyReportConfig
	if err := rs.db.First(&config, id).Error; err != nil {
		return nil, fmt.Errorf("failed to get report config: %w", err)
	}
	return &config, nil
}

// CreateReportConfig 创建报告配置
func (rs *ReportService) CreateReportConfig(config *model.AnomalyReportConfig) error {
	// 验证 Cron 表达式
	if config.Schedule != "" {
		if _, err := cron.ParseStandard(config.Schedule); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// 创建配置
	if err := rs.db.Create(config).Error; err != nil {
		return fmt.Errorf("failed to create report config: %w", err)
	}

	// 如果配置已启用，添加到调度器
	if config.Enabled && rs.enabled {
		rs.mu.Lock()
		if err := rs.addSchedulerJob(*config); err != nil {
			rs.logger.Errorf("Failed to add scheduler job for new config %d: %v", config.ID, err)
		}
		rs.mu.Unlock()
	}

	return nil
}

// UpdateReportConfig 更新报告配置
func (rs *ReportService) UpdateReportConfig(id uint, updates *model.AnomalyReportConfig) error {
	// 验证 Cron 表达式
	if updates.Schedule != "" {
		if _, err := cron.ParseStandard(updates.Schedule); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
	}

	// 更新配置
	if err := rs.db.Model(&model.AnomalyReportConfig{}).Where("id = ?", id).Updates(updates).Error; err != nil {
		return fmt.Errorf("failed to update report config: %w", err)
	}

	// 重新同步调度器
	if rs.enabled {
		if err := rs.SyncSchedulerJobs(); err != nil {
			rs.logger.Errorf("Failed to sync scheduler after config update: %v", err)
		}
	}

	return nil
}

// DeleteReportConfig 删除报告配置
func (rs *ReportService) DeleteReportConfig(id uint) error {
	// 从调度器移除
	rs.mu.Lock()
	if entryID, exists := rs.jobMap[id]; exists {
		rs.scheduler.Remove(entryID)
		delete(rs.jobMap, id)
	}
	rs.mu.Unlock()

	// 从数据库删除
	if err := rs.db.Delete(&model.AnomalyReportConfig{}, id).Error; err != nil {
		return fmt.Errorf("failed to delete report config: %w", err)
	}

	return nil
}

// ValidateCronExpression 验证 Cron 表达式
func (rs *ReportService) ValidateCronExpression(cronExpr string) error {
	_, err := cron.ParseStandard(cronExpr)
	return err
}

// ExecuteReport 执行报告生成（手动或定时触发）
func (rs *ReportService) ExecuteReport(configID uint) error {
	// 获取配置
	config, err := rs.GetReportConfig(configID)
	if err != nil {
		return err
	}

	// 记录开始时间
	startTime := time.Now()

	// 生成报告内容
	content, err := rs.GenerateReport(config)
	if err != nil {
		return fmt.Errorf("failed to generate report: %w", err)
	}

	// 发送报告
	var sendErrors []error

	if config.FeishuEnabled && config.FeishuWebhook != "" {
		if err := rs.SendReportToFeishu(config, content); err != nil {
			rs.logger.Errorf("Failed to send report to Feishu: %v", err)
			sendErrors = append(sendErrors, err)
		}
	}

	if config.EmailEnabled && config.EmailRecipients != "" {
		if err := rs.SendReportToEmail(config, content); err != nil {
			rs.logger.Errorf("Failed to send report to Email: %v", err)
			sendErrors = append(sendErrors, err)
		}
	}

	// 更新最后执行时间
	now := time.Now()
	if err := rs.db.Model(config).Updates(map[string]interface{}{
		"last_run_time": now,
	}).Error; err != nil {
		rs.logger.Warningf("Failed to update last run time: %v", err)
	}

	rs.logger.Infof("Report '%s' executed successfully in %v", config.ReportName, time.Since(startTime))

	if len(sendErrors) > 0 {
		return fmt.Errorf("report generated but failed to send to some channels: %v", sendErrors)
	}

	return nil
}

// ReportContent 报告内容结构
type ReportContent struct {
	ReportName  string
	PeriodStart time.Time
	PeriodEnd   time.Time
	Clusters    []string
	Summary     ReportSummary
	TrendData   []model.AnomalyStatistics
	TypeStats   []model.AnomalyTypeStatistics
	TopNodes    []TopNodeInfo
	SLAMetrics  *model.SLAMetrics
	MTTRStats   []model.MTTRStatistics
}

// ReportSummary 报告摘要
type ReportSummary struct {
	TotalAnomalies    int64
	ActiveAnomalies   int64
	ResolvedAnomalies int64
	AffectedNodes     int64
	TotalClusters     int
}

// TopNodeInfo 节点异常信息
type TopNodeInfo struct {
	NodeName     string
	ClusterName  string
	AnomalyCount int64
	HealthScore  float64
}

// GenerateReport 生成报告内容
func (rs *ReportService) GenerateReport(config *model.AnomalyReportConfig) (*ReportContent, error) {
	// 确定报告时间范围
	endTime := time.Now()
	var startTime time.Time

	switch config.Frequency {
	case model.ReportFrequencyDaily:
		startTime = endTime.AddDate(0, 0, -1)
	case model.ReportFrequencyWeekly:
		startTime = endTime.AddDate(0, 0, -7)
	case model.ReportFrequencyMonthly:
		startTime = endTime.AddDate(0, -1, 0)
	default:
		startTime = endTime.AddDate(0, 0, -1)
	}

	// 解析集群ID列表
	var clusterIDs []uint
	if config.ClusterIDs != "" {
		if err := json.Unmarshal([]byte(config.ClusterIDs), &clusterIDs); err != nil {
			rs.logger.Warningf("Failed to parse cluster IDs: %v", err)
		}
	}

	// 构建报告内容
	content := &ReportContent{
		ReportName:  config.ReportName,
		PeriodStart: startTime,
		PeriodEnd:   endTime,
	}

	// 获取集群列表
	var clusters []model.Cluster
	query := rs.db.Where("status = ?", model.ClusterStatusActive)
	if len(clusterIDs) > 0 {
		query = query.Where("id IN ?", clusterIDs)
	}
	if err := query.Find(&clusters).Error; err == nil {
		for _, c := range clusters {
			content.Clusters = append(content.Clusters, c.Name)
		}
		content.Summary.TotalClusters = len(clusters)
	}

	// 获取统计摘要
	var clusterID *uint
	if len(clusterIDs) == 1 {
		clusterID = &clusterIDs[0]
	}

	summary, err := rs.anomalySvc.GetAnomalySummary(clusterID)
	if err == nil {
		if totalCount, ok := summary["total_count"].(int64); ok {
			content.Summary.TotalAnomalies = totalCount
		}
		if activeCount, ok := summary["active_count"].(int64); ok {
			content.Summary.ActiveAnomalies = activeCount
		}
		if resolvedCount, ok := summary["resolved_count"].(int64); ok {
			content.Summary.ResolvedAnomalies = resolvedCount
		}
		if affectedNodes, ok := summary["affected_nodes"].(int64); ok {
			content.Summary.AffectedNodes = affectedNodes
		}
	}

	// 获取趋势数据
	trendData, err := rs.anomalySvc.GetStatistics(StatisticsRequest{
		ClusterID: clusterID,
		StartTime: &startTime,
		EndTime:   &endTime,
		Dimension: "day",
	})
	if err == nil {
		content.TrendData = trendData
	}

	// 获取类型统计
	typeStats, err := rs.anomalySvc.GetTypeStatistics(clusterID, &startTime, &endTime)
	if err == nil {
		content.TypeStats = typeStats
	}

	// 获取 Top 10 异常节点
	topNodes, err := rs.anomalySvc.GetTopUnhealthyNodes(clusterID, 10, &startTime, &endTime)
	if err == nil {
		for _, node := range topNodes {
			content.TopNodes = append(content.TopNodes, TopNodeInfo{
				NodeName:     node.NodeName,
				ClusterName:  node.ClusterName,
				AnomalyCount: node.TotalAnomalies,
				HealthScore:  node.HealthScore,
			})
		}
	}

	// 获取 MTTR 统计
	entityType := "cluster"
	mttrStats, err := rs.anomalySvc.GetMTTRStatistics(entityType, clusterID, &startTime, &endTime)
	if err == nil {
		content.MTTRStats = mttrStats
	}

	return content, nil
}

// SendReportToFeishu 发送报告到飞书
func (rs *ReportService) SendReportToFeishu(config *model.AnomalyReportConfig, content *ReportContent) error {
	rs.logger.Infof("Sending report to Feishu webhook: %s", config.FeishuWebhook)

	// 构建飞书卡片
	card := rs.buildFeishuReportCard(content)

	// 准备 webhook 请求体
	webhookReq := map[string]interface{}{
		"msg_type": "interactive",
		"card":     card,
	}

	jsonData, err := json.Marshal(webhookReq)
	if err != nil {
		return fmt.Errorf("failed to marshal webhook request: %w", err)
	}

	// 发送 HTTP POST 请求到 webhook
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(config.FeishuWebhook, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send webhook request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read webhook response: %w", err)
	}

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("webhook returned error status %d: %s", resp.StatusCode, string(body))
	}

	// 解析响应 JSON
	var webhookResp map[string]interface{}
	if err := json.Unmarshal(body, &webhookResp); err != nil {
		rs.logger.Warningf("Failed to parse webhook response: %v", err)
		// 不返回错误，因为消息可能已发送成功
		return nil
	}

	// 检查飞书 API 返回码
	if code, ok := webhookResp["code"].(float64); ok && code != 0 {
		msg := webhookResp["msg"].(string)
		return fmt.Errorf("feishu webhook error: code=%v, msg=%s", code, msg)
	}

	rs.logger.Infof("Successfully sent report to Feishu webhook")
	return nil
}

// SendReportToEmail 发送报告到邮箱
func (rs *ReportService) SendReportToEmail(config *model.AnomalyReportConfig, content *ReportContent) error {
	// TODO: 实现邮件发送逻辑
	// 这里需要 SMTP 配置和邮件模板

	rs.logger.Infof("Sending report to email: %s", config.EmailRecipients)
	// 实现略，后续根据需要完善
	return nil
}

// TestReportSend 测试报告发送
func (rs *ReportService) TestReportSend(configID uint) error {
	config, err := rs.GetReportConfig(configID)
	if err != nil {
		return err
	}

	// 生成测试报告（使用最近7天数据）
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -7)

	testContent := &ReportContent{
		ReportName:  config.ReportName + " (测试)",
		PeriodStart: startTime,
		PeriodEnd:   endTime,
	}

	// 发送测试报告
	if config.FeishuEnabled && config.FeishuWebhook != "" {
		if err := rs.SendReportToFeishu(config, testContent); err != nil {
			return fmt.Errorf("failed to send test report to Feishu: %w", err)
		}
	}

	if config.EmailEnabled && config.EmailRecipients != "" {
		if err := rs.SendReportToEmail(config, testContent); err != nil {
			return fmt.Errorf("failed to send test report to Email: %w", err)
		}
	}

	return nil
}

// buildFeishuReportCard 构建飞书报告卡片
func (rs *ReportService) buildFeishuReportCard(content *ReportContent) map[string]interface{} {
	// 格式化时间范围
	timeRange := fmt.Sprintf("%s ~ %s",
		content.PeriodStart.Format("2006-01-02 15:04"),
		content.PeriodEnd.Format("2006-01-02 15:04"))

	// 构建元素列表
	elements := []interface{}{
		// 报告时间范围
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**📅 报告周期**: %s", timeRange),
				"tag":     "lark_md",
			},
		},
		// 集群信息
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": fmt.Sprintf("**🏢 监控集群**: %s（共 %d 个）",
					formatClusters(content.Clusters),
					content.Summary.TotalClusters),
				"tag": "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "hr",
		},
		// 统计摘要
		map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**📊 统计摘要**",
				"tag":     "lark_md",
			},
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**总异常数**\n%d", content.Summary.TotalAnomalies),
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**活跃异常**\n🔴 %d", content.Summary.ActiveAnomalies),
					},
				},
			},
		},
		map[string]interface{}{
			"tag": "div",
			"fields": []interface{}{
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**已恢复**\n✅ %d", content.Summary.ResolvedAnomalies),
					},
				},
				map[string]interface{}{
					"is_short": true,
					"text": map[string]interface{}{
						"tag":     "lark_md",
						"content": fmt.Sprintf("**受影响节点**\n⚠️ %d", content.Summary.AffectedNodes),
					},
				},
			},
		},
	}

	// 添加异常类型统计
	if len(content.TypeStats) > 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**🔍 异常类型分布**",
				"tag":     "lark_md",
			},
		})

		typeTexts := make([]string, 0, len(content.TypeStats))
		for _, ts := range content.TypeStats {
			icon := getAnomalyTypeIcon(string(ts.AnomalyType))
			typeTexts = append(typeTexts, fmt.Sprintf("• %s %s: %d 次", icon, ts.AnomalyType, ts.TotalCount))
		}

		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": formatList(typeTexts, 5),
				"tag":     "lark_md",
			},
		})
	}

	// 添加问题节点 Top 5
	if len(content.TopNodes) > 0 {
		elements = append(elements, map[string]interface{}{
			"tag": "hr",
		})
		elements = append(elements, map[string]interface{}{
			"tag": "div",
			"text": map[string]interface{}{
				"content": "**⚠️ 异常最多的节点（Top 5）**",
				"tag":     "lark_md",
			},
		})

		topCount := 5
		if len(content.TopNodes) < topCount {
			topCount = len(content.TopNodes)
		}

		for i := 0; i < topCount; i++ {
			node := content.TopNodes[i]
			healthIcon := getHealthIcon(node.HealthScore)
			elements = append(elements, map[string]interface{}{
				"tag": "div",
				"text": map[string]interface{}{
					"content": fmt.Sprintf("%d. **%s** (%s)\n   异常: %d 次 | 健康度: %s %.1f%%",
						i+1, node.NodeName, node.ClusterName,
						node.AnomalyCount, healthIcon, node.HealthScore),
					"tag": "lark_md",
				},
			})
		}
	}

	// 添加注释
	elements = append(elements, map[string]interface{}{
		"tag": "hr",
	})
	elements = append(elements, map[string]interface{}{
		"tag": "note",
		"elements": []interface{}{
			map[string]interface{}{
				"tag":     "plain_text",
				"content": "💡 查看详细数据请访问 Kube Node Manager 控制台",
			},
		},
	})

	// 构建卡片
	card := map[string]interface{}{
		"config": map[string]interface{}{
			"wide_screen_mode": true,
		},
		"header": map[string]interface{}{
			"template": "blue",
			"title": map[string]interface{}{
				"content": fmt.Sprintf("📊 %s", content.ReportName),
				"tag":     "plain_text",
			},
		},
		"elements": elements,
	}

	return card
}

// formatClusters 格式化集群列表
func formatClusters(clusters []string) string {
	if len(clusters) == 0 {
		return "所有集群"
	}
	if len(clusters) <= 3 {
		result := ""
		for i, c := range clusters {
			if i > 0 {
				result += ", "
			}
			result += c
		}
		return result
	}
	return fmt.Sprintf("%s 等 %d 个", clusters[0], len(clusters))
}

// formatList 格式化列表，最多显示 maxItems 项
func formatList(items []string, maxItems int) string {
	if len(items) == 0 {
		return "无"
	}

	result := ""
	count := maxItems
	if len(items) < count {
		count = len(items)
	}

	for i := 0; i < count; i++ {
		if i > 0 {
			result += "\n"
		}
		result += items[i]
	}

	if len(items) > maxItems {
		result += fmt.Sprintf("\n... 还有 %d 项", len(items)-maxItems)
	}

	return result
}

// getAnomalyTypeIcon 获取异常类型对应的图标
func getAnomalyTypeIcon(anomalyType string) string {
	icons := map[string]string{
		"NotReady":           "🔴",
		"DiskPressure":       "💾",
		"MemoryPressure":     "🧠",
		"PIDPressure":        "⚙️",
		"NetworkUnavailable": "🌐",
	}
	if icon, ok := icons[anomalyType]; ok {
		return icon
	}
	return "⚠️"
}

// getHealthIcon 根据健康度获取图标
func getHealthIcon(score float64) string {
	if score >= 90 {
		return "🟢"
	} else if score >= 70 {
		return "🟡"
	} else if score >= 50 {
		return "🟠"
	}
	return "🔴"
}
