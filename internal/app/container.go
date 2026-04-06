package app

import (
	"context"
	"log"
	"os"
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

// Container 依赖容器，持有所有初始化好的组件
type Container struct {
	db          *gorm.DB
	redisClient *redis.Client

	// Repositories
	userRepo  *repository.UserRepository
	paperRepo *repository.PaperRepository
	chunkRepo *repository.ChunkRepository

	// Services
	authRedis     *service.RedisService
	authService   *service.AuthService
	paperService  *service.PaperService
	embeddingClient service.EmbeddingClient

	// Handlers
	healthHandler *handler.HealthHandler
	authHandler   *handler.AuthHandler
	paperHandler  *handler.PaperHandler

	// Config
	config *config.Config
}

// NewContainer 初始化所有依赖
func NewContainer(cfg config.Config) *Container {
	c := &Container{config: &cfg}

	// 1. 初始化基础设施
	c.initDatabase()
	c.initRedis()

	// 2. 初始化 Repositories
	c.initRepositories()

	// 3. 初始化 Services
	c.initServices()

	// 4. 初始化 Handlers
	c.initHandlers()

	return c
}

// initDatabase 初始化数据库连接
func (c *Container) initDatabase() {
	db, err := gorm.Open(postgres.Open(c.config.PostgresDSN()), &gorm.Config{
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

	c.db = db
}

// initRedis 初始化 Redis 连接
func (c *Container) initRedis() {
	c.redisClient = redis.NewClient(&redis.Options{
		Addr:     c.config.RedisAddr,
		Password: c.config.RedisPassword,
		DB:       c.config.RedisDB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Redis Ping 失败: %v", err)
	}
}

// initRepositories 初始化所有 Repository
func (c *Container) initRepositories() {
	c.userRepo = repository.NewUserRepository(c.db)
	c.paperRepo = repository.NewPaperRepository(c.db)
	c.chunkRepo = repository.NewChunkRepository(c.db)
}

// initServices 初始化所有 Service
func (c *Container) initServices() {
	// Redis Service（带前缀）
	c.authRedis = service.NewRedisService(c.redisClient, "papermind:auth:")

	// Auth Service
	c.authService = service.NewAuthService(c.userRepo, c.authRedis, c.config.JWTSecret, c.config.JWTTTL)

	// Embedding Client（暂时使用 Mock）
	c.embeddingClient = service.NewMockEmbeddingClient(1024)

	// 确保上传目录存在
	if err := os.MkdirAll(c.config.UploadDir, 0755); err != nil {
		log.Fatalf("创建上传目录失败: %v", err)
	}

	// Paper Service
	c.paperService = service.NewPaperService(
		c.paperRepo,
		c.chunkRepo,
		c.embeddingClient,
		c.config.UploadDir,
		c.config.EmbeddingBatchSize,
		c.config.EmbeddingMaxConcurrency,
	)
}

// initHandlers 初始化所有 Handler
func (c *Container) initHandlers() {
	c.healthHandler = handler.NewHealthHandler()
	c.authHandler = handler.NewAuthHandler(c.authService)
	c.paperHandler = handler.NewPaperHandler(c.paperService)
}

// RegisterRoutes 注册所有路由
func (c *Container) RegisterRoutes(router *gin.Engine) {
	c.healthHandler.RegisterRoutes(router)

	api := router.Group("/api/v1")
	c.authHandler.RegisterRoutes(api, c.authRedis, []byte(c.config.JWTSecret), c.config.JWTTTL)
	c.paperHandler.RegisterRoutes(api, c.authRedis, []byte(c.config.JWTSecret), c.config.JWTTTL)
}

// Close 关闭所有连接
func (c *Container) Close() {
	if c.redisClient != nil {
		c.redisClient.Close()
	}

	if c.db != nil {
		sqlDB, err := c.db.DB()
		if err == nil {
			sqlDB.Close()
		}
	}
}