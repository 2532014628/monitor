package gin

import (
	"monitor-server/internal/repository"

	"github.com/gin-gonic/gin"
)

// SetupRouter 设置路由
func SetupRouter(repo *repository.RedisRepository) *gin.Engine {
	r := gin.Default()

	// 系统信息相关路由
	systemGroup := r.Group("/api/v1/system")
	{
		// 接收系统信息
		systemGroup.POST("/metrics", func(c *gin.Context) {
			ReceiveAndStoreSystemMetrics(c, repo)
		})

	}

	return r
}
