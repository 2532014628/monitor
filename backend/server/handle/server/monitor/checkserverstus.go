package monitor

import (
	"cmd/server/model"
	"log"
	"time"
)

// 定时检查服务器状态
func CheckServerStatus() {
	db, err := model.InitDB()
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	for {
		time.Sleep(5 * time.Minute)

		// 查找超过 5 分钟没有更新的服务器
		query := `
        UPDATE hostandtoken 
        SET status = 'offline'
        WHERE NOW() - last_heartbeat > INTERVAL '5 minutes' AND status != 'offline'`
		_, err = db.Exec(query)
		if err != nil {
			log.Printf("Failed to update offline status: %v", err)
		} else {
			log.Println("Server status check completed")
		}
	}
}
