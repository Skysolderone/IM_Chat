package postgresql

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
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
	err = godotenv.Load()
	if err != nil {
		log.Fatal("Failed to load .env file: ", err)
	}
	host := os.Getenv("host")
	port := os.Getenv("port")
	user := os.Getenv("user")
	password := os.Getenv("password")
	dbname := os.Getenv("dbname")
	sslmode := os.Getenv("sslmode")
	timezone := os.Getenv("timezone")

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

	// 连接池参数建议“默认值 + 可配置覆盖”，避免不同环境下写死不合适。
	// 环境变量：
	// - PG_MAX_OPEN_CONNS: 最大打开连接数（int）
	// - PG_MAX_IDLE_CONNS: 最大空闲连接数（int，需 <= open）
	// - PG_CONN_MAX_LIFETIME: 连接最大生命周期（time.ParseDuration，如 30m/1h；0 表示不限制）
	// - PG_CONN_MAX_IDLE_TIME: 连接最大空闲时间（time.ParseDuration，如 5m；0 表示不限制）
	maxOpen := envInt("PG_MAX_OPEN_CONNS", 25, 1, 10000)
	maxIdleDefault := 10
	if maxIdleDefault > maxOpen {
		maxIdleDefault = maxOpen
	}
	maxIdle := envInt("PG_MAX_IDLE_CONNS", maxIdleDefault, 0, maxOpen)
	connMaxLifetime := envDuration("PG_CONN_MAX_LIFETIME", 30*time.Minute)
	connMaxIdleTime := envDuration("PG_CONN_MAX_IDLE_TIME", 5*time.Minute)

	// MaxOpenConns 应结合 DB 端 max_connections、应用实例数、以及业务并发来评估。
	sqlDB.SetMaxOpenConns(maxOpen)
	// MaxIdleConns 通常不应大于 MaxOpenConns；太小会导致频繁建连，太大会浪费资源。
	sqlDB.SetMaxIdleConns(maxIdle)
	// 连接生命周期：常用 30m~1h；建议略小于负载均衡/代理的 idle timeout，减少被动断链带来的错误。
	sqlDB.SetConnMaxLifetime(connMaxLifetime)
	// 空闲超时：建议设置，避免长期空闲连接占用资源；常用 1m~10m。
	sqlDB.SetConnMaxIdleTime(connMaxIdleTime)
}

func envInt(key string, def, min, max int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}

func envDuration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	if d < 0 {
		return def
	}
	return d
}
