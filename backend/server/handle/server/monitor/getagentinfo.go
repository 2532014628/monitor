package monitor

import (
	"backend/server/model"
	"backend/server/redis"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

func GetAgentInfo(c *gin.Context) {
	hostname := c.Param("hostname")
	if len(hostname) == 0 {
		log.Printf("名字出错！")
		c.JSON(http.StatusBadRequest, gin.H{"error": "主机名不能为空"})
		return
	}

	queryType := c.DefaultQuery("type", "all")
	from := c.Query("from")
	to := c.Query("to")

	if from == "" {
		from = "1970-01-01T00:00:00Z"
	}
	if to == "" {
		to = "9999-12-31T23:59:59Z"
	}

	// 解析时间范围
	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'from' time format"})
		return
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid 'to' time format"})
		return
	}

	// 从 Redis 中查询指定时间段的数据
	ctx := context.Background()
	redisData := make([]RequestData, 0)
	var cursor uint64
	for {
		keys, nextCursor, err := redis.Rdb.Scan(ctx, cursor, fmt.Sprintf("system_info:%s:*", hostname), 100).Result()
		if err != nil {
			log.Printf("Error scanning Redis keys: %v\n", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan Redis keys"})
			return
		}

		// 遍历键，过滤出在时间范围内的数据
		for _, key := range keys {
			timestampStr := key[len(fmt.Sprintf("system_info:%s:", hostname)):]
			timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				log.Printf("Error parsing timestamp from key %s: %v\n", key, err)
				continue
			}

			// 检查时间是否在范围内
			dataTime := time.Unix(timestamp, 0)
			if dataTime.After(fromTime) && dataTime.Before(toTime) {
				var requestData RequestData
				err := redis.Rdb.Get(ctx, key).Scan(&requestData)
				if err != nil {
					log.Printf("Error getting data from Redis for key %s: %v\n", key, err)
					continue
				}
				redisData = append(redisData, requestData)
			}
		}

		// 如果遍历完成，退出循环
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}

	// 如果 Redis 中的数据覆盖了整个时间段，则直接返回
	if len(redisData) > 0 && isTimeRangeCovered(redisData, fromTime, toTime) {
		c.JSON(http.StatusOK, redisData)
		return
	}

	// 如果 Redis 中的数据不完整，则从数据库中查询缺失的部分
	dbData, err := model.ReadDB(queryType, from, to, hostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Printf("error:%f", err)
		return
	}

	// 合并 Redis 和数据库中的数据
	mergedData := mergeData(redisData, dbData)
	c.JSON(http.StatusOK, mergedData)
}

// 检查 Redis 中的数据是否覆盖了整个时间段
func isTimeRangeCovered(data []RequestData, from, to time.Time) bool {
	if len(data) == 0 {
		return false
	}

	// 检查数据的时间范围是否覆盖了 from 到 to
	firstDataTime := data[0].HostInfo.CreatedAt
	lastDataTime := data[len(data)-1].HostInfo.CreatedAt
	return firstDataTime.Before(from) && lastDataTime.After(to)
}

// 合并 Redis 和数据库中的数据
func mergeData(redisData []RequestData, dbData map[string]interface{}) []RequestData {
	// 将数据库中的数据转换为 RequestData 格式
	var mergedData []RequestData
	for _, data := range redisData {
		mergedData = append(mergedData, data)
	}

	// 添加数据库中的数据
	if dbData != nil {
		// 这里假设 dbData 是一个包含 RequestData 的 map
		// 根据实际数据结构进行调整
		for _, value := range dbData {
			if requestData, ok := value.(RequestData); ok {
				mergedData = append(mergedData, requestData)
			}
		}
	}

	return mergedData
}
