package admin

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
	Realname string `json:"realname"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

// 公司管理员邀请成员加入公司
func JoinCompany(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户信息")
		c.JSON(401, gin.H{
			"message": "未找到用户信息",
		})
		return
	}
	Username := username.(string)

	var input JoinRequest
	// 解析JSON数据
	if err := c.BindJSON(&input); err != nil {
		log.Printf("请求数据格式错误")
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	// 查询公司管理员所在公司
	var admin u.User
	if err := m_init.DB.Where("name =?", Username).First(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}
	if admin.CompanyId == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "你没有就职于某个公司"})
		return
	}
	if admin.RoleId != 1 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "你没有邀请加入公司的权限"})
		return
	}

	//查找需邀请成员是否存在
	var user u.User
	if err := m_init.DB.Where("name =?", input.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "用户不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户失败"})
		return
	}
	if user.Realname == ""{
		c.JSON(http.StatusBadRequest, gin.H{"message": "邀请成员未实名" })
		return
	}
	if user.Realname != input.Realname{
		c.JSON(http.StatusBadRequest, gin.H{"message": "邀请成员真实姓名不匹配"})
		return
	}

	//查找要邀请人是否还在其他公司就职
	if user.CompanyId != 0 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "该用户已在其他公司就职"})
		return
	}

	//查看成员邮箱是否匹配
	if user.Email != input.Email {
		c.JSON(http.StatusBadRequest, gin.H{"message": "成员邮箱不匹配"})
		return
	}

	//查看公司管理员所在公司
	var company u.Company
	if err := m_init.DB.Where("id =?", admin.CompanyId).First(&company).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "公司不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}

	//发送邀请,并提示具体情况
	content := Username + "邀请" + input.Username + "加入" + company.Name
	invitation := u.Notice{
		Content:	content,
		Send:     	Username,
		Receive: 	input.Username,
		State: 		"unprocessed",
	}
	if err := m_init.DB.Create(&invitation).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库插入邀请失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "发出加入公司邀请",
	})
}

// type JoinRequest struct {
// 	Username string `json:"username"` // 应该通过c.Get("username")获取
// 	Company  string `json:"company"`
// }

// func JoinCompany(c *gin.Context) {
// 	_, exists := c.Get("username")
// 	if !exists {
// 		log.Printf("未找到用户信息")
// 		c.JSON(401, gin.H{
// 			"message": "未找到用户信息",
// 		})
// 		return
// 	}

// 	var input JoinRequest
// 	// 解析JSON数据
// 	if err := c.BindJSON(&input); err != nil {
// 		log.Printf("请求数据格式错误")
// 		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
// 		return
// 	}

// 	//检查公司名是否存在
// 	var company u.Company
// 	if err := m_init.DB.Where("name = ?", input.Company).First(&company).Error; err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			log.Printf("公司不存在")
// 			c.JSON(http.StatusUnauthorized, gin.H{"message": "公司不存在"})
// 			return
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
// 			return
// 		}
// 	}

// 	// 查询用户
// 	var user u.User
// 	err := m_init.DB.Where("name = ?", input.Username).First(&user).Error
// 	if err != nil {
// 		if errors.Is(err, gorm.ErrRecordNotFound) {
// 			log.Printf("用户不存在")
// 			c.JSON(http.StatusUnauthorized, gin.H{"message": "用户不存在"})
// 			return
// 		} else {
// 			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户名失败"})
// 			return
// 		}
// 	}

// 	// 加入公司
// 	user.CompanyId = company.ID
// 	if err := m_init.DB.Save(&user).Error; err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库更新用户信息失败"})
// 		return
// 	}

// 	c.JSON(http.StatusOK, gin.H{
// 		"message": "成功加入公司",
// 	})
// }
