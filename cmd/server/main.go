package main

import (
	"log"

	"wolfden.website/papermind/internal/app"
	"wolfden.website/papermind/internal/config"
)

func main() {
	cfg := config.Load()

	application := app.New(cfg)
	if err := application.Run(); err != nil {
		log.Fatalf("启动应用失败: %v", err)
	}
}