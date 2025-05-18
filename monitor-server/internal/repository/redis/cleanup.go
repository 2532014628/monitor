package redis

import (
	"context"
	"fmt"
	"log"
	"time"

	"monitor-server/internal/repository"
)

// StartCleanupTask 启动定时清理任务
func StartCleanupTask(ctx context.Context, redisRepo *repository.RedisRepository, tdengineRepo *repository.TDengineRepository) {
	ticker := time.NewTicker(5 * time.Minute) // 每 5 分钟执行一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := cleanupAndPersistData(ctx, redisRepo, tdengineRepo); err != nil {
				log.Printf("Failed to cleanup and persist data: %v", err)
			}
		case <-ctx.Done():
			log.Println("Cleanup task stopped")
			return
		}
	}
}

// cleanupAndPersistData 清理 Redis 中的数据并持久化到 TDengine
func cleanupAndPersistData(ctx context.Context, redisRepo *repository.RedisRepository, tdengineRepo *repository.TDengineRepository) error {
	var cursor uint64
	var keys []string
	var err error

	for {
		// 使用 SCAN 命令遍历所有键
		keys, cursor, err = redisRepo.Scan(ctx, cursor, "system_info:*", 100)
		if err != nil {
			return fmt.Errorf("error scanning keys: %v", err)
		}

		// 处理每个键
		for _, key := range keys {
			// 获取键的剩余生存时间
			ttl, err := redisRepo.TTL(ctx, key)
			if err != nil {
				log.Printf("Error getting TTL for key %s: %v", key, err)
				continue
			}

			// 如果键即将过期（例如剩余时间小于 1 分钟），则写入数据库并删除
			if ttl <= time.Minute {
				var systemInfo repository.RequestData
				err := redisRepo.Get(ctx, key, &systemInfo)
				if err != nil {
					log.Printf("Error getting data for key %s: %v", key, err)
					continue
				}

				// 将数据写入 TDengine
				if err := tdengineRepo.InsertSystemInfo(
					systemInfo.HostInfo.Hostname,
					systemInfo.HostInfo,
					systemInfo.CPUInfo,
					systemInfo.MemInfo,
					systemInfo.NetInfo,
				); err != nil {
					log.Printf("Error saving data to TDengine for key %s: %v", key, err)
					continue
				}

				// 删除 Redis 中的键
				if err := redisRepo.Delete(ctx, key); err != nil {
					log.Printf("Error deleting key %s: %v", key, err)
				} else {
					log.Printf("Key %s saved to TDengine and deleted from Redis", key)
				}
			}
		}

		// 如果遍历完成，退出循环
		if cursor == 0 {
			break
		}
	}

	return nil
}
