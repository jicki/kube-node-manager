package service

import (
	"fmt"
	"os"
	"time"

	"kube-node-manager/internal/cache"
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/realtime"
	"kube-node-manager/internal/service/ansible"
	"kube-node-manager/internal/service/anomaly"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/auth"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/internal/service/feishu"
	"kube-node-manager/internal/service/gitlab"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/label"
	"kube-node-manager/internal/service/ldap"
	"kube-node-manager/internal/service/node"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/internal/service/taint"
	"kube-node-manager/internal/service/user"
	"kube-node-manager/internal/websocket"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

type Services struct {
	Auth          *auth.Service
	User          *user.Service
	Cluster       *cluster.Service
	Node          *node.Service
	Label         *label.Service
	Taint         *taint.Service
	Audit         *audit.Service
	LDAP          *ldap.Service
	K8s           *k8s.Service
	Progress      *progress.Service
	Gitlab        *gitlab.Service
	Feishu        *feishu.Service
	Anomaly       *anomaly.Service
	AnomalyReport *anomaly.ReportService
	Ansible       *ansible.Service    // Ansible 任务服务
	Realtime      *realtime.Manager   // 实时同步管理器
	WSHub         *websocket.Hub      // WebSocket Hub（导出供 handler 使用）
}

// clusterServiceAdapter 适配器，将 cluster.Service 适配为 feishu.ClusterServiceInterface
type clusterServiceAdapter struct {
	svc *cluster.Service
}

func (a *clusterServiceAdapter) List(req interface{}, userID uint) (interface{}, error) {
	listReq, ok := req.(cluster.ListRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}
	return a.svc.List(listReq, userID)
}

// nodeServiceAdapter 适配器，将 node.Service 适配为 feishu.NodeServiceInterface
type nodeServiceAdapter struct {
	svc *node.Service
}

func (a *nodeServiceAdapter) List(req interface{}, userID uint) (interface{}, error) {
	listReq, ok := req.(node.ListRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}
	return a.svc.List(listReq, userID)
}

func (a *nodeServiceAdapter) Get(req interface{}, userID uint) (interface{}, error) {
	getReq, ok := req.(node.GetRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}
	return a.svc.Get(getReq, userID)
}

func (a *nodeServiceAdapter) Cordon(req interface{}, userID uint) error {
	cordonReq, ok := req.(node.CordonRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.Cordon(cordonReq, userID)
}

func (a *nodeServiceAdapter) Uncordon(req interface{}, userID uint) error {
	uncordonReq, ok := req.(node.CordonRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.Uncordon(uncordonReq, userID)
}

// auditServiceAdapter 适配器，将 audit.Service 适配为 feishu.AuditServiceInterface
type auditServiceAdapter struct {
	svc *audit.Service
}

func (a *auditServiceAdapter) List(req interface{}) (interface{}, error) {
	listReq, ok := req.(audit.ListRequest)
	if !ok {
		return nil, fmt.Errorf("invalid request type")
	}
	return a.svc.List(listReq)
}

// labelServiceAdapter 适配器，将 label.Service 适配为 feishu.LabelServiceInterface
type labelServiceAdapter struct {
	svc *label.Service
}

func (a *labelServiceAdapter) UpdateNodeLabels(req interface{}, userID uint) error {
	updateReq, ok := req.(label.UpdateLabelsRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.UpdateNodeLabels(updateReq, userID)
}

func (a *labelServiceAdapter) BatchUpdateLabels(req interface{}, userID uint) error {
	batchReq, ok := req.(label.BatchUpdateRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.BatchUpdateLabels(batchReq, userID)
}

// taintServiceAdapter 适配器，将 taint.Service 适配为 feishu.TaintServiceInterface
type taintServiceAdapter struct {
	svc *taint.Service
}

func (a *taintServiceAdapter) UpdateNodeTaints(req interface{}, userID uint) error {
	updateReq, ok := req.(taint.UpdateTaintsRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.UpdateNodeTaints(updateReq, userID)
}

func (a *taintServiceAdapter) BatchUpdateTaints(req interface{}, userID uint) error {
	batchReq, ok := req.(taint.BatchUpdateRequest)
	if !ok {
		return fmt.Errorf("invalid request type")
	}
	return a.svc.BatchUpdateTaints(batchReq, userID)
}

func (a *taintServiceAdapter) RemoveTaint(clusterName, nodeName, taintKey string, userID uint) error {
	return a.svc.RemoveTaint(clusterName, nodeName, taintKey, userID)
}

// anomalyServiceAdapter 适配器，将 anomaly.Service 适配为 feishu.AnomalyServiceInterface
type anomalyServiceAdapter struct {
	svc *anomaly.Service
}

func (a *anomalyServiceAdapter) GetActiveAnomalies(clusterID *uint) (interface{}, error) {
	return a.svc.GetActiveAnomalies(clusterID)
}

func NewServices(db *gorm.DB, logger *logger.Logger, cfg *config.Config) *Services {
	auditSvc := audit.NewService(db, logger)
	
	// 创建实时同步管理器（必须在 k8s service 之前创建）
	realtimeMgr := realtime.NewManager(logger)
	realtimeMgr.Start()
	logger.Info("Realtime Manager started successfully")
	
	// 创建 K8s 服务
	k8sSvc := k8s.NewService(logger, realtimeMgr)
	
	// 注册 Pod 事件处理器（连接 PodCountCache 到 Informer）
	// 这样 Informer 就能实时更新 Pod 统计数据
	realtimeMgr.RegisterPodEventHandler(k8sSvc.GetPodCountCache())
	logger.Info("Pod event handler registered successfully")
	ldapSvc := ldap.NewService(logger, cfg.LDAP)
	progressSvc := progress.NewService(logger)

	// 检查是否启用数据库模式（用于多副本环境）
	if cfg.Progress.EnableDatabase {
		progressSvc.EnableDatabaseMode(
			db,
			cfg.Progress.NotifyType,
			cfg.Progress.PollInterval,
			cfg.Progress.Redis.Addr,
			cfg.Progress.Redis.Password,
			cfg.Progress.Redis.DB,
		)
		logger.Infof("Progress service database mode enabled for multi-replica support")
	}

	// 创建服务实例
	authSvc := auth.NewService(db, logger, cfg.JWT, ldapSvc, auditSvc)
	labelSvc := label.NewService(db, logger, auditSvc, k8sSvc)
	taintSvc := taint.NewService(db, logger, auditSvc, k8sSvc)
	nodeSvc := node.NewService(logger, k8sSvc, auditSvc)

	// 设置进度服务
	progressSvc.SetAuthService(authSvc)
	labelSvc.SetProgressService(progressSvc)
	taintSvc.SetProgressService(progressSvc)
	nodeSvc.SetProgressService(progressSvc)

	// 创建集群和飞书服务
	clusterSvc := cluster.NewService(db, logger, auditSvc, k8sSvc)
	feishuSvc := feishu.NewService(db, logger)

	// 初始化缓存
	cacheInstance, err := cache.NewCache(&cfg.Monitoring.Cache, db, logger)
	if err != nil {
		logger.Errorf("Failed to initialize cache: %v", err)
		panic(fmt.Sprintf("Failed to initialize cache: %v", err))
	}

	// 创建缓存TTL配置
	cacheTTL := &anomaly.CacheTTL{
		Statistics: time.Duration(cfg.Monitoring.Cache.TTL.Statistics) * time.Second,
		Active:     time.Duration(cfg.Monitoring.Cache.TTL.Active) * time.Second,
		Clusters:   time.Duration(cfg.Monitoring.Cache.TTL.Clusters) * time.Second,
		TypeStats:  time.Duration(cfg.Monitoring.Cache.TTL.TypeStats) * time.Second,
	}

	// 创建数据清理服务
	cleanupConfig := &anomaly.CleanupConfig{
		Enabled:       cfg.Monitoring.Cleanup.Enabled,
		RetentionDays: cfg.Monitoring.Cleanup.RetentionDays,
		CleanupTime:   cfg.Monitoring.Cleanup.CleanupTime,
		BatchSize:     cfg.Monitoring.Cleanup.BatchSize,
	}
	cleanupSvc := anomaly.NewCleanupService(db, logger, cleanupConfig)

	// 创建异常监控服务
	anomalySvc := anomaly.NewService(db, logger, k8sSvc, clusterSvc, cacheInstance, cacheTTL, cleanupSvc, cfg.Monitoring.Enabled, cfg.Monitoring.Interval)

	// 创建适配器并设置飞书服务的依赖
	clusterAdapter := &clusterServiceAdapter{svc: clusterSvc}
	nodeAdapter := &nodeServiceAdapter{svc: nodeSvc}
	auditAdapter := &auditServiceAdapter{svc: auditSvc}
	labelAdapter := &labelServiceAdapter{svc: labelSvc}
	taintAdapter := &taintServiceAdapter{svc: taintSvc}
	anomalyAdapter := &anomalyServiceAdapter{svc: anomalySvc}

	feishuSvc.SetClusterService(clusterAdapter)
	feishuSvc.SetNodeService(nodeAdapter)
	feishuSvc.SetAuditService(auditAdapter)
	feishuSvc.SetLabelService(labelAdapter)
	feishuSvc.SetTaintService(taintAdapter)
	feishuSvc.SetAnomalyService(anomalyAdapter)

	// 创建异常报告服务
	reportEnabled := false
	if cfg.Monitoring.ReportSchedulerEnabled {
		reportEnabled = true
	}
	anomalyReportSvc := anomaly.NewReportService(db, logger, anomalySvc, reportEnabled)

	// 创建 Ansible 服务
	// 从环境变量获取加密密钥
	encryptionKey := os.Getenv("ANSIBLE_ENCRYPTION_KEY")
	ansibleSvc := ansible.NewService(db, logger, k8sSvc, realtimeMgr.GetWebSocketHub(), encryptionKey)

	return &Services{
		Auth:          authSvc,
		User:          user.NewService(db, logger, auditSvc),
		Cluster:       clusterSvc,
		Node:          nodeSvc,
		Label:         labelSvc,
		Taint:         taintSvc,
		Audit:         auditSvc,
		LDAP:          ldapSvc,
		K8s:           k8sSvc,
		Progress:      progressSvc,
		Gitlab:        gitlab.NewService(db, logger),
		Feishu:        feishuSvc,
		Anomaly:       anomalySvc,
		AnomalyReport: anomalyReportSvc,
		Ansible:       ansibleSvc,
		Realtime:      realtimeMgr,
		WSHub:         realtimeMgr.GetWebSocketHub(),
	}
}
