package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"wolfden.website/papermind/internal/service"
)

const CurrentUserIDKey = "currentUserID"

func RequireAuth(redisService *service.RedisService, jwtSecret []byte) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 从 Authorization header 获取 token
		header := c.GetHeader("Authorization")
		if header == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "缺少 Authorization 头"})
			return
		}

		if !strings.HasPrefix(header, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization 格式错误"})
			return
		}

		tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
		if tokenString == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authorization 格式错误"})
			return
		}

		// 2. 查 Redis：token 是否有效
		exists, err := redisService.Exists(c.Request.Context(), tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
			return
		}
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "token 已失效"})
			return
		}

		// 3. 解析 JWT 验证签名
		token, err := jwt.ParseWithClaims(tokenString, &service.TokenClaims{}, func(token *jwt.Token) (any, error) {
			if token.Method != jwt.SigningMethodHS256 {
				return nil, errors.New("签名算法错误")
			}
			return jwtSecret, nil
		})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "无效的 token"})
			return
		}

		claims, ok := token.Claims.(*service.TokenClaims)
		if !ok || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "无效的 token"})
			return
		}

		// 4. 存入 context
		c.Set(CurrentUserIDKey, claims.UserID)
		c.Next()
	}
}
