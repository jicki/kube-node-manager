package main

import (
	"context"
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/handler"
	"kube-node-manager/internal/handler/health"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service"
	"kube-node-manager/pkg/database"
	"kube-node-manager/pkg/logger"
	"kube-node-manager/pkg/static"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func main() {
	cfg := config.LoadConfig()

	logger := logger.NewLogger()

	// 初始化数据库
	dbConfig := database.DatabaseConfig{
		Type:         cfg.Database.Type,
		DSN:          cfg.Database.DSN,
		Host:         cfg.Database.Host,
		Port:         cfg.Database.Port,
		Database:     cfg.Database.Database,
		Username:     cfg.Database.Username,
		Password:     cfg.Database.Password,
		SSLMode:      cfg.Database.SSLMode,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
		MaxLifetime:  cfg.Database.MaxLifetime,
	}
	db, err := database.InitDatabase(dbConfig)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}

	if err := model.AutoMigrate(db); err != nil {
		log.Fatal("Failed to run migrations:", err)
	}

	if err := model.SeedDefaultData(db); err != nil {
		log.Fatal("Failed to seed default data:", err)
	}

	services := service.NewServices(db, logger, cfg)
	handlers := handler.NewHandlers(services, logger)
	healthHandler := health.NewHealthHandler(db)

	// 初始化飞书事件客户端（如果已启用）
	go func() {
		if err := services.Feishu.InitializeEventClient(); err != nil {
			logger.Error("Failed to initialize Feishu event client: " + err.Error())
		}
	}()

	// 启动节点异常监控服务
	services.Anomaly.StartMonitoring()

	// 启动异常报告调度器
	services.AnomalyReport.StartScheduler()

	// 启动 Ansible 定时任务调度服务
	if err := services.Ansible.GetScheduleService().Start(); err != nil {
		logger.Error("Failed to start Ansible schedule service: " + err.Error())
	}

	router := gin.Default()

	// 在生产模式下不需要CORS，因为前后端在同一域名下
	if cfg.Server.Mode == "debug" {
		router.Use(cors.New(cors.Config{
			AllowOrigins:     []string{"http://localhost:3000", "http://localhost:8080"},
			AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
			ExposeHeaders:    []string{"Content-Length"},
			AllowCredentials: true,
		}))
	}

	// 设置API路由
	setupRoutes(router, handlers, healthHandler)

	// 设置静态文件服务（必须在API路由之后）
	router.Use(static.StaticFileHandler())

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        router,
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1 MB
	}

	// 启动服务器
	go func() {
		logger.Info("Server starting on port " + cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server:", err)
		}
	}()

	// 优雅关闭
	gracefulShutdown(srv, db, logger, services)
}

