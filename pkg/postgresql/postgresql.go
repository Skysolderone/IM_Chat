package postgresql

import (
	"fmt"
	"log"
	"time"

	"wsim/pkg/config"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func GetDB() *gorm.DB {
	if db == nil {
		InitPostgreSQL()
	}
	return db
}

func InitPostgreSQL() {
	var err error
	log.Println("Initializing PostgreSQL...")

	cfg := config.GetDatabaseConfig()
	host := cfg.Database.Host
	port := cfg.Database.Port
	user := cfg.Database.User
	password := cfg.Database.Password
	dbname := cfg.Database.DBName
	sslmode := cfg.Database.SSLMode
	timezone := cfg.Database.Timezone

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s timezone=%s", host, port, user, password, dbname, sslmode, timezone)
	if dsn == "" {
		log.Fatal("PG_DSN is not set")
	}

	// https://github.com/go-gorm/postgres
	db, err = gorm.Open(postgres.New(postgres.Config{
		DSN:                  dsn,
		PreferSimpleProtocol: true, // disables implicit prepared statement usage
	}), &gorm.Config{
		SkipDefaultTransaction: true,
		PrepareStmt:            true,
		PrepareStmtMaxSize:     1000,
		PrepareStmtTTL:         10 * time.Minute,
	})
	if err != nil {
		log.Fatal("Failed to connect to PostgreSQL: ", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get SQL DB: ", err)
	}

	// 连接池参数从配置文件读取
	maxOpen := cfg.Database.MaxOpenConns
	if maxOpen <= 0 {
		maxOpen = 25
	}
	if maxOpen > 10000 {
		maxOpen = 10000
	}

	maxIdle := cfg.Database.MaxIdleConns
	if maxIdle <= 0 {
		maxIdle = 10
	}
	if maxIdle > maxOpen {
		maxIdle = maxOpen
	}

	connMaxLifetime := cfg.Database.ConnMaxLifetime
	if connMaxLifetime <= 0 {
		connMaxLifetime = 30 * time.Minute
	}

	connMaxIdleTime := cfg.Database.ConnMaxIdleTime
	if connMaxIdleTime <= 0 {
		connMaxIdleTime = 5 * time.Minute
	}

	// MaxOpenConns 应结合 DB 端 max_connections、应用实例数、以及业务并发来评估。
	sqlDB.SetMaxOpenConns(maxOpen)
	// MaxIdleConns 通常不应大于 MaxOpenConns；太小会导致频繁建连，太大会浪费资源。
	sqlDB.SetMaxIdleConns(maxIdle)
	// 连接生命周期：常用 30m~1h；建议略小于负载均衡/代理的 idle timeout，减少被动断链带来的错误。
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	// 空闲超时：建议设置，避免长期空闲连接占用资源；常用 1m~10m。
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
}
