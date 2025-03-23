package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"

	m_init "cmd/server/model/init"
	u "cmd/server/model/user"
)

// 管理员获取当前公司的信息
func GetCompanyInfo(c *gin.Context) {
	username, _ := c.Get("username")
	// 判断当前用户是否有管理员权限
	if !isAdmin(username.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"message": "非公司管理员，权限不足"})
		return
	}
	// 查询管理员信息
	var admin u.User
	if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}

	// 查询管理员所在公司
	var company u.Company
	if err := m_init.DB.Where("id =?", admin.CompanyId).First(&company).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}

	// 查询公司成员
	var members []u.User
	if err := m_init.DB.Where("company_id =?", admin.CompanyId).Find(&members).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司成员失败"})
		return
	}

	response := struct {
		Company struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Admin       string `json:"admin"`
			MemberNum   int    `json:"member_num"`
			ServerNum   int    `json:"server_num"`
			Description string `json:"description"`
		}
		Members []struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		}
	}{
		Company: struct {
			ID          int    `json:"id"`
			Name        string `json:"name"`
			Admin       string `json:"admin"`
			MemberNum   int    `json:"member_num"`
			ServerNum   int    `json:"server_num"`
			Description string `json:"description"`
		}{
			ID:          company.ID,
			Name:        company.Name, // 假设字段名是 CompanyName 而不是 Name
			Admin:       username.(string),
			MemberNum:   company.MemberNum,
			ServerNum:   company.ServerNum, // 确保这个字段在 u.Company 中存在
			Description: company.Description,
		},
		Members: []struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		}{},
	}

	// 填充成员信息
	for _, member := range members {
		response.Members = append(response.Members, struct {
			Username string `json:"username"`
			Email    string `json:"email"`
		}{
			Username: member.Name,
			Email:    member.Email,
		})
	}

	// 返回响应
	c.JSON(http.StatusOK, gin.H{"message": "获取公司信息成功", "data": response})
}
