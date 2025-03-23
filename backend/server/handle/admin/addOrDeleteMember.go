package admin

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	_ "github.com/lib/pq"
	"gorm.io/gorm"

	m_init "cmd/server/model/init"
	u "cmd/server/model/user"
)

// 管理员权限检查
func isAdmin(username string) bool {
	var user u.User
	if err := m_init.DB.Where("name =?", username).First(&user).Error; err != nil {
		return false
	}
	return user.RoleId == 1
}

// 公司管理员添加公司成员
func AddMember(c *gin.Context) {
	var request struct {
		Name  string `json:"username"`
		Email string `json:"email"`
	}

	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}

	username, _ := c.Get("username")
	// 判断当前用户是否有管理员权限
	if !isAdmin(username.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"message": "非公司管理员，权限不足"})
		return
	}

	// 查询管理员所在公司
	var admin u.User
	if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}

	// 检查用户名是否存在
	var isExist bool = false
	var user u.User
	err := m_init.DB.Where("name = ?", request.Name).First(&user).Error
	if err == nil {
		// 判断该用户是否已经属于某个公司
		if user.CompanyId == admin.CompanyId {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "该用户名已属于当前公司"})
			return
		} else if user.CompanyId != 0 {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "该用户名已属于其他某个公司"})
			return
		}
		isExist = true
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果 err 不为 nil 且不是因为记录未找到导致的，则是其他数据库错误
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询用户名失败"})
		return
	}

	if isExist {
		// 更新用户信息
		err := m_init.DB.Model(&user).Updates(u.User{
			CompanyId: admin.CompanyId,
		}).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "用户已存在，且用户加入公司失败"})
			return
		}
	} else {
		// 创建用户
		// 判断邮箱是否存在
		err = m_init.DB.Where("email = ?", request.Email).First(&user).Error
		if err == nil {
			c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "邮箱已存在"})
			return
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			// 如果 err 不为 nil 且不是因为记录未找到导致的，则是其他数据库错误
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询邮箱失败"})
			return
		}
		newUser := u.User{
			Name:       request.Name,
			Email:      request.Email,
			Password:   "123456",
			RoleId:     0,
			CompanyId:  admin.CompanyId,
			IsVerified: true,
		}
		err = m_init.DB.Create(&newUser).Error
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "用户创建失败"})
			return
		}
	}

	// 返回结果，包括加密后的密码
	c.JSON(http.StatusOK, gin.H{
		"message": "管理员" + username.(string) + "成功添加成员" + request.Name,
	})
}

// 公司管理员批量删除公司成员
func DeleteMember(c *gin.Context) {
	var request struct {
		Names []string `json:"usernames"`
	}
	if err := c.BindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求数据格式错误"})
		return
	}
	username, _ := c.Get("username")
	// 判断当前用户是否有管理员权限
	if !isAdmin(username.(string)) {
		c.JSON(http.StatusForbidden, gin.H{"message": "非公司管理员，权限不足"})
		return
	}
	// 查询管理员所在公司
	var admin u.User
	if err := m_init.DB.Where("name =?", username.(string)).First(&admin).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询管理员失败"})
		return
	}

	companyId := admin.CompanyId

	// 开始事务
	tx := m_init.DB.Begin()
	if tx.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "事务开始失败"})
		return
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 标记是否发生错误
	hasError := false

	// 遍历删除成员
	for _, name := range request.Names {
		var user u.User
		// 查询用户是否存在
		err := tx.Where("name =?", name).First(&user).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				continue // 成员不存在，跳过
			}
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"message": "数据库查询成员" + name + "失败；已取消当前所有成员删除操作"})
			return
		}
		// 检查成员与管理员是否同一个公司
		if user.CompanyId != companyId {
			hasError = true
			continue // 不是同一个公司，记录错误，最后统一处理；跳过
		}

		// 删除成员：更新用户表中的 company_id 为 0
		if err := m_init.DB.Model(&u.User{}).Where("name = ?", user.Name).Update("company_id", 0).Error; err != nil {
			hasError = true
			fmt.Printf("Error updating user %s: %v\n", user.Name, err)
			continue
		}
	}
	if hasError {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "部分成员删除失败；已取消当前所有成员删除操作"})
		return
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"message": "事务提交失败，取消当前所有成员删除操作"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "管理员" + username.(string) + "成功删除指定成员",
	})
}
