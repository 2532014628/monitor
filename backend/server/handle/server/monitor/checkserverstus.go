package monitor

import (
	"backend/server/model"
	"backend/server/redis"
	"context"
	"database/sql"
	"log"
	"time"
)

func CheckServerStatus() {
	ctx := context.Background()

	// 初始化数据库
	db, _, err := model.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	// 定时任务
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			checkAndUpdateStatus(ctx, db)
		}
	}
}

// 检查并更新状态
func checkAndUpdateStatus(ctx context.Context, db *sql.DB) {
	// 获取所有主机的键
	var cursor uint64
	var keys []string
	var err error

	for {
		// 使用 SCAN 命令遍历所有主机键
		keys, cursor, err = redis.Rdb.Scan(ctx, cursor, "host:*", 100).Result()
		if err != nil {
			log.Printf("Error scanning Redis keys: %v\n", err)
			return
		}

		// 检查每个主机的最后更新时间
		for _, key := range keys {
			// 获取主机的最后更新时间
			lastUpdatedStr, err := redis.Rdb.HGet(ctx, key, "last_updated").Result()
			if err != nil {
				log.Printf("Error getting last_updated for key %s: %v\n", key, err)
				continue
			}

			// 解析时间
			lastUpdated, err := time.Parse(time.RFC3339, lastUpdatedStr)
			if err != nil {
				log.Printf("Error parsing last_updated for key %s: %v\n", key, err)
				continue
			}

			// 检查是否超过 5 分钟未更新
			if time.Since(lastUpdated) > 5*time.Minute {
				// 更新 hostandtoken 表
				hostname := key[len("host:"):] // 提取主机名
				query := `
                UPDATE hostandtoken 
                SET status = 'offline', last_heartbeat = $1
                WHERE host_name = $2`
				_, err := db.Exec(query, lastUpdated, hostname)
				if err != nil {
					log.Printf("Failed to update status for host %s: %v\n", hostname, err)
				} else {
					log.Printf("Updated status for host %s to offline\n", hostname)
				}
			}
		}

		// 如果遍历完成，退出循环
		if cursor == 0 {
			break
		}
	}
}
