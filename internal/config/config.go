package config

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                  string
	HTTPAddr                string
	JWTSecret               string
	JWTTTL                  time.Duration
	PostgresHost            string
	PostgresPort            string
	PostgresUser            string
	PostgresPassword        string
	PostgresDB              string
	RedisAddr               string
	RedisPassword           string
	RedisDB                 int
	UploadDir               string
	AliYunAPIKey            string // 阿里云 DashScope API Key
	EmbeddingType           string // Embedding 类型：mock / qwen
	EmbeddingModel          string // Embedding 模型名称
	EmbeddingBatchSize      int    // Embedding API 单次请求最大条数
	EmbeddingMaxConcurrency int    // Embedding API 最大并发请求数
}

func Load() Config {
	// 加载 .env 文件（如果存在，不存在也不报错）
	godotenv.Load()

	return Config{
		AppEnv:                  getEnv("APP_ENV", "development"),
		HTTPAddr:                getEnv("HTTP_ADDR", ":8080"),
		JWTSecret:               getEnv("JWT_SECRET", "papermind-dev-secret"),
		JWTTTL:                  24 * time.Hour,
		PostgresHost:            getEnv("PG_HOST", "127.0.0.1"),
		PostgresPort:            getEnv("PG_PORT", "5432"),
		PostgresUser:            getEnv("PG_USER", "wolf"),
		PostgresPassword:        getEnv("PG_PASSWORD", "Wolf1024!"),
		PostgresDB:              getEnv("PG_DATABASE", "paper_mind"),
		RedisAddr:               getEnv("REDIS_ADDR", "127.0.0.1:6379"),
		RedisPassword:           getEnv("REDIS_PASSWORD", "Wolf1024!"),
		RedisDB:                 0,
		UploadDir:               getEnv("UPLOAD_DIR", "./uploads/papers"),
		AliYunAPIKey:            getEnv("ALIYUN_API_KEY", ""),
		EmbeddingType:           getEnv("EMBEDDING_TYPE", "mock"),
		EmbeddingModel:          getEnv("EMBEDDING_MODEL", "qwen3-vl-embedding"),
		EmbeddingBatchSize:      getEnvInt("EMBEDDING_BATCH_SIZE", 20),
		EmbeddingMaxConcurrency: getEnvInt("EMBEDDING_MAX_CONCURRENCY", 4),
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

func getEnvInt(key string, fallback int) int {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	// 尝试转换为整数，失败则返回默认值
	var result int
	if _, err := fmt.Sscanf(value, "%d", &result); err != nil {
		return fallback
	}
	return result
}
