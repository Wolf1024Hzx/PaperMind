package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/config"
	"wolfden.website/papermind/internal/handler"
	"wolfden.website/papermind/internal/repository"
	"wolfden.website/papermind/internal/service"
)

func main() {
	cfg := config.Load()

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := gorm.Open(postgres.Open(cfg.PostgresDSN()), &gorm.Config{
		TranslateError: true,
	})
	if err != nil {
		log.Fatalf("连接 PostgreSQL 失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("获取 PostgreSQL 底层连接失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		log.Fatalf("PostgreSQL Ping 失败: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.RedisAddr,
		Password: cfg.RedisPassword,
		DB:       cfg.RedisDB,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis Ping 失败: %v", err)
	}

	// 创建 RedisService（带前缀）
	authRedis := service.NewRedisService(redisClient, "papermind:auth:")

	userRepo := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepo, authRedis, cfg.JWTSecret, cfg.JWTTTL)

	healthHandler := handler.NewHealthHandler()
	authHandler := handler.NewAuthHandler(authService)

	router := gin.Default()
	healthHandler.RegisterRoutes(router)

	api := router.Group("/api/v1")
	authHandler.RegisterRoutes(api, authRedis, []byte(cfg.JWTSecret))

	log.Printf("HTTP 服务启动成功，监听地址 %s", cfg.HTTPAddr)
	if err := router.Run(cfg.HTTPAddr); err != nil {
		log.Fatalf("启动 HTTP 服务失败: %v", err)
	}
}
