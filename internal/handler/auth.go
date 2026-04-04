package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/middleware"
	"wolfden.website/papermind/internal/service"
)

type AuthHandler struct {
	authService *service.AuthService
}

type registerRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type updateCurrentUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

func NewAuthHandler(authService *service.AuthService) *AuthHandler {
	return &AuthHandler{authService: authService}
}

func (h *AuthHandler) RegisterRoutes(router *gin.RouterGroup) {
	authGroup := router.Group("/auth")
	authGroup.POST("/register", h.Register)
	authGroup.POST("/login", h.Login)

	userGroup := router.Group("/users")
	userGroup.Use(middleware.RequireAuth(h.authService))
	userGroup.GET("/me", h.GetCurrentUser)
	userGroup.PUT("/me", h.UpdateCurrentUser)
	userGroup.DELETE("/me", h.DeleteCurrentUser)
}

func (h *AuthHandler) Register(c *gin.Context) {
	var request registerRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	result, err := h.authService.Register(c.Request.Context(), service.RegisterInput{
		Username: request.Username,
		Email:    request.Email,
		Password: request.Password,
	})
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var request loginRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	result, err := h.authService.Login(c.Request.Context(), service.LoginInput{
		Account:  request.Account,
		Password: request.Password,
	})
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
	var request updateCurrentUserRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求体格式错误"})
		return
	}

	userID := c.GetString(middleware.CurrentUserIDKey)
	result, err := h.authService.UpdateCurrentUser(c.Request.Context(), userID, service.UpdateCurrentUserInput{
		Username: request.Username,
		Email:    request.Email,
	})
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
