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

// RegisterRequest 定义了公司注册请求结构体
type RegisterRequest struct {
	Company            string `json:"company"`
	Social_Credit_Code string `json:"social_credit_code"`
	Legal_Name         string `json:"legal_name"`
	Admin_Name         string `json:"admin_name"`
	Admin_Email        string `json:"admin_email"`
}

// 邀请团队
func Register(c *gin.Context) {

	Username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户信息")
		c.JSON(401, gin.H{
			"message": "未找到用户信息",
		})
		return
	}
	username := Username.(string)

	//定义用于接收JSON数据的请求体
	var input RegisterRequest

	// 解析JSON数据
	if err := c.BindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	//检查公司名是否已经存在
	var company u.Company
	if err := m_init.DB.Where("name = ?", input.Company).First(&company).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "公司名已存在"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}

	//简单检查社会信用代码格式是否正确
	//if len(input.Social_Credit_Code) != 18 {
	//	c.JSON(http.StatusBadRequest, gin.H{"message": "社会信用代码应该为18位"})
	//	return
	//}
	//检测公司统一社会信用代码是否已经存在
	if err := m_init.DB.Where("social_credit_code = ?", input.Social_Credit_Code).First(&company).Error; err == nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "公司统一社会信用代码已存在"})
		return
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询公司失败"})
		return
	}

	//查看公司管理员的用户信息
	var admin u.User
	if err := m_init.DB.Where("name = ?", username).First(&admin).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusBadRequest, gin.H{"message": "用户信息不存在"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户失败"})
		return
	}
	if admin.Realname == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "管理员需要实名"})
		return
	}
	if admin.Realname != input.Admin_Name {
		c.JSON(http.StatusBadRequest, gin.H{"message": "管理员实名信息不匹配"})
		return
	}
	if admin.Email != input.Admin_Email {
		c.JSON(http.StatusBadRequest, gin.H{"message": "管理员邮箱不匹配"})
		return
	}

	//团队申请
	content := username + "申请注册公司:" + input.Company + "，法人：" + input.Legal_Name +
		",管理员:" + input.Admin_Name + ",社会信用代码:" + input.Social_Credit_Code +
		",管理员邮箱:" + input.Admin_Email

	notice := u.Notice{
		Content: content,
		Send:    username,
		Receive: "root",
		State:   "unprocessed",
	}
	if err := m_init.DB.Create(&notice).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库插入申请失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "发出团队申请",
	})

	//创建公司
	//newCompany := u.Company{
	//	Name:             input.Company,
	//	SocialCreditCode: input.Social_Credit_Code,
	//	AdminID:          admin.ID,
	//	Description:      "暂无",
	//	MemberNum:        1,
	//}
	//if err := m_init.DB.Create(&newCompany).Error; err != nil {
	//	c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库创建公司失败"})
	//	return
	//}

	//同步更新公司管理员的company_id和role_id
	//if err := m_init.DB.Model(&admin).Updates(u.User{CompanyId: newCompany.ID, RoleId: 1}).Error; err != nil {
	//	c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库更新用户失败"})
	//	return
	//}

	//c.JSON(http.StatusOK, gin.H{
	//	"message": "公司注册成功",
	//})
}
