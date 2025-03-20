package monitor

import (
	"cmd/server/model"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
)

// GetAgentInfo 用于查询特定主机信息
func GetAgentInfo(c *gin.Context) {
	db, _, err := model.InitDB()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "数据库初始化失败"})
		return
	}
	defer db.Close()

	hostname := c.Param("hostname")
	if len(hostname) == 0 {
		log.Printf("名字出错！")
		c.JSON(http.StatusBadRequest, gin.H{"error": "主机名不能为空"})
		return
	}
	fmt.Printf("hostname:%v", hostname)
	fmt.Println()
	queryType := c.DefaultQuery("type", "all")
	from := c.Query("from")
	to := c.Query("to")

	if from == "" {
		from = "1970-01-01T00:00:00Z"
	}
	if to == "" {
		to = "9999-12-31T23:59:59Z"
	}

	result, err := model.ReadDB(db, queryType, from, to, hostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Printf("error:%f", err)
		return
	}
	c.JSON(http.StatusOK, result)
}
