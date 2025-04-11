package admin

import(
	"github.com/gin-gonic/gin"
	"net/http"
	"errors"
	"gorm.io/gorm"

	m_init "backend/server/model/init"
	u "backend/server/model/user"
)

//检测是否为系统/公司管理员
func withPermission(username string) bool{
	var user u.User
	if err := m_init.DB.Where("name =?", username).First(&user).Error; err != nil {
		return false
	}
	if user.RoleId == 1 || user.RoleId == 2 {
		return true
	}
	return false
}

func AddSShkey(c *gin.Context) {
	var request struct {
		Hostname string `json:"hostname"`
		SSHKey string `json:"sshkey"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	username, exists := c.Get("username")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "未登录"})
		return
	}
	// 判断当前用户是否有管理员权限
	if !withPermission(username.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"message": "非系统/公司管理员，权限不足"})
		return
	}

	//检查是否存在该主机
	var sshkey u.SSHKey
	err := m_init.DB.Where("host_name =?", request.Hostname).First(&sshkey).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询失败"})
	}

	//如果存在该主机，则进行更新
	if err == nil {
		err = m_init.DB.Model(&sshkey).Update("sshkey", request.SSHKey).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库更新失败"})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"message": "SSH密钥更新成功",
		})
		return
	}

	newSSHkey := u.SSHKey{
		Hostname: request.Hostname, 
		SSHKey: request.SSHKey,
	}

	err = m_init.DB.Create(&newSSHkey).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "用户创建失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "SSH密钥添加成功",
	})
}