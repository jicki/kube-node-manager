package health

import (
	"context"
	"fmt"
	"kube-node-manager/internal/service"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HealthHandler struct {
	DB               *gorm.DB
	migrationService *service.MigrationService
}

type HealthStatus struct {
	Status    string                 `json:"status"`
	Service   string                 `json:"service"`
	Version   string                 `json:"version"`
	Timestamp string                 `json:"timestamp"`
	Uptime    string                 `json:"uptime"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

type HealthDetail struct {
	Status  string      `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

var (
	startTime = time.Now()
)

func NewHealthHandler(db *gorm.DB) *HealthHandler {
	migrationService, err := service.NewMigrationService(db)
	if err != nil {
		// 迁移服务初始化失败不影响基础健康检查
		fmt.Printf("Warning: Failed to initialize migration service: %v\n", err)
	}
	
	return &HealthHandler{
		DB:               db,
		migrationService: migrationService,
	}
}

// HealthCheck 基础健康检查端点
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	status := HealthStatus{
		Status:    "healthy",
		Service:   "kube-node-manager",
		Version:   getVersion(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
	}

	c.JSON(http.StatusOK, status)
}

// LivenessProbe Kubernetes 存活探针
func (h *HealthHandler) LivenessProbe(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessProbe Kubernetes 就绪探针
func (h *HealthHandler) ReadinessProbe(c *gin.Context) {
	details := make(map[string]interface{})
	overallStatus := "ready"

	// 数据库连接检查
	dbHealth := h.checkDatabase()
	details["database"] = dbHealth
	if dbHealth.Status != "healthy" {
		overallStatus = "not ready"
	}

	status := http.StatusOK
	if overallStatus != "ready" {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"details":   details,
	})
}

// DetailedHealthCheck 详细健康检查（包含所有组件状态）
func (h *HealthHandler) DetailedHealthCheck(c *gin.Context) {
	details := make(map[string]interface{})
	overallStatus := "healthy"

	// 数据库检查
	dbHealth := h.checkDatabase()
	details["database"] = dbHealth
	if dbHealth.Status != "healthy" {
		overallStatus = "degraded"
	}

	// 系统资源检查
	systemHealth := h.checkSystemResources()
	details["system"] = systemHealth

	// 运行时信息
	runtimeInfo := h.getRuntimeInfo()
	details["runtime"] = runtimeInfo

	status := HealthStatus{
		Status:    overallStatus,
		Service:   "kube-node-manager",
		Version:   getVersion(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Uptime:    time.Since(startTime).String(),
		Details:   details,
	}

	httpStatus := http.StatusOK
	if overallStatus == "degraded" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, status)
}

// HealthMetrics 返回健康指标（用于监控系统）
func (h *HealthHandler) HealthMetrics(c *gin.Context) {
	var metrics []string

	// 基础指标
	metrics = append(metrics, fmt.Sprintf("# HELP kube_node_manager_up Application up status"))
	metrics = append(metrics, fmt.Sprintf("# TYPE kube_node_manager_up gauge"))
	metrics = append(metrics, fmt.Sprintf("kube_node_manager_up 1"))

	// 启动时间指标
	metrics = append(metrics, fmt.Sprintf("# HELP kube_node_manager_start_time_seconds Start time of the application"))
	metrics = append(metrics, fmt.Sprintf("# TYPE kube_node_manager_start_time_seconds gauge"))
	metrics = append(metrics, fmt.Sprintf("kube_node_manager_start_time_seconds %d", startTime.Unix()))

	// 数据库状态指标
	dbHealth := h.checkDatabase()
	dbStatus := 0
	if dbHealth.Status == "healthy" {
		dbStatus = 1
	}
	metrics = append(metrics, fmt.Sprintf("# HELP kube_node_manager_database_up Database connection status"))
	metrics = append(metrics, fmt.Sprintf("# TYPE kube_node_manager_database_up gauge"))
	metrics = append(metrics, fmt.Sprintf("kube_node_manager_database_up %d", dbStatus))

	// 内存使用指标
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics = append(metrics, fmt.Sprintf("# HELP kube_node_manager_memory_usage_bytes Memory usage in bytes"))
	metrics = append(metrics, fmt.Sprintf("# TYPE kube_node_manager_memory_usage_bytes gauge"))
	metrics = append(metrics, fmt.Sprintf("kube_node_manager_memory_usage_bytes %d", m.Alloc))

	// Goroutine 数量指标
	metrics = append(metrics, fmt.Sprintf("# HELP kube_node_manager_goroutines_total Number of goroutines"))
	metrics = append(metrics, fmt.Sprintf("# TYPE kube_node_manager_goroutines_total gauge"))
	metrics = append(metrics, fmt.Sprintf("kube_node_manager_goroutines_total %d", runtime.NumGoroutine()))

	c.Header("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	c.String(http.StatusOK, strings.Join(metrics, "\n")+"\n")
}

// checkDatabase 检查数据库连接状态
func (h *HealthHandler) checkDatabase() HealthDetail {
	if h.DB == nil {
		return HealthDetail{
			Status:  "unhealthy",
			Message: "Database connection not initialized",
		}
	}

	// 设置超时上下文
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 获取底层SQL DB连接
	sqlDB, err := h.DB.DB()
	if err != nil {
		return HealthDetail{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to get database connection: %v", err),
		}
	}

	// 检查连接
	if err := sqlDB.PingContext(ctx); err != nil {
		return HealthDetail{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Database ping failed: %v", err),
		}
	}

	// 获取连接池状态
	stats := sqlDB.Stats()

	return HealthDetail{
		Status: "healthy",
		Data: map[string]interface{}{
			"max_open_connections": stats.MaxOpenConnections,
			"open_connections":     stats.OpenConnections,
			"in_use":               stats.InUse,
			"idle":                 stats.Idle,
			"wait_count":           stats.WaitCount,
			"wait_duration":        stats.WaitDuration.String(),
			"max_idle_closed":      stats.MaxIdleClosed,
			"max_idle_time_closed": stats.MaxIdleTimeClosed,
			"max_lifetime_closed":  stats.MaxLifetimeClosed,
		},
	}
}

// checkSystemResources 检查系统资源状态
func (h *HealthHandler) checkSystemResources() HealthDetail {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return HealthDetail{
		Status: "healthy",
		Data: map[string]interface{}{
			"memory": map[string]interface{}{
				"alloc_mb":       bToMb(m.Alloc),
				"total_alloc_mb": bToMb(m.TotalAlloc),
				"sys_mb":         bToMb(m.Sys),
				"heap_alloc_mb":  bToMb(m.HeapAlloc),
				"heap_sys_mb":    bToMb(m.HeapSys),
			},
			"goroutines": runtime.NumGoroutine(),
			"num_cpu":    runtime.NumCPU(),
			"num_gc":     m.NumGC,
		},
	}
}

// getRuntimeInfo 获取运行时信息
func (h *HealthHandler) getRuntimeInfo() map[string]interface{} {
	hostname, _ := os.Hostname()

	return map[string]interface{}{
		"go_version": runtime.Version(),
		"go_os":      runtime.GOOS,
		"go_arch":    runtime.GOARCH,
		"hostname":   hostname,
		"pid":        os.Getpid(),
	}
}

// getVersion 读取VERSION文件内容
func getVersion() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return "dev"
	}
	return strings.TrimSpace(string(data))
}

// bToMb 将字节转换为MB
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// DatabaseHealth 数据库健康检查（包含版本信息）
func (h *HealthHandler) DatabaseHealth(c *gin.Context) {
	details := make(map[string]interface{})
	overallStatus := "healthy"

	// 基础连接检查
	dbHealth := h.checkDatabase()
	details["connection"] = dbHealth
	if dbHealth.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// 版本信息
	if h.migrationService != nil {
		versionInfo := h.migrationService.GetVersionInfo()
		details["version"] = map[string]interface{}{
			"app_version":          versionInfo.AppVersion,
			"db_version":           versionInfo.DBVersion,
			"latest_schema":        versionInfo.LatestSchemaVersion,
			"needs_migration":      versionInfo.NeedsMigration,
			"migrations_applied":   versionInfo.MigrationCount,
			"last_migration":       versionInfo.LastMigration,
			"last_migration_time":  versionInfo.LastMigrationTime,
		}

		// 如果需要迁移，标记为需要注意
		if versionInfo.NeedsMigration {
			overallStatus = "needs_migration"
		}
	}

	httpStatus := http.StatusOK
	if overallStatus == "unhealthy" {
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":    overallStatus,
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"details":   details,
	})
}

