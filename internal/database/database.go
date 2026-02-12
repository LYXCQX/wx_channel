package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/glebarez/sqlite"
)

// DB 是全局数据库实例
var (
	db   *sql.DB
	once sync.Once
)

// Config 包含数据库配置
type Config struct {
	DBPath string
}

// Initialize 初始化数据库连接并运行迁移
func Initialize(cfg *Config) error {
	var initErr error
	once.Do(func() {
		// 确保目录存在
		dir := filepath.Dir(cfg.DBPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			initErr = fmt.Errorf("failed to create database directory: %w", err)
			return
		}

		// 打开数据库连接
		var err error
		db, err = sql.Open("sqlite", cfg.DBPath+"?_foreign_keys=on&_journal_mode=WAL")
		if err != nil {
			initErr = fmt.Errorf("failed to open database: %w", err)
			return
		}

		// 测试连接
		if err := db.Ping(); err != nil {
			initErr = fmt.Errorf("failed to ping database: %w", err)
			return
		}

		// 运行迁移
		if err := runMigrations(); err != nil {
			initErr = fmt.Errorf("failed to run migrations: %w", err)
			return
		}
	})
	return initErr
}

// GetDB 返回数据库实例
func GetDB() *sql.DB {
	return db
}

// Close 关闭数据库连接
func Close() error {
	if db != nil {
		return db.Close()
	}
	return nil
}
