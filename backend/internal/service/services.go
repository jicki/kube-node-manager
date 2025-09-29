package service

import (
	"kube-node-manager/internal/config"
	"kube-node-manager/internal/service/audit"
	"kube-node-manager/internal/service/auth"
	"kube-node-manager/internal/service/cluster"
	"kube-node-manager/internal/service/k8s"
	"kube-node-manager/internal/service/label"
	"kube-node-manager/internal/service/ldap"
	"kube-node-manager/internal/service/node"
	"kube-node-manager/internal/service/progress"
	"kube-node-manager/internal/service/taint"
	"kube-node-manager/internal/service/user"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

type Services struct {
	Auth     *auth.Service
	User     *user.Service
	Cluster  *cluster.Service
	Node     *node.Service
	Label    *label.Service
	Taint    *taint.Service
	Audit    *audit.Service
	LDAP     *ldap.Service
	K8s      *k8s.Service
	Progress *progress.Service
}

func NewServices(db *gorm.DB, logger *logger.Logger, cfg *config.Config) *Services {
	auditSvc := audit.NewService(db, logger)
	k8sSvc := k8s.NewService(logger)
	ldapSvc := ldap.NewService(logger, cfg.LDAP)
	progressSvc := progress.NewService(logger)

	// 检查是否启用数据库模式（用于多副本环境）
	if cfg.Progress.EnableDatabase {
		progressSvc.EnableDatabaseMode(db)
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

	return &Services{
		Auth:     authSvc,
		User:     user.NewService(db, logger, auditSvc),
		Cluster:  cluster.NewService(db, logger, auditSvc, k8sSvc),
		Node:     nodeSvc,
		Label:    labelSvc,
		Taint:    taintSvc,
		Audit:    auditSvc,
		LDAP:     ldapSvc,
		K8s:      k8sSvc,
		Progress: progressSvc,
	}
}
