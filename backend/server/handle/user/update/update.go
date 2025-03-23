package update

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"strings"
	"time"

	m_init "cmd/server/model/init"
	u "cmd/server/model/user"
	"os"
	"strconv"

	"gopkg.in/gomail.v2"
	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

// 修改用户名、密码、邮箱
func UpdateUserInfo(c *gin.Context) {
	// 从上下文中获取用户名
	Username, exists := c.Get("username")
	if !exists {
		log.Printf("未找到用户名")
		c.JSON(401, gin.H{
			"code":    401,
			"success": false,
			"message": "未找到用户信息",
		})
		return
	}
	username := Username.(string)

	// 解析请求体	前端可只传递要修改的字段
	var request struct {
		NewName     string `json:"new_name"`
		NewPassword string `json:"new_password"`
		Email       string `json:"new_email"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误", "error": err.Error()})
		return
	}

	// 检查新用户名是否已存在
	if request.NewName != "" {
		var existingUser u.User
		if err := m_init.DB.Where("name = ?", request.NewName).First(&existingUser).Error; err == nil {
			c.JSON(http.StatusConflict, gin.H{"message": "更新用户名错误：新用户名已存在", "error": err.Error()})
			return
		} else if err == gorm.ErrRecordNotFound {
			// 用户名不存在，执行更新操作
			if err := m_init.DB.Model(&u.User{}).Where("name =?", username).Updates(map[string]interface{}{"name": request.NewName}).Error; err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "更新用户名失败", "error": err.Error()})
				return
			}
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询失败"})
			return
		}
	}

	// 检查新密码是否为空
	if request.NewPassword != "" {
		// 执行密码更新操作
		if err := m_init.DB.Model(&u.User{}).Where("name =?", username).Updates(map[string]interface{}{"password": request.NewPassword}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "更新密码失败", "error": err.Error()})
			return
		}
	}

	// 检查新邮箱是否为空
	if request.Email != "" {
		// 执行邮箱更新操作
		if err := m_init.DB.Model(&u.User{}).Where("name =?", username).Updates(map[string]interface{}{"email": request.Email}).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "更新邮箱失败", "error": err.Error()})
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "更新成功"})
}

// 密码找回：-----------------------------------
const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

func generateRandomToken(length int) string {
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	token := make([]byte, length)
	for i := range token {
		token[i] = charset[r.Intn(len(charset))]
	}

	return string(token)
}

// 处理重置密码请求
func RequestResetPassword(c *gin.Context) {
	// 实现请求重置密码的逻辑
	var request struct {
		Email string `json:"email"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	// 查找用户
	var user u.User
	err := m_init.DB.Where("email = ?", request.Email).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "用户未找到"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询失败"})
		}
		return
	}

	// 生成唯一的重置密码 token
	// token := fmt.Sprintf("%d", time.Now().UnixNano())
	token := generateRandomToken(6) // 生成6位长度的token
	fmt.Println("密码找回时生成的token为：", token)
	// 在数据库中保存 token
	err = m_init.DB.Model(&user).Update("token", token).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "保存 token 失败"})
		return
	}

	// 发送重置密码邮件
	sendResetPasswordEmail(request.Email, token)

	c.JSON(http.StatusOK, gin.H{
		"message": "重置密码请求成功",
	})
}

// 重置密码
func ResetPassword(c *gin.Context) {
	// 实现重置密码的逻辑
	var request struct {
		Token       string `json:"token"`
		NewPassword string `json:"new_password"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}
	fmt.Println("The new password is : ", request.NewPassword, ", and the token is : ", request.Token)

	var user u.User
	err := m_init.DB.Where("token = ?", request.Token).First(&user).Error
	// username, _ := c.Get("username")
	// err := m_init.DB.Where("token = ? and name = ?", request.Token, username.(string)).First(&user).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"message": "无效的重置密码 token"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询失败"})
		}
		return
	}

	err = m_init.DB.Model(&user).Update("password", request.NewPassword).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "密码重置失败"})
		return
	}

	err = m_init.DB.Model(&user).Update("token", nil).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "密码重置成功，但是 token 重置失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "重置密码成功",
	})
}

// 方式一 发送token
func sendResetPasswordEmail(email, token string) {
	myEmail := os.Getenv("EMAIL_NAME")
	myPassword := os.Getenv("EMAIL_PASSWORD")
	baseUrl := os.Getenv("BASE_URL")
	smtpServerHost := os.Getenv("SMTP_SERVER_HOST")
	smtpServerPortStr := os.Getenv("SMTP_SERVER_PORT")

	if myEmail == "" || myPassword == "" || baseUrl == "" || smtpServerHost == "" || smtpServerPortStr == "" {
		log.Fatalf("环境变量未正确设置")
	}

	smtpServerPort, err := strconv.Atoi(smtpServerPortStr)
	if err != nil {
		log.Fatalf("将端口号转换为整数时出错: %v", err)
	}

	log.Printf("Email: %s, Password: %s, SMTP Server: %s, Port: %d, BaseUrl: %s", myEmail, myPassword, smtpServerHost, smtpServerPort, baseUrl)

	m := gomail.NewMessage()
	m.SetHeader("From", myEmail)
	m.SetHeader("To", email)
	m.SetHeader("Subject", "Password Reset Request")
	m.SetBody("text/html", fmt.Sprintf(`
		<h1>密码找回</h1>
		<p>这是你的验证码：%s</p>
	`, token))

	d := gomail.NewDialer(smtpServerHost, smtpServerPort, myEmail, myPassword)
	if err := d.DialAndSend(m); err != nil {
		log.Printf("发送邮件失败: %v", err)
		if strings.Contains(err.Error(), "535") { // 例如，检查错误消息中是否包含 SMTP 身份验证失败的代码
			log.Printf("可能是 SMTP 身份验证错误")
		} else if strings.Contains(err.Error(), "connection refused") {
			log.Printf("SMTP 服务器连接被拒绝")
		}
	} else {
		log.Println("邮件发送成功")
	}
}

// 发送链接
// func sendResetPasswordEmail(email, token string) {
// 	myEmail := os.Getenv("EMAIL_NAME")
// 	myPassword := os.Getenv("EMAIL_PASSWORD")
// 	baseUrl := os.Getenv("BASE_URL")
// 	smtpServerHost := os.Getenv("SMTP_SERVER_HOST")
// 	smtpServerPortStr := os.Getenv("SMTP_SERVER_PORT")

// 	if myEmail == "" || myPassword == "" || baseUrl == "" || smtpServerHost == "" || smtpServerPortStr == "" {
// 		log.Fatalf("环境变量未正确设置")
// 	}

// 	smtpServerPort, err := strconv.Atoi(smtpServerPortStr)
// 	if err != nil {
// 		log.Fatalf("将端口号转换为整数时出错: %v", err)
// 	}

// 	log.Printf("Email: %s, Password: %s, SMTP Server: %s, Port: %d, BaseUrl: %s", myEmail, myPassword, smtpServerHost, smtpServerPort, baseUrl)

// 	m := gomail.NewMessage()
// 	m.SetHeader("From", myEmail)
// 	m.SetHeader("To", email)
// 	m.SetHeader("Subject", "Password Reset Request")
// m.SetBody("text/html", fmt.Sprintf(`
// 	<h1>Password Reset</h1>
// 	<p>Click the link to reset your password: <a href="%s/static/reset_password.html?token=%s" >Reset Password</a></p>
// `, baseUrl, token))

// 	d := gomail.NewDialer(smtpServerHost, smtpServerPort, myEmail, myPassword)
// 	if err := d.DialAndSend(m); err != nil {
// 		log.Printf("发送邮件失败: %v", err)
// 		if strings.Contains(err.Error(), "535") { // 例如，检查错误消息中是否包含 SMTP 身份验证失败的代码
// 			log.Printf("可能是 SMTP 身份验证错误")
// 		} else if strings.Contains(err.Error(), "connection refused") {
// 			log.Printf("SMTP 服务器连接被拒绝")
// 		}
// 	} else {
// 		log.Println("邮件发送成功")
// 	}
// }
