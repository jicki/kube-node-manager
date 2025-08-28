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

	api := router.Group("/api/v1")

	auth := api.Group("/auth")
	{
		auth.POST("/login", handlers.Auth.Login)
		auth.POST("/logout", handlers.Auth.Logout)
		auth.POST("/refresh", handlers.Auth.RefreshToken)
	}

	protected := api.Group("/")
	protected.Use(handlers.Auth.AuthMiddleware())

	users := protected.Group("/users")
	{
		users.GET("", handlers.User.List)
		users.GET("/:id", handlers.User.GetByID)
		users.POST("", handlers.User.Create)
		users.PUT("/:id", handlers.User.Update)
		users.DELETE("/:id", handlers.User.Delete)
		users.PUT("/:id/password", handlers.User.UpdatePassword)
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
	}

	labels := protected.Group("/labels")
	{
		labels.GET("/:cluster_id/:node_name", handlers.Label.GetLabelUsage)
		labels.POST("/:cluster_id/:node_name", handlers.Label.UpdateNodeLabels)
		labels.DELETE("/:cluster_id/:node_name", handlers.Label.BatchUpdateLabels)
		labels.GET("/templates", handlers.Label.ListTemplates)
		labels.POST("/templates", handlers.Label.CreateTemplate)
		labels.DELETE("/templates/:id", handlers.Label.DeleteTemplate)
	}

	taints := protected.Group("/taints")
	{
		taints.GET("/:cluster_id/:node_name", handlers.Taint.GetTaintUsage)
		taints.POST("/:cluster_id/:node_name", handlers.Taint.UpdateNodeTaints)
		taints.DELETE("/:cluster_id/:node_name", handlers.Taint.RemoveTaint)
		taints.GET("/templates", handlers.Taint.ListTemplates)
		taints.POST("/templates", handlers.Taint.CreateTemplate)
		taints.DELETE("/templates/:id", handlers.Taint.DeleteTemplate)
	}

	audit := protected.Group("/audit")
	{
		audit.GET("", handlers.Audit.List)
		audit.GET("/:id", handlers.Audit.GetByID)
	}
}
