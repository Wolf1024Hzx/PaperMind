package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/dto"
	"wolfden.website/papermind/internal/middleware"
	"wolfden.website/papermind/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup, redisService *service.RedisService, jwtSecret []byte, jwtTTL time.Duration) {
	authGroup := router.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)
	authGroup.POST("/logout", h.Logout)

	userGroup := router.Group("/users")
	userGroup.Use(middleware.RequireAuth(redisService, jwtSecret, jwtTTL))
	userGroup.GET("/me", h.GetCurrentUser)
	userGroup.PUT("/me", h.UpdateCurrentUser)
	userGroup.DELETE("/me", h.DeleteCurrentUser)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request dto.RegisterRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	profile, err := h.authService.Register(c.Request.Context(), request)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, profile)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request dto.LoginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), request)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) GetCurrentUser(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	result, err := h.authService.GetCurrentUser(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) UpdateCurrentUser(c *gin.Context) {
	var request dto.UpdateCurrentUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	userID := c.GetString(middleware.CurrentUserIDKey)
	result, err := h.authService.UpdateCurrentUser(c.Request.Context(), userID, request)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *AuthHandler) DeleteCurrentUser(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	if err := h.authService.DeleteCurrentUser(c.Request.Context(), userID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *AuthHandler) Logout(c *gin.Context) {
	header := c.GetHeader("Authorization")
	tokenString := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))

	if tokenString == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少 token"})
		return
	}

	if err := h.authService.Logout(c.Request.Context(), tokenString); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "登出成功"})
}

func (h *AuthHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrUserAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "用户不存在"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
	}
}