func setupRoutes(router *gin.Engine, handlers *handler.Handlers, healthHandler *health.HealthHandler) {
	// 健康检查端点（支持微服务架构）
	healthGroup := router.Group("/health")
	{
		healthGroup.GET("/", healthHandler.HealthCheck)                 // 基础健康检查
		healthGroup.GET("/live", healthHandler.LivenessProbe)           // K8s 存活探针
		healthGroup.GET("/ready", healthHandler.ReadinessProbe)         // K8s 就绪探针
		healthGroup.GET("/detailed", healthHandler.DetailedHealthCheck) // 详细健康检查
	}

	// 监控指标端点
	router.GET("/metrics", healthHandler.HealthMetrics)

	// 保持向后兼容的健康检查端点
	router.GET("/api/v1/health", healthHandler.HealthCheck)

	// 版本信息端点
	router.GET("/api/v1/version", func(c *gin.Context) {
		version := getVersion()
		c.JSON(200, gin.H{
			"version": version,
			"service": "kube-node-manager",
		})
	})

	api := router.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/login", handlers.Auth.Login)
		auth.POST("/logout", handlers.Auth.Logout)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
		auth.GET("/user", handlers.Auth.AuthMiddleware(), handlers.Auth.GetUser)
		auth.PUT("/profile", handlers.Auth.AuthMiddleware(), handlers.Auth.UpdateProfile)
		auth.POST("/change-password", handlers.Auth.AuthMiddleware(), handlers.Auth.ChangePassword)
		auth.GET("/profile/stats", handlers.Auth.AuthMiddleware(), handlers.Auth.GetProfileStats)
		auth.POST("/test-ldap", handlers.Auth.AuthMiddleware(), handlers.Auth.TestLDAPConnection)
		auth.POST("/diagnose-ldap", handlers.Auth.AuthMiddleware(), handlers.Auth.DiagnoseLDAP)
	}

	protected := api.Group("/")
	protected.Use(handlers.Auth.AuthMiddleware())

	// WebSocket 进度推送 (不需要中间件，在内部处理认证)
	api.GET("/progress/ws", handlers.Progress.HandleWebSocket)

	// WebSocket 节点实时同步 (节点状态实时推送)
	api.GET("/nodes/ws", handlers.WebSocket.HandleWebSocket)

	users := protected.Group("/users")
	{
		users.GET("", handlers.User.List)
		users.GET("/:id", handlers.User.GetByID)
		users.POST("", handlers.User.Create)
		users.PUT("/:id", handlers.User.Update)
		users.DELETE("/:id", handlers.User.Delete)
		users.PUT("/:id/password", handlers.User.UpdatePassword)
		users.POST("/:id/reset-password", handlers.User.ResetPassword)
	}

	clusters := protected.Group("/clusters")
	{
		clusters.GET("", handlers.Cluster.List)
		clusters.GET("/:id", handlers.Cluster.GetByID)
		clusters.POST("", handlers.Cluster.Create)
		clusters.PUT("/:id", handlers.Cluster.Update)
		clusters.DELETE("/:id", handlers.Cluster.Delete)
		clusters.POST("/:id/sync", handlers.Cluster.Sync)
		clusters.POST("/:id/test", handlers.Cluster.TestConnection)
	}

	nodes := protected.Group("/nodes")
	{
		nodes.GET("", handlers.Node.List)
		nodes.GET("/:cluster_id/:node_name", handlers.Node.Get)
		nodes.GET("/:cluster_id/stats", handlers.Node.GetSummary)
		// 单节点操作
		nodes.POST("/:node_name/cordon", handlers.Node.Cordon)
		nodes.POST("/:node_name/uncordon", handlers.Node.Uncordon)
		nodes.POST("/:node_name/drain", handlers.Node.Drain)
		// 批量节点操作
		nodes.POST("/batch-cordon", handlers.Node.BatchCordon)
		nodes.POST("/batch-uncordon", handlers.Node.BatchUncordon)
		nodes.POST("/batch-drain", handlers.Node.BatchDrain)
		// 禁止调度历史查询 (避免路由冲突，放在批量操作中)
		nodes.POST("/batch-cordon-history", handlers.Node.GetBatchCordonHistory)
		nodes.POST("/cordon-history", handlers.Node.GetCordonHistory)
		nodes.POST("/cordon-info", handlers.Node.GetNodeCordonInfo)
		// kubectl-plugin annotations同步
		nodes.POST("/sync-cordon-annotations", handlers.Node.SyncCordonAnnotations)
		// 批量标签操作
		nodes.POST("/labels/batch-add", handlers.Label.BatchAddLabels)
		nodes.POST("/labels/batch-delete", handlers.Label.BatchDeleteLabels)
		nodes.POST("/labels/batch-add-progress", handlers.Label.BatchAddLabelsWithProgress)
		nodes.POST("/labels/batch-delete-progress", handlers.Label.BatchDeleteLabelsWithProgress)
		// 批量污点操作
		nodes.POST("/taints/batch-add", handlers.Taint.BatchAddTaints)
		nodes.POST("/taints/batch-delete", handlers.Taint.BatchDeleteTaints)
		nodes.POST("/taints/batch-add-progress", handlers.Taint.BatchAddTaintsWithProgress)
		nodes.POST("/taints/batch-delete-progress", handlers.Taint.BatchDeleteTaintsWithProgress)
		nodes.POST("/taints/batch-copy", handlers.Taint.BatchCopyTaints)
		nodes.POST("/taints/batch-copy-progress", handlers.Taint.BatchCopyTaintsWithProgress)
		// 节点操作（带进度）
		nodes.POST("/batch-cordon-progress", handlers.Node.BatchCordonWithProgress)
		nodes.POST("/batch-uncordon-progress", handlers.Node.BatchUncordonWithProgress)
		nodes.POST("/batch-drain-progress", handlers.Node.BatchDrainWithProgress)
	}

	labels := protected.Group("/labels")
	{
		labels.GET("/:cluster_id/:node_name", handlers.Label.GetLabelUsage)
		labels.POST("/:cluster_id/:node_name", handlers.Label.UpdateNodeLabels)
		labels.DELETE("/:cluster_id/:node_name", handlers.Label.BatchUpdateLabels)
		labels.GET("/templates", handlers.Label.ListTemplates)
		labels.POST("/templates", handlers.Label.CreateTemplate)
		labels.PUT("/templates/:id", handlers.Label.UpdateTemplate)
		labels.DELETE("/templates/:id", handlers.Label.DeleteTemplate)
		labels.POST("/templates/apply", handlers.Label.ApplyTemplate)
	}

	taints := protected.Group("/taints")
	{
		taints.GET("/:cluster_id/:node_name", handlers.Taint.GetTaintUsage)
		taints.POST("/:cluster_id/:node_name", handlers.Taint.UpdateNodeTaints)
		taints.DELETE("/:cluster_id/:node_name", handlers.Taint.RemoveTaint)
		taints.POST("/copy", handlers.Taint.CopyNodeTaints)
		taints.POST("/batch-copy", handlers.Taint.BatchCopyTaints)
		taints.POST("/batch-copy-progress", handlers.Taint.BatchCopyTaintsWithProgress)
		taints.GET("/templates", handlers.Taint.ListTemplates)
		taints.POST("/templates", handlers.Taint.CreateTemplate)
		taints.PUT("/templates/:id", handlers.Taint.UpdateTemplate)
		taints.DELETE("/templates/:id", handlers.Taint.DeleteTemplate)
		taints.POST("/templates/apply", handlers.Taint.ApplyTemplate)
	}

	audit := protected.Group("/audit")
	{
		audit.GET("/logs", handlers.Audit.List)
		audit.GET("/logs/:id", handlers.Audit.GetByID)
	}

	// GitLab routes (admin only)
	gitlab := protected.Group("/gitlab")
	{
		gitlab.GET("/settings", handlers.Gitlab.GetSettings)
		gitlab.PUT("/settings", handlers.Gitlab.UpdateSettings)
		gitlab.POST("/test", handlers.Gitlab.TestConnection)
		gitlab.GET("/runners", handlers.Gitlab.ListRunners)
		gitlab.POST("/runners", handlers.Gitlab.CreateRunner)
		gitlab.GET("/runners/:id", handlers.Gitlab.GetRunner)
		gitlab.GET("/runners/:id/jobs", handlers.Gitlab.GetRunnerJobs)
		gitlab.GET("/runners/:id/token", handlers.Gitlab.GetRunnerToken)
		gitlab.POST("/runners/:id/reset-token", handlers.Gitlab.ResetRunnerToken)
		gitlab.PUT("/runners/:id", handlers.Gitlab.UpdateRunner)
		gitlab.DELETE("/runners/:id", handlers.Gitlab.DeleteRunner)
		gitlab.GET("/pipelines", handlers.Gitlab.ListPipelines)
		gitlab.GET("/pipelines/:project_id/:pipeline_id", handlers.Gitlab.GetPipelineDetail)
		gitlab.GET("/pipelines/:project_id/:pipeline_id/jobs", handlers.Gitlab.GetPipelineJobs)
	}

	// Feishu routes (使用长连接模式，无需 webhook)
	feishu := protected.Group("/feishu")
	{
		// 所有用户可访问
		feishu.GET("/settings", handlers.Feishu.GetSettings)
		feishu.POST("/groups/query", handlers.Feishu.QueryGroup)
		feishu.GET("/groups", handlers.Feishu.ListGroups)
		feishu.GET("/bind", handlers.Feishu.GetBinding)
		feishu.POST("/bind", handlers.Feishu.BindUser)
		feishu.DELETE("/bind", handlers.Feishu.UnbindUser)
		// 仅管理员可访问
		feishu.PUT("/settings", handlers.Feishu.UpdateSettings)
		feishu.POST("/test", handlers.Feishu.TestConnection)
	}

	// Anomaly routes (节点异常统计)
	anomalies := protected.Group("/anomalies")
	{
		anomalies.GET("", handlers.Anomaly.List)
		anomalies.GET("/statistics", handlers.Anomaly.GetStatistics)
		anomalies.GET("/active", handlers.Anomaly.GetActive)
		anomalies.GET("/type-statistics", handlers.Anomaly.GetTypeStatistics)
		anomalies.POST("/check", handlers.Anomaly.TriggerCheck)

		// 高级统计接口
		anomalies.GET("/role-statistics", handlers.Anomaly.GetRoleStatistics)
		anomalies.GET("/cluster-aggregate", handlers.Anomaly.GetClusterAggregate)
		anomalies.GET("/node-trend", handlers.Anomaly.GetNodeTrend)
		anomalies.GET("/mttr", handlers.Anomaly.GetMTTR)
		anomalies.GET("/sla", handlers.Anomaly.GetSLA)
		anomalies.GET("/recovery-metrics", handlers.Anomaly.GetRecoveryMetrics)
		anomalies.GET("/node-health", handlers.Anomaly.GetNodeHealth)
		anomalies.GET("/heatmap", handlers.Anomaly.GetHeatmap)
		anomalies.GET("/calendar", handlers.Anomaly.GetCalendar)
		anomalies.GET("/top-unhealthy-nodes", handlers.Anomaly.GetTopUnhealthyNodes)

		// 数据清理相关
		anomalies.POST("/cleanup", handlers.Anomaly.TriggerCleanup)
		anomalies.GET("/cleanup/config", handlers.Anomaly.GetCleanupConfig)
		anomalies.PUT("/cleanup/config", handlers.Anomaly.UpdateCleanupConfig)
		anomalies.GET("/cleanup/stats", handlers.Anomaly.GetCleanupStats)
	}

	// Anomaly Report routes (异常报告配置管理)
	anomalyReports := protected.Group("/anomaly-reports")
	{
		anomalyReports.GET("/configs", handlers.AnomalyReport.GetReportConfigs)
		anomalyReports.GET("/configs/:id", handlers.AnomalyReport.GetReportConfig)
		anomalyReports.POST("/configs", handlers.AnomalyReport.CreateReportConfig)
		anomalyReports.PUT("/configs/:id", handlers.AnomalyReport.UpdateReportConfig)
		anomalyReports.DELETE("/configs/:id", handlers.AnomalyReport.DeleteReportConfig)
		anomalyReports.POST("/configs/:id/test", handlers.AnomalyReport.TestReportSend)
		anomalyReports.POST("/configs/:id/run", handlers.AnomalyReport.RunReportNow)
	}

	// Ansible routes (Ansible 任务管理)
	ansible := protected.Group("/ansible")
	{
		// 任务管理
		ansible.GET("/tasks", handlers.Ansible.ListTasks)
		ansible.GET("/tasks/:id", handlers.Ansible.GetTask)
		ansible.POST("/tasks", handlers.Ansible.CreateTask)
		ansible.DELETE("/tasks/:id", handlers.Ansible.DeleteTask)
		ansible.POST("/tasks/batch-delete", handlers.Ansible.DeleteTasks)
		ansible.POST("/tasks/:id/cancel", handlers.Ansible.CancelTask)
		ansible.POST("/tasks/:id/retry", handlers.Ansible.RetryTask)
		ansible.GET("/tasks/:id/logs", handlers.Ansible.GetTaskLogs)
		ansible.POST("/tasks/:id/refresh", handlers.Ansible.RefreshTaskStatus)

		// 统计信息
		ansible.GET("/statistics", handlers.Ansible.GetStatistics)

		// 模板管理
		ansible.GET("/templates", handlers.AnsibleTemplate.ListTemplates)
		ansible.GET("/templates/:id", handlers.AnsibleTemplate.GetTemplate)
		ansible.POST("/templates", handlers.AnsibleTemplate.CreateTemplate)
		ansible.PUT("/templates/:id", handlers.AnsibleTemplate.UpdateTemplate)
		ansible.DELETE("/templates/:id", handlers.AnsibleTemplate.DeleteTemplate)
		ansible.POST("/templates/validate", handlers.AnsibleTemplate.ValidateTemplate)

		// 主机清单管理
		ansible.GET("/inventories", handlers.AnsibleInventory.ListInventories)
		ansible.GET("/inventories/:id", handlers.AnsibleInventory.GetInventory)
		ansible.POST("/inventories", handlers.AnsibleInventory.CreateInventory)
		ansible.PUT("/inventories/:id", handlers.AnsibleInventory.UpdateInventory)
		ansible.DELETE("/inventories/:id", handlers.AnsibleInventory.DeleteInventory)
		ansible.POST("/inventories/generate", handlers.AnsibleInventory.GenerateFromCluster)
		ansible.POST("/inventories/:id/refresh", handlers.AnsibleInventory.RefreshInventory)

		// SSH 密钥管理
		ansible.GET("/ssh-keys", handlers.AnsibleSSHKey.List)
		ansible.GET("/ssh-keys/:id", handlers.AnsibleSSHKey.Get)
		ansible.POST("/ssh-keys", handlers.AnsibleSSHKey.Create)
		ansible.PUT("/ssh-keys/:id", handlers.AnsibleSSHKey.Update)
		ansible.DELETE("/ssh-keys/:id", handlers.AnsibleSSHKey.Delete)
		ansible.POST("/ssh-keys/:id/test", handlers.AnsibleSSHKey.TestConnection)

		// 定时任务调度管理
		ansible.GET("/schedules", handlers.AnsibleSchedule.ListSchedules)
		ansible.GET("/schedules/:id", handlers.AnsibleSchedule.GetSchedule)
		ansible.POST("/schedules", handlers.AnsibleSchedule.CreateSchedule)
		ansible.PUT("/schedules/:id", handlers.AnsibleSchedule.UpdateSchedule)
		ansible.DELETE("/schedules/:id", handlers.AnsibleSchedule.DeleteSchedule)
		ansible.POST("/schedules/:id/toggle", handlers.AnsibleSchedule.ToggleSchedule)
		ansible.POST("/schedules/:id/run-now", handlers.AnsibleSchedule.RunNow)
	}

	// Ansible WebSocket (任务日志流)
	api.GET("/ansible/tasks/:id/ws", handlers.AnsibleWebSocket.HandleTaskLogStream)
}

