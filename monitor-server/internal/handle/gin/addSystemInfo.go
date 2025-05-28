package gin

import (
	"context"
	"fmt"
	"log"
	"monitor-server/internal/model"
	"monitor-server/internal/repository"
	"net/http"

	"github.com/gin-gonic/gin"
)



func ReceiveAndStoreSystemMetrics(c *gin.Context, repo *repository.RedisRepository) {
	// 解析请求数据
	var requestData repository.RequestData
	if err := c.ShouldBindJSON(&requestData); err != nil {
		s := fmt.Sprintf("Invalid JSON data: %s", err)
		log.Printf("Invalid JSON data: %s", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": s})
		return
	}

	// 将数据插入 Redis
	ctx := context.Background()
	if err := repo.AddSystemInfo(ctx, requestData); err != nil {
		log.Printf("Failed to store system info: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to store system information"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "System information inserted successfully"})
}

// 辅助函数：转换 CPU 信息
func convertCPUInfo(cpuInfo []*model.CPUInfo) []model.CPUInfo {
	result := make([]model.CPUInfo, len(cpuInfo))
	for i, info := range cpuInfo {
		result[i] = *info
	}
	return result
}

// 辅助函数：转换网络信息
func convertNetworkInfo(netInfo []*model.NetworkInfo) []model.NetworkInfo {
	result := make([]model.NetworkInfo, len(netInfo))
	for i, info := range netInfo {
		result[i] = *info
	}
	return result
}
