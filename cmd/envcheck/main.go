package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pgHost := "127.0.0.1"
	pgPort := "5432"
	pgUser := "wolf"
	pgPassword := "Wolf1024!"
	pgDatabase := "paper_mind"

	redisAddr := "127.0.0.1:6379"
	redisPassword := "Wolf1024!"
	redisDB := 0

	log.Printf("开始检查 PostgreSQL: host=%s port=%s user=%s db=%s", pgHost, pgPort, pgUser, pgDatabase)
	log.Printf("开始检查 Redis: addr=%s db=%d", redisAddr, redisDB)

	hasError := false

	if err := checkPostgres(ctx, pgHost, pgPort, pgUser, pgPassword, pgDatabase); err != nil {
		log.Printf("PostgreSQL 检查失败: %v", err)
		hasError = true
	} else {
		log.Println("PostgreSQL 连接成功")
	}

	if err := checkRedis(ctx, redisAddr, redisPassword, redisDB); err != nil {
		log.Printf("Redis 检查失败: %v", err)
		hasError = true
	} else {
		log.Println("Redis 连接成功")
	}

	if hasError {
		os.Exit(1)
	}
}

func checkPostgres(ctx context.Context, host, port, user, password, database string) error {
	conn, err := pgx.Connect(ctx, fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host,
		port,
		user,
		password,
		database,
	))
	if err != nil {
		return err
	}
	defer conn.Close(ctx)

	var result int
	if err := conn.QueryRow(ctx, "select 1").Scan(&result); err != nil {
		return err
	}

	if result != 1 {
		return fmt.Errorf("unexpected result: %d", result)
	}

	return nil
}

func checkRedis(ctx context.Context, addr, password string, db int) error {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})
	defer client.Close()

	return client.Ping(ctx).Err()
}
