package infrastructure

import (
	"fmt"
	"log"
	"os"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDB() (*gorm.DB, error) {
	dsn := buildDSN()

	gormLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond,
			LogLevel:                  logLevel(),
			IgnoreRecordNotFoundError: true,
			Colorful:                  false,
		},
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// buildDSN は環境変数から DSN を組み立てる。
// DATABASE_URL が設定されていればそれを優先する。
// Cloud SQL（Unix socket）は CLOUD_SQL_CONNECTION_NAME が設定されていれば使用する。
// それ以外は個別の環境変数（DB_HOST 等）から構築する。
func buildDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	user     := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname   := getEnv("DB_NAME", "fleamarket")
	timezone := getEnv("DB_TIMEZONE", "Asia/Tokyo")

	// Cloud SQL Auth Proxy (Unix socket) — Cloud Run での推奨接続方式
	if instanceName := os.Getenv("CLOUD_SQL_CONNECTION_NAME"); instanceName != "" {
		socketDir := getEnv("DB_SOCKET_DIR", "/cloudsql")
		return fmt.Sprintf(
			"user=%s password=%s dbname=%s host=%s/%s sslmode=disable TimeZone=%s",
			user, password, dbname, socketDir, instanceName, timezone,
		)
	}

	host     := getEnv("DB_HOST", "localhost")
	port     := getEnv("DB_PORT", "5432")
	sslmode  := getEnv("DB_SSLMODE", "disable")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=%s",
		host, port, user, password, dbname, sslmode, timezone,
	)
}

func logLevel() logger.LogLevel {
	if os.Getenv("APP_ENV") == "production" {
		return logger.Error
	}
	return logger.Info
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
