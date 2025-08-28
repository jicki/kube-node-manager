package database

import (
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDatabase(dsn string) (*gorm.DB, error) {
	dir := filepath.Dir(dsn)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// 添加SQLite配置参数以优化内存使用
	dsn = dsn + "?cache=shared&mode=rwc&_journal_mode=WAL&_cache_size=1000&_temp_store=memory"

	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return nil, err
	}

	// 设置SQLite运行时配置
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	// 设置连接池参数
	sqlDB.SetMaxOpenConns(1) // SQLite只支持单个写连接
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}
