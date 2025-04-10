package company

import (
	m_init "backend/server/model/init"
	"net/http"

	admin "backend/server/handle/admin"
	u "backend/server/model/user"

	"github.com/gin-gonic/gin"
)

type Company struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AdminID     int    `json:"admin_id"`
	AdminName   string `json:"adminName"`
	AdminEmail  string `json:"adminEmail"`
	MemberNum   int    `json:"membernum"`
	SystemNum   int    `json:"systemnum"`
}

func GetCompanyList(c *gin.Context) {
	username, _ := c.Get("username")

	if !admin.IsRoot(username.(string)) { // 系统管理员
		c.JSON(http.StatusForbidden, gin.H{"message": "非系统管理员，权限不足"})
		return
	}
	var companies []u.Company
	if err := m_init.DB.Find(&companies).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}

	var responseCompanies []Company

	for _, company := range companies {
		var admin u.User
		if err := m_init.DB.Where("id = ?", company.AdminID).First(&admin).Error; err != nil {
			continue // 找不到管理员，这里先简单跳过这条记录
		}

		responseCompany := Company{
			ID:          company.ID,
			Name:        company.Name,
			Description: company.Description,
			AdminID:     company.AdminID,
			AdminName:   admin.Name,
			AdminEmail:  admin.Email,
			MemberNum:   company.MemberNum,
			SystemNum:   company.SystemNum,
		}
		responseCompanies = append(responseCompanies, responseCompany)
	}

	c.JSON(http.StatusOK, gin.H{"message": "查询成功", "data": responseCompanies})
}
