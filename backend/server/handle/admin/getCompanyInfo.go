package admin

import (
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/gorm"

	m_init "backend/server/model/init"
	u "backend/server/model/user"
)

func IsRoot(username string) bool {
	var user u.User
	if err := m_init.DB.Where("name =?", username).First(&user).Error; err != nil {
		return false
	}
	return user.RoleId == 2
}

type Request struct {
	Name string `json:"company-name"`
}

// 获取指定公司的信息
func GetCompanyInfo(c *gin.Context) {
	username, _ := c.Get("username")

	var company u.Company
	companyName := c.Query("company-name")
	if companyName == "" { // 未传递公司名，默认当前登录用户（管理员）所在的公司
		// 判断当前用户是否有管理员权限
		if !IsAdmin(username.(string)) && !IsRoot(username.(string)) {
			log.Println("非管理员，权限不足")
			c.JSON(http.StatusForbidden, gin.H{"message": "非管理员，权限不足"})
			return
		}
		// 查询管理员信息
		var admin u.User
		if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
			log.Printf("数据库查询管理员失败")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
			return
		}

		// 查询管理员所在公司
		if err := m_init.DB.Where("id =?", admin.CompanyId).First(&company).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				log.Println("当前管理员未加入任何公司")
				c.JSON(http.StatusUnauthorized, gin.H{"message": "当前管理员未加入任何公司"})
				return
			} else {
				log.Println("数据库查询公司失败")
				c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
				return
			}
		}

	} else { // 传递公司名，判断当前用户是否为root或指定公司的管理员
		if !IsRoot(username.(string)) {
			// 非系统管理员，判断是否为指定公司的管理员
			if !IsAdmin(username.(string)) {
				log.Println("非系统管理员或公司管理员，权限不足")
				c.JSON(http.StatusForbidden, gin.H{"message": "非管理员，权限不足"})
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
				if errors.Is(err, gorm.ErrRecordNotFound) {
					log.Println("当前管理员未加入任何公司")
				}
				log.Println("数据库查询公司失败")
				c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
				return
			}
			// 判断是否为指定公司的管理员
			if company.ID != admin.CompanyId {
				log.Println("非系统管理员或指定公司的管理员，权限不足")
				c.JSON(http.StatusForbidden, gin.H{"message": "非系统管理员或指定公司的管理员，权限不足"})
				return
			}
		}
		// 系统管理员，查询指定公司信息
		if err := m_init.DB.Where("name = ?", companyName).First(&company).Error; err != nil {
			log.Println("数据查询公司失败")
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据查询公司失败"})
			return
		}
	}

	// 查询公司成员
	var members []u.User
	if err := m_init.DB.Where("company_id =?", company.ID).Find(&members).Error; err != nil {
		log.Println("数据库查询公司成员失败")
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司成员失败"})
		return
	}

	response := struct {
		Company struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Admin       string `json:"admin"`
			MemberNum   int    `json:"member_num"`
			SystemNum   int    `json:"system_num"`
			Description string `json:"description"`
		}
		Members []struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			RoleId   int    `json:"role_id"`
		}
	}{
		Company: struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Admin       string `json:"admin"`
			MemberNum   int    `json:"member_num"`
			SystemNum   int    `json:"system_num"`
			Description string `json:"description"`
		}{
			ID:          company.ID,
			Name:        company.Name, // 假设字段名是 CompanyName 而不是 Name
			Admin:       username.(string),
			MemberNum:   company.MemberNum,
			SystemNum:   company.SystemNum, // 确保这个字段在 u.Company 中存在
			Description: company.Description,
		},
		Members: []struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			RoleId   int    `json:"role_id"`
		}{},
	}

	// 填充成员信息
	for _, member := range members {
		response.Members = append(response.Members, struct {
			Username string `json:"username"`
			Email    string `json:"email"`
			RoleId   int    `json:"role_id"`
		}{
			Username: member.Name,
			Email:    member.Email,
			RoleId:   member.RoleId,
		})
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{"message": "获取公司信息成功", "data": response})
}
