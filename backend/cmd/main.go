package main

import (
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/handler"
	"kube-node-manager/internal/model"
	"kube-node-manager/internal/service"
	"kube-node-manager/pkg/database"
	"kube-node-manager/pkg/logger"
	"kube-node-manager/pkg/static"
	"log"
	"os"
	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.LoadConfig()

	logger := logger.NewLogger()

	db, err := database.InitDatabase(cfg.Database.DSN)
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
	setupRoutes(router, handlers)

	// 设置静态文件服务（必须在API路由之后）
	router.Use(static.StaticFileHandler())

	logger.Info("Server starting on port " + cfg.Server.Port)
	if err := router.Run(":" + cfg.Server.Port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func setupRoutes(router *gin.Engine, handlers *handler.Handlers) {
	// 健康检查端点
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "kube-node-manager"})
	})

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

	// WebSocket 进度推送
	protected.GET("/progress/ws", handlers.Progress.HandleWebSocket)

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
		// 批量污点操作
		nodes.POST("/taints/batch-add", handlers.Taint.BatchAddTaints)
		nodes.POST("/taints/batch-delete", handlers.Taint.BatchDeleteTaints)
		nodes.POST("/taints/batch-add-progress", handlers.Taint.BatchAddTaintsWithProgress)
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
}

// getVersion 读取VERSION文件内容
func getVersion() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev" // 如果读取失败，返回默认版本
	}
	return strings.TrimSpace(string(data))
}
