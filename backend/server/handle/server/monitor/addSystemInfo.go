package monitor

import (
	"backend/server/model"
	"backend/server/redis"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// RequestData 用于接收系统监控数据的请求体
// @Description RequestData 包含所有需要收集的系统信息
type RequestData struct {
	CPUInfo  []model.CPUInfo     `json:"cpu_info"`  // CPU 信息
	HostInfo model.HostInfo      `json:"host_info"` // 主机信息
	MemInfo  model.MemoryInfo    `json:"mem_info"`  // 内存信息
	ProInfo  []model.ProcessInfo `json:"pro_info"`  // 进程信息
	NetInfo  []model.NetworkInfo `json:"net_info"`  // 网络信息
}

// AddSystemInfo 接收并处理系统监控数据
//
// @Summary 接收系统监控信息（CPU、内存、主机信息等）
// @Description 该API用于接收客户端发送的系统监控数据，并验证token和JWT后将数据存储到数据库中。
// @Tags Monitor
// @Accept json
// @Produce json
// @Param request body RequestData true "请求体包含系统监控数据"
// @Success 201 {object} map[string]string "成功响应"
// @Failure 400 {object} map[string]string "无效的JSON数据或令牌长度错误"
// @Failure 401 {object} map[string]string "授权头缺失或无效的token格式或无效的JWT token"
// @Failure 500 {object} map[string]string "数据库操作失败"
// @Router /monitor [post]
func ReceiveAndStoreSystemMetrics(c *gin.Context) {
	// 解析请求数据
	var requestData RequestData
	if err := c.ShouldBindJSON(&requestData); err != nil {
		s := fmt.Sprintf("Invalid JSON data: %s", err)
		log.Printf("Invalid JSON data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": s})
		return
	}

	// 将数据插入 Redis
	ctx := context.Background()
	timestamp := time.Now().Unix() // 获取当前时间戳
	key := fmt.Sprintf("system_info:%s:%d", requestData.HostInfo.Hostname, timestamp)
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		log.Printf("Failed to marshal data to JSON: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to marshal data to JSON"})
		return
	}

	// 将 JSON 字符串存储到 Redis
	err = redis.Rdb.Set(ctx, key, jsonData, 30*time.Minute).Err()
	if err != nil {
		log.Printf("Failed to insert data into Redis: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to insert data into Redis"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "System information inserted successfully"})
}
