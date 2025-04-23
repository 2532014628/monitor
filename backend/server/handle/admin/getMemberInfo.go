package admin

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	m_init "backend/server/model/init"
	u "backend/server/model/user"
)

func GetMemberInfo(c *gin.Context) {
	username, _ := c.Get("username")

	var company u.Company
	companyName := c.Query("companyName")
	if companyName == "" { // 未传递公司名，默认当前登录用户（管理员）所在的公司
		// 判断当前用户是否有管理员权限
		if !IsAdmin(username.(string)) && !IsRoot(username.(string)) {
			log.Println("非公司管理员，权限不足")
			c.JSON(http.StatusForbidden, gin.H{"message": "非公司管理员，权限不足"})
			return
		}
		// 查询管理员信息
		var admin u.User
		if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
			log.Println("数据库查询管理员失败")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
			return
		}
		// 查询管理员所在公司
		if err := m_init.DB.Where("id =?", admin.CompanyId).First(&company).Error; err != nil {
			log.Println("数据库查询公司失败")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
			return
		}

	} else { // 传递公司名，判断当前用户是否为root或指定公司的管理员
		if !IsRoot(username.(string)) {
			// 非系统管理员，判断是否为指定公司的管理员
			if !IsAdmin(username.(string)) {
				c.JSON(http.StatusForbidden, gin.H{"message": "非系统管理员或公司管理员，权限不足"})
				return
			}
			// 查询管理员信息
			var admin u.User
			if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
				return
			}

			// 查询管理员所在公司
			if err := m_init.DB.Where("id =?", admin.CompanyId).First(&company).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
				return
			}
			// 判断是否为指定公司的管理员
			if company.ID != admin.CompanyId {
				c.JSON(http.StatusForbidden, gin.H{"message": "非指定公司的管理员，权限不足"})
				return
			}
		}
		// 系统管理员，查询指定公司信息
		if err := m_init.DB.Where("name =?", companyName).First(&company).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据查询公司失败"})
			return
		}
	}

	// 查询公司成员
	var members []u.User
	if err := m_init.DB.Where("company_id =?", company.ID).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司成员失败"})
		return
	}

	// 返回公司成员信息
	type Member struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		RoleId   int    `json:"role_id"`
	}

	var memberList []Member
	for _, member := range members {
		memberList = append(memberList, Member{Username: member.Name, Email: member.Email, RoleId: member.RoleId})
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "获取公司成员信息成功",
		"members": memberList,
	})
}