// MigrationHealth 迁移状态检查
func (h *HealthHandler) MigrationHealth(c *gin.Context) {
	if h.migrationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unavailable",
			"message": "Migration service not initialized",
		})
		return
	}

	// 获取迁移状态
	migrationStatus, err := h.migrationService.GetMigrationStatus()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("Failed to get migration status: %v", err),
		})
		return
	}

	// 获取最近的迁移历史
	histories, err := h.migrationService.GetMigrationHistory(10)
	if err != nil {
		// 历史记录获取失败不影响整体状态
		fmt.Printf("Warning: Failed to get migration history: %v\n", err)
	} else {
		migrationStatus["recent_history"] = histories
	}

	// 获取迁移统计
	stats := h.migrationService.GetMigrationStatistics()
	migrationStatus["statistics"] = stats

	// 确定状态
	status := "healthy"
	if needsMigration, ok := migrationStatus["needs_migration"].(bool); ok && needsMigration {
		status = "needs_migration"
	}

	migrationStatus["status"] = status
	migrationStatus["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	c.JSON(http.StatusOK, migrationStatus)
}

// SchemaValidation 数据库结构验证
func (h *HealthHandler) SchemaValidation(c *gin.Context) {
	if h.migrationService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status":  "unavailable",
			"message": "Migration service not initialized",
		})
		return
	}

	// 验证数据库结构
	validationResult, err := h.migrationService.ValidateSchema()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"status":  "error",
			"message": fmt.Sprintf("Validation failed: %v", err),
		})
		return
	}

	status := "valid"
	if !validationResult.Valid {
		status = "invalid"
	}

	c.JSON(http.StatusOK, gin.H{
		"status":           status,
		"valid":            validationResult.Valid,
		"critical_issues":  validationResult.CriticalIssues,
		"warnings":         validationResult.WarningIssues,
		"total_issues":     validationResult.TotalIssues,
		"missing_tables":   validationResult.MissingTables,
		"extra_tables":     validationResult.ExtraTables,
		"timestamp":        time.Now().UTC().Format(time.RFC3339),
	})
}
