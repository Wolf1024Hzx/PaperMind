package app

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"wolfden.website/papermind/internal/config"
)

// App 应用主体，持有所有依赖
type App struct {
	config     *config.Config
	router     *gin.Engine
	container  *Container
	httpServer *http.Server
}

// New 创建应用实例
func New(cfg config.Config) *App {
	// 生产环境设置 Release 模式
	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化依赖容器
	container := NewContainer(cfg)

	// 创建路由
	router := gin.Default()

	// 注册路由
	container.RegisterRoutes(router)

	return &App{
		config:    &cfg,
		router:    router,
		container: container,
	}
}

// Run 启动应用，监听信号实现优雅关闭
func (a *App) Run() error {
	a.httpServer = &http.Server{
		Addr:    a.config.HTTPAddr,
		Handler: a.router,
	}

	// 启动 HTTP 服务（非阻塞）
	go func() {
		log.Printf("HTTP 服务启动成功，监听地址 %s", a.config.HTTPAddr)
		if err := a.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动 HTTP 服务失败: %v", err)
		}
	}()

	// 等待中断信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务...")

	// 优雅关闭，最多等待 5 秒
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := a.httpServer.Shutdown(ctx); err != nil {
		log.Printf("HTTP 服务关闭错误: %v", err)
		return err
	}

	// 关闭数据库连接
	a.container.Close()

	log.Println("服务已关闭")
	return nil
}
