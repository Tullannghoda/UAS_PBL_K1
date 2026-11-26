
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort string
	AppEnv  string
	AppRole string

	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
	DBSSLMode  string
	DBMaxConns int32

	MongoURI      string
	MongoDatabase string

	RedisAddr     string
	RedisPassword string
	RedisDB       int

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	LogLevel string
	LogFile  string

	UploadDir        string
	MaxUploadSizeMB  int64
}

func LoadConfig() (*Config, error) {
	// Membaca file .env jika tersedia.
	// Jika tidak tersedia, aplikasi tetap dapat membaca environment variables.
	_ = godotenv.Load()

	cfg := &Config{
		AppPort:     getEnv("APP_PORT", "3000"),
		AppEnv:      getEnv("APP_ENV", "development"),
		AppRole:     getEnv("APP_ROLE", "all"),
		DBHost:      getEnv("DB_HOST", "localhost"),
		DBPort:      getEnv("DB_PORT", "5432"),
		DBUser:      getEnv("DB_USER", "postgres"),
		DBPassword:  getEnv("DB_PASSWORD", ""),
		DBName:      getEnv("DB_NAME", "bakeorder"),
		DBSSLMode:   getEnv("DB_SSLMODE", "disable"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		MongoDatabase: getEnv("MONGO_DATABASE", "bakeorder"),
		RedisAddr:     getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", ""),
		LogLevel:      getEnv("LOG_LEVEL", "debug"),
		LogFile:       getEnv("LOG_FILE", "logs/app.log"),
		UploadDir:     getEnv("UPLOAD_DIR", "uploads"),
	}

	var err error

	cfg.DBMaxConns, err = getInt32("DB_MAX_CONNS", 10)
	if err != nil {
		return nil, err
	}

	cfg.RedisDB, err = getInt("REDIS_DB", 0)
	if err != nil {
		return nil, err
	}

	cfg.MaxUploadSizeMB, err = getInt64("MAX_UPLOAD_SIZE_MB", 5)
	if err != nil {
		return nil, err
	}

	cfg.JWTAccessTTL, err = time.ParseDuration(getEnv("JWT_ACCESS_TTL", "15m"))
	if err != nil {
		return nil, fmt.Errorf("JWT_ACCESS_TTL tidak valid: %w", err)
	}

	cfg.JWTRefreshTTL, err = time.ParseDuration(getEnv("JWT_REFRESH_TTL", "168h"))
	if err != nil {
		return nil, fmt.Errorf("JWT_REFRESH_TTL tidak valid: %w", err)
	}

	if cfg.AppRole != "api" && cfg.AppRole != "worker" && cfg.AppRole != "all" {
		return nil, fmt.Errorf("APP_ROLE harus api, worker, atau all")
	}

	if cfg.DBMaxConns < 1 {
		return nil, fmt.Errorf("DB_MAX_CONNS minimal 1")
	}

	if cfg.MaxUploadSizeMB < 1 {
		return nil, fmt.Errorf("MAX_UPLOAD_SIZE_MB minimal 1")
	}

	if cfg.JWTAccessTTL <= 0 || cfg.JWTRefreshTTL <= 0 {
		return nil, fmt.Errorf("JWT token duration harus lebih dari 0")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func getInt(key string, fallback int) (int, error) {
	value := getEnv(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return parsed, nil
}

func getInt32(key string, fallback int32) (int32, error) {
	value, err := getInt(key, int(fallback))
	if err != nil {
		return 0, err
	}
	return int32(value), nil
}

func getInt64(key string, fallback int64) (int64, error) {
	value := getEnv(key, strconv.FormatInt(fallback, 10))
	parsed, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%s harus berupa angka: %w", key, err)
	}
	return parsed, nil
}
