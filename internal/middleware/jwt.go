package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"wolfden.website/papermind/internal/service"
)

const CurrentUserIDKey = "currentUserID"

func RequireAuth(authService *service.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
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

		claims, err := authService.ParseToken(tokenString)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "无效的 token"})
			return
		}

		c.Set(CurrentUserIDKey, claims.UserID)
		c.Next()
	}
}
