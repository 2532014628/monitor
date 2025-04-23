package admin

import(
	"github.com/gin-gonic/gin"
	"log"
	"net/http"
	"gorm.io/gorm"
	"errors"


	m_init "backend/server/model/init"
	u "backend/server/model/user"
)

type RepalceRequest struct {
	Realname string `json:"realname"`
	Username string `json:"username"`
	Email    string `json:"email"`
}

//更换公司管理员
func ReplaceAdmin(c *gin.Context) {
	username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户信息")
		c.JSON(401, gin.H{
			"message": "未找到用户信息",
		})
		return
	}
	Username := username.(string)

	var input RepalceRequest
	if err := c.BindJSON(&input); err != nil {
		log.Printf("请求数据格式错误")
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	//查看当前管理员信息
	var oldAdmin u.User
	if err := m_init.DB.Where("name =?", Username).First(&oldAdmin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "管理员不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}
	if oldAdmin.RoleId != 1 {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "没有更换管理员权限"})
		return
	}

	//查找新管理员信息
	var user u.User
	if err := m_init.DB.Where("name =?", input.Username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "新管理员不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}
	if user.Realname == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "新管理员未实名"})
		return
	}
	if user.Email != input.Email {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "新管理员邮箱不匹配"})
		return
	}
	if user.CompanyId != oldAdmin.CompanyId {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "新管理员和你不在一个公司"})
		return
	}
	
	//发出更换管理员申请
	content := Username + "申请更换公司管理,新管理员姓名" + input.Realname + ",新管理员用户名" + 
			input.Username  + ",新管理员邮箱" + input.Email
	notice := u.Notice{
		Content:    content,
		Send:      	Username,
		Receive: 	"root",
		Processed: 	false,
	}
	if err := m_init.DB.Create(&notice).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库插入申请失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "发出更换管理员申请",
	})
}