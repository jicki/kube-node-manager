package handler

import (
	"kube-node-manager/internal/handler/anomaly"
	"kube-node-manager/internal/handler/audit"
	"kube-node-manager/internal/handler/auth"
	"kube-node-manager/internal/handler/cluster"
	"kube-node-manager/internal/handler/feishu"
	"kube-node-manager/internal/handler/gitlab"
	"kube-node-manager/internal/handler/label"
	"kube-node-manager/internal/handler/node"
	"kube-node-manager/internal/handler/progress"
	"kube-node-manager/internal/handler/taint"
	"kube-node-manager/internal/handler/user"
	"kube-node-manager/internal/service"
	"kube-node-manager/pkg/logger"
)

type Handlers struct {
	Auth     *auth.Handler
	User     *user.Handler
	Cluster  *cluster.Handler
	Node     *node.Handler
	Label    *label.Handler
	Taint    *taint.Handler
	Audit    *audit.Handler
	Progress *progress.Handler
	Gitlab   *gitlab.Handler
	Feishu   *feishu.Handler
	Anomaly  *anomaly.Handler
}

func NewHandlers(services *service.Services, logger *logger.Logger) *Handlers {
	return &Handlers{
		Auth:     auth.NewHandler(services.Auth, logger),
		User:     user.NewHandler(services.User, logger),
		Cluster:  cluster.NewHandler(services.Cluster, logger),
		Node:     node.NewHandler(services.Node, logger),
		Label:    label.NewHandler(services.Label, logger),
		Taint:    taint.NewHandler(services.Taint, logger),
		Audit:    audit.NewHandler(services.Audit, logger),
		Progress: progress.NewHandler(services.Progress, logger),
		Gitlab:   gitlab.NewHandler(services.Gitlab, logger),
		Feishu:   feishu.NewHandler(services.Feishu, services.Audit, logger),
		Anomaly:  anomaly.NewHandler(services.Anomaly, services.Anomaly.GetCleanupService(), logger),
	}
}
