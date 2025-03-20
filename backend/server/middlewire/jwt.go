package middlewire

import (
	"net/http"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
)

var JwtKey = []byte("wujinhao123") // 用于加密的密钥，换成你想要的秘钥

type Claims struct {
	Username string `json:"username"`
	jwt.StandardClaims
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.GetHeader("Authorization")
		if tokenStr == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "请求头中缺少Token"})
			c.Abort()
			return
		}

		claims := &Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return JwtKey, nil // jwtKey 是你的签名密钥
		})
		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"message": "无效的Token"})
			c.Abort()
			return
		}

		// 将用户名保存到上下文，供后续处理使用
		c.Set("username", claims.Username)
		c.Next()
	}
}

// if err != nil {
// 	// 可以根据不同的错误类型给出不同的响应
// 	if ve, ok := err.(*jwt.ValidationError); ok {
// 		if ve.Errors&jwt.ValidationErrorMalformed != 0 {
// 			c.JSON(http.StatusBadRequest, gin.H{"message": "Token 格式错误"})
// 		} else if ve.Errors&(jwt.ValidationErrorExpired|jwt.ValidationErrorNotValidYet) != 0 {
// 			c.JSON(http.StatusUnauthorized, gin.H{"message": "Token 已过期或未激活"})
// 		} else {
// 			c.JSON(http.StatusUnauthorized, gin.H{"message": "无法处理 Token"})
// 		}
// 	} else {
// 		c.JSON(http.StatusUnauthorized, gin.H{"message": "Token 无效"})
// 	}
// 	c.Abort()
// 	return
// }
