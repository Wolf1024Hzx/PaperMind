package dto

import "time"

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

// UpdateCurrentUserRequest 更新当前用户请求
type UpdateCurrentUserRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
}

// AuthResult 登录返回结果
type AuthResult struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

// UserProfile 用户信息
type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
