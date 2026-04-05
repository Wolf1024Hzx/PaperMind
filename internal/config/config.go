package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	AppEnv           string
	HTTPAddr         string
	JWTSecret        string
	JWTTTL           time.Duration
	PostgresHost     string
	PostgresPort     string
	PostgresUser     string
	PostgresPassword string
	PostgresDB       string
	RedisAddr        string
	RedisPassword    string
	RedisDB          int
	UploadDir        string
}

func Load() Config {
	return Config{
		AppEnv:           getEnv("APP_ENV", "development"),
		HTTPAddr:         getEnv("HTTP_ADDR", ":8080"),
		JWTSecret:        getEnv("JWT_SECRET", "papermind-dev-secret"),
		JWTTTL:           24 * time.Hour,
		PostgresHost:     getEnv("PG_HOST", "127.0.0.1"),
		PostgresPort:     getEnv("PG_PORT", "5432"),
		PostgresUser:     getEnv("PG_USER", "wolf"),
		PostgresPassword: getEnv("PG_PASSWORD", "Wolf1024!"),
		PostgresDB:       getEnv("PG_DATABASE", "paper_mind"),
		RedisAddr:        getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:    getEnv("REDIS_PASSWORD", "Wolf1024!"),
		RedisDB:          0,
		UploadDir:        getEnv("UPLOAD_DIR", "./uploads/papers"),
	}
}

func (c Config) PostgresDSN() string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		c.PostgresHost,
		c.PostgresPort,
		c.PostgresUser,
		c.PostgresPassword,
		c.PostgresDB,
	)
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
