package monitor

import (
	"backend/server/model"
	"fmt"
	"log"
	"net/http"

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

	result, err := model.ReadDB(queryType, from, to, hostname)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		log.Printf("error:%f", err)
		return
	}
	c.JSON(http.StatusOK, result)
}
