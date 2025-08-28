package database

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func InitDatabase(dsn string) (*gorm.DB, error) {
	dir := filepath.Dir(dsn)
	log.Printf("Creating database directory: %s", dir)
	
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create database directory: %v", err)
	}

	// 检查目录权限
	if stat, err := os.Stat(dir); err != nil {
		return nil, fmt.Errorf("failed to check directory stats: %v", err)
	} else {
		log.Printf("Database directory permissions: %v", stat.Mode())
	}

	// 尝试创建测试文件验证权限
	testFile := filepath.Join(dir, "test.tmp")
	if f, err := os.Create(testFile); err != nil {
		log.Printf("Cannot write to database directory: %v", err)
		log.Println("Falling back to memory database")
		
		// 直接使用内存数据库
		db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize memory database: %v", err)
		}
		
		sqlDB, _ := db.DB()
		sqlDB.SetMaxOpenConns(1)
		sqlDB.SetMaxIdleConns(1)
		
		log.Println("Using in-memory database (data will not persist)")
		return db, nil
	} else {
		f.Close()
		os.Remove(testFile)
	}

	// 尝试文件数据库
	log.Printf("Attempting to open database file: %s", dsn)
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		log.Printf("Failed to open file database: %v", err)
		log.Println("Falling back to memory database")
		
		// 回退到内存数据库
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Silent),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize memory database: %v", err)
		}
		
		log.Println("Using in-memory database (data will not persist)")
	} else {
		log.Printf("Successfully opened file database: %s", dsn)
	}

	// 设置连接池参数
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	
	sqlDB.SetMaxOpenConns(1)
	sqlDB.SetMaxIdleConns(1)

	return db, nil
}
