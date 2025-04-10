package company

import (
	m_init "backend/server/model/init"
	u "backend/server/model/user"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type JoinRequest struct {
	Username string `json:"username"` // 应该通过c.Get("username")获取
	Company  string `json:"company"`
}

func JoinCompany(c *gin.Context) {
	_, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户信息")
		c.JSON(401, gin.H{
			"message": "未找到用户信息",
		})
		return
	}

	var input JoinRequest
	// 解析JSON数据
	if err := c.BindJSON(&input); err != nil {
		log.Printf("请求数据格式错误")
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	//检查公司名是否存在
	var company u.Company
	if err := m_init.DB.Where("name = ?", input.Company).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("公司不存在")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "公司不存在"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
			return
		}
	}

	// 查询用户
	var user u.User
	err := m_init.DB.Where("name = ?", input.Username).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Printf("用户不存在")
			c.JSON(http.StatusUnauthorized, gin.H{"message": "用户不存在"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户名失败"})
			return
		}
	}

	// 加入公司
	user.CompanyId = company.ID
	if err := m_init.DB.Save(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库更新用户信息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "成功加入公司",
	})
}
