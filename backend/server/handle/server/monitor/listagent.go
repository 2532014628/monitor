package monitor

import (
	"cmd/server/model"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// ListAgent 用于查询所有主机信息
func ListAgent(c *gin.Context) {
	db, _, err := model.InitDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库初始化失败"})
		return
	}
	defer db.Close()

	// 从上下文中获取用户名
	Username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户名")
		c.JSON(401, gin.H{
			"code":    401,
			"success": false,
			"message": "未找到用户信息",
		})
		return
	}
	username := Username.(string)

	// 解析时间查询参数
	from := c.Query("from")
	to := c.Query("to")

	if from == "" {
		from = "1970-01-01T00:00:00Z"
	}
	if to == "" {
		to = "9999-12-31T23:59:59Z"
	}

	fromTime, err := time.Parse(time.RFC3339, from)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 from 时间格式"})
		return
	}
	toTime, err := time.Parse(time.RFC3339, to)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "无效的 to 时间格式"})
		return
	}

	// 查询数据库，过滤出当前用户的主机
	query := `
		SELECT id, host_name, os, platform, kernel_arch, created_at
		FROM host_info
		WHERE user_name = $1 AND created_at BETWEEN $2 AND $3
	`

	rows, err := db.Query(query, username, fromTime, toTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to query host_info", "details": err.Error()})
		return
	}
	defer rows.Close()

	var hosts []model.HostInfo
	for rows.Next() {
		var host model.HostInfo
		if err := rows.Scan(&host.ID, &host.Hostname, &host.OS, &host.Platform, &host.KernelArch, &host.CreatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to scan host_info", "details": err.Error()})
			return
		}
		hosts = append(hosts, host)
	}
	if err := rows.Err(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error occurred during iteration", "details": err.Error()})
		return
	}

	c.JSON(http.StatusOK, hosts)
}
