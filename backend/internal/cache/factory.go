package cache

import (
	"fmt"

	"kube-node-manager/internal/config"
	"kube-node-manager/pkg/logger"

	"gorm.io/gorm"
)

// NewCache 缓存工厂方法
func NewCache(cfg *config.CacheConfig, db *gorm.DB, log *logger.Logger) (Cache, error) {
	if cfg == nil || !cfg.Enabled {
		log.Infof("Cache is disabled, using NoCache")
		return NewNoCache(), nil
	}

	switch cfg.Type {
	case "postgres", "postgresql":
		log.Infof("Initializing PostgreSQL cache")
		return NewPostgresCache(
			db,
			cfg.Postgres.TableName,
			cfg.Postgres.CleanupInterval,
			cfg.Postgres.UseUnlogged,
			log,
		)

	case "memory":
		log.Warningf("Initializing Memory cache - NOT recommended for multi-replica deployment")
		return NewMemoryCache(log), nil

	case "none", "":
		log.Infof("Cache type set to 'none', using NoCache")
		return NewNoCache(), nil

	default:
		return nil, fmt.Errorf("unsupported cache type: %s (supported: postgres, memory, none)", cfg.Type)
	}
}
