package company

import (
	m_init "cmd/server/model/init"
	"net/http"

	"github.com/gin-gonic/gin"
	u "cmd/server/model/user"
	admin "cmd/server/handle/admin"
)

func GetCompanyList(c *gin.Context) {
	username, _ := c.Get("username")
	
	if !admin.IsRoot(username.(string)) { // 系统管理员
		c.JSON(http.StatusForbidden, gin.H{"message": "非系统管理员，权限不足"})
		return
	}
	var companies []u.Company
	if err := m_init.DB.Find(&companies).Error; err!= nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "查询成功", "data": companies})
}