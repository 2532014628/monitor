package info

import (
	"errors"
	"log"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	m_init "backend/server/model/init"
	u "backend/server/model/user"
)

//获取该用户作为接收者所接收到的所有信息

func GetReceiveList(c *gin.Context) {
	username , exists := c.Get("username")
	if !exists {
		log.Printf("用户还未登录")
		c.JSON(401, gin.H{
			"message": "用户未登录",
		})
	}
	Username := username.(string)

	// type notice struct{
	// 	Send      	string `json:"send"`
	// 	Content   	string `json:"content"`
	// 	State 		string   `json:"state"`
	// 	CreatedAt 	string `json:"created_at"`
	// }

	//获取该用户作为接收者所接收到的所有信息
	var receiveNotices []u.Notice
	err := m_init.DB.Where("receive = ?", Username).Find(&receiveNotices).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(200, gin.H{
				"message": "还未收到任何通知",
			})
			return
		} 
		log.Printf("Failed to get receiveNotices: %v\n", err)
		c.JSON(500, gin.H{
			"message": "数据库查询接收到的通知失败",
		})
		return
	}

	c.JSON(200, gin.H{
		"message": "获取接收到的通知成功",
		"receiveNotices": receiveNotices,
	})
}

//获取该用户作为发送者所发送的所有信息
func GetSendList(c *gin.Context) {
	username , exists := c.Get("username")
	if !exists {
		log.Printf("用户还未登录")
		c.JSON(401, gin.H{
			"message": "用户未登录",
		})
	}
	Username := username.(string)

	// type notice struct{
	// 	Receive   	string `json:"receive"`
	// 	Content   	string `json:"content"`
	// 	State 		string   `json:"state"`
	// 	CreatedAt 	string `json:"created_at"`
	// }

	//获取该用户作为发送者所发送的所有信息
	var sendNotices []u.Notice
	err := m_init.DB.Where("send = ?", Username).Find(&sendNotices).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(200, gin.H{
				"message": "还未发送任何通知",
			})
			return
		}
		log.Printf("Failed to get sendNotices: %v\n", err)
		c.JSON(500, gin.H{
			"message": "数据库查询发送的通知失败",
		})
	}

	c.JSON(200, gin.H{
		"message": "获取发送的通知成功",
		"sendNotices": sendNotices,
	})
}