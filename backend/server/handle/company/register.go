package company

import (
	m_init "cmd/server/model/init"
	u "cmd/server/model/user"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

)

// RegisterRequest 定义了公司注册请求结构体
type RegisterRequest struct {
    Username string `json:"username"`
	Company  string `json:"company"`
	RealName string `json:"realname"`
	Identity string `json:"identity"`
}

//检查法人是否年满18周岁和年月日部分是否合法
func checkLegalAge(identity string) bool {

	yearStr := identity[6:10]
	year , _ := strconv.Atoi(yearStr)
	monthStr := identity[10:12]
	month , _ := strconv.Atoi(monthStr)
	dayStr := identity[12:14]
	day , _ := strconv.Atoi(dayStr)

	if year < 1900 || year > 2007 || month < 1 || month > 12 || day < 1 || day > 31 {
		return false
	}
	if month == 2 {
		if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
			if day > 29 {
				return false
			}
		} else {
			if day > 28 {
				return false
			}
		}
	} else if month == 4 || month == 6 || month == 9 || month == 11 {
		if day >30 {
			return false
		}
	}else {
		if day > 31 {
			return false
		}
	}
    return true
}

//对身份证格式进行检验
func checkIdentity(identity string) bool {
    // 身份证长度为18位
    if len(identity) != 18 {
        return false
    }

    // 检查每一位是否为数字或最后一位为大写X
    for i, char := range identity {
        if i < 17 && (char < '0' || char > '9') {
            return false
        }
        if i == 17 && char != 'X' && (char < '0' || char > '9') {
            return false
        }
    }

    // 计算校验码
    weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
    sum := 0
    for i := 0; i < 17; i++ {
        digit := int(identity[i] - '0')
        sum += digit * weights[i]
    }

    remainder := sum % 11
    validChecksums := "10X98765432"
    expectedChecksum := validChecksums[remainder]

    return identity[17] == expectedChecksum
}

func Register(c *gin.Context) {

	// 从上下文中获取用户名
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

	//检查当前用户用户名是否和公司法人匹配
	if username != input.Username {
		c.JSON(http.StatusUnauthorized , gin.H{"message": "没有权限注册"})
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

	//检查法人的年龄是否年满18周岁
	if !checkLegalAge(input.Identity) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "法人年龄未满18周岁"})
		return
	}

	//检查法人的身份证格式
	if !checkIdentity(input.Identity) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "身份证格式错误"})
		return
	}

	//检查法人身份证和真实名字是否匹配
	//暂时还没有，跳过
	
	//查找该法人的id
	var user u.User
	if err := m_init.DB.Where("name = ?", username).First(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户失败"})
		return
	}

	//创建公司
	newCompany := u.Company{
		Name: input.Company,
		AdminID: user.ID,
		Description: "暂无",
		MemberNum: 1,
	}
	if err := m_init.DB.Create(&newCompany).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库创建公司失败"})
		return
	}

	//同步更新该法人的company_id和role_id
	if err := m_init.DB.Model(&user).Updates(u.User{CompanyId: newCompany.ID, RoleId: 1}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库更新用户失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "公司注册成功",
	})
}