package redis

import (
	"backend/server/model"
	"context"
	"fmt"

	"log"
	"time"
)

func StartCleanupTask(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute) // 每 5 分钟执行一次
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			cleanupAndPersistData(ctx)
		case <-ctx.Done():
			log.Println("Cleanup task stopped")
			return
		}
	}
}

func cleanupAndPersistData(ctx context.Context) {
	var cursor uint64
	var keys []string
	var err error

	for {
		// 使用 SCAN 命令遍历所有键
		keys, cursor, err = Rdb.Scan(ctx, cursor, "system_info:*", 100).Result()
		if err != nil {
			log.Printf("Error scanning keys: %v\n", err)
			return
		}

		// 处理每个键
		for _, key := range keys {
			// 获取键的剩余生存时间
			ttl, err := Rdb.TTL(ctx, key).Result()
			if err != nil {
				log.Printf("Error getting TTL for key %s: %v\n", key, err)
				continue
			}

			// 如果键即将过期（例如剩余时间小于 1 分钟），则写入数据库并删除
			if ttl <= time.Minute {
				var requestData model.RequestData
				err := Rdb.Get(ctx, key).Scan(&requestData)
				if err != nil {
					log.Printf("Error getting data for key %s: %v\n", key, err)
					continue
				}

				// 将数据写入数据库
				if err := saveToDatabase(requestData); err != nil {
					log.Printf("Error saving data to database for key %s: %v\n", key, err)
					continue
				}

				// 删除 Redis 中的键
				if _, err := Rdb.Del(ctx, key).Result(); err != nil {
					log.Printf("Error deleting key %s: %v\n", key, err)
				} else {
					log.Printf("Key %s saved to database and deleted from Redis\n", key)
				}
			}
		}

		// 如果遍历完成，退出循环
		if cursor == 0 {
			break
		}
	}
}

// 模拟将数据存入数据库的函数
func saveToDatabase(data model.RequestData) error {
	// 这里实现将数据存入数据库的逻辑
	fmt.Printf("Saving data for host %s to database\n", data.HostInfo.Hostname)
	return nil
}