// gracefulShutdown 优雅关闭服务器
func gracefulShutdown(srv *http.Server, db *gorm.DB, logger *logger.Logger, services *service.Services) {
	// 创建一个接收系统信号的channel
	quit := make(chan os.Signal, 1)

	// 监听指定的信号: SIGINT (Ctrl+C) 和 SIGTERM
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	// 阻塞等待信号
	sig := <-quit
	logger.Info("Received signal: " + sig.String() + ", shutting down server gracefully...")

	// 创建一个带超时的context，给服务器30秒时间来完成正在处理的请求
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 停止节点异常监控服务
	if services != nil && services.Anomaly != nil {
		services.Anomaly.StopMonitoring()
	}

	// 停止异常报告调度器
	if services != nil && services.AnomalyReport != nil {
		services.AnomalyReport.StopScheduler()
	}

	// 停止 Ansible 定时任务调度服务
	if services != nil && services.Ansible != nil && services.Ansible.GetScheduleService() != nil {
		services.Ansible.GetScheduleService().Stop()
	}

	// 关闭HTTP服务器
	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown: " + err.Error())
	}

	// 关闭数据库连接
	if db != nil {
		if sqlDB, err := db.DB(); err == nil {
			if err := sqlDB.Close(); err != nil {
				logger.Error("Failed to close database connection: " + err.Error())
			} else {
				logger.Info("Database connection closed")
			}
		}
	}

	logger.Info("Server shutdown completed")
}

// getVersion 读取VERSION文件内容
func getVersion() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev" // 如果读取失败，返回默认版本
	}
	return strings.TrimSpace(string(data))
}
