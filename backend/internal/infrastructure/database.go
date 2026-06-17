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
// DATABASE_URL が設定されていればそれを優先し、
// なければ個別の環境変数（DB_HOST 等）から構築する。
func buildDSN() string {
	if url := os.Getenv("DATABASE_URL"); url != "" {
		return url
	}

	host     := getEnv("DB_HOST", "localhost")
	port     := getEnv("DB_PORT", "5432")
	user     := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "")
	dbname   := getEnv("DB_NAME", "fleamarket")
	sslmode  := getEnv("DB_SSLMODE", "disable")
	timezone := getEnv("DB_TIMEZONE", "Asia/Tokyo")

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
