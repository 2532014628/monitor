package info

import (
	m_init "backend/server/model/init"
	m_user "backend/server/model/user"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func ManageNotice(c *gin.Context) {
	Username, exists := c.Get("username")
	if !exists {
		log.Printf("用户还未登录")
		c.JSON(401, gin.H{
			"message": "用户未登录",
		})
	}
	username := Username.(string)

	var requestBody m_user.Notice
	err := c.BindJSON(&requestBody)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	mode := c.Query("mode")
	if mode == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少mode"})
		return
	}

	var notice m_user.Notice
	if strings.Contains(notice.Content, "注册") && strings.Contains(notice.Content, "申请") { // 处理注册公司的申请
		// 判断是否有权限
		var user m_user.User
		if err := m_init.DB.Where("name = ?", username).First(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "查询用户失败"})
			return
		}
		if user.RoleId != 2 { // 权限不足
			c.JSON(http.StatusForbidden, gin.H{"message": "审批人不具有权限"})
			return
		}
	}

	// 获取消息
	err = m_init.DB.Where("receive = ? and send = ? and created_at = ?",
		requestBody.Send, requestBody.Receive, requestBody.CreateAt).First(&notice).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "找不到该消息"})
			return
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "查询消息失败"})
			return
		}
	}

	// 判断消息是否已过期
	if notice.Processed == "expired" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "消息已过期"})
		return
	}

	// 修改消息状态
	err = m_init.DB.Model(&m_user.User{}).Where("id = ?", notice.ID).
		Update(map[string]interface{}{"processed": "processed"}).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "更新消息的处理状态失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "消息处理成功"})
}
