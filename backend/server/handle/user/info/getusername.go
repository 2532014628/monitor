package info

import (
	m_init "backend/server/model/init"
	u "backend/server/model/user"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

func GetUserInfo(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		log.Printf("用户还未登录")
		c.JSON(401, gin.H{
			"message": "用户未登录",
		})
	}

	var user u.User
	err := m_init.DB.Where("name = ?", username.(string)).First(&user).Error
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "用户不存在"})
		return
	}
	// 过滤掉密码字段
	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户信息成功",
		"user":    user,
	})
}

func GetAllUserInfo(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		log.Printf("用户还未登录")
		c.JSON(401, gin.H{
			"message": "用户未登录",
		})
	}

	if username.(string) != "ROOT" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "用户没有权限"})
		return
	}

	var users []u.User
	err := m_init.DB.Find(&users).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "获取所有用户信息失败"})
		return
	}
	// 过滤掉密码字段
	for i := range users {
		users[i].Password = ""
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取用户信息成功",
		"users":   users,
	})
}
