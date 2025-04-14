package redis

import (
	"context"
	"github.com/go-redis/redis/v8"
	"os"
	"strconv"
)

var Rdb *redis.Client

func InitRedis() error {
	host := os.Getenv("REDIS_HOST")
	port := os.Getenv("REDIS_PORT")
	password := os.Getenv("REDIS_PASSWORD")
	db, _ := strconv.Atoi(os.Getenv("REDIS_DB"))
	addr := host + ":" + port
	Rdb = redis.NewClient(&redis.Options{
		Addr:     addr,     // Redis服务器地址
		Password: password, // 密码
		DB:       db,       // 数据库
	})
	ctx := context.Background()
	_, err := Rdb.Ping(ctx).Result()
	if err != nil {
		return err
	}
	// 启动定时清理任务
	go StartCleanupTask(ctx)
	return nil
}
