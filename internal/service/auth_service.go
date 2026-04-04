package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
	"wolfden.website/papermind/internal/repository"
)

var (
	ErrInvalidCredentials = errors.New("用户名或密码错误")
	ErrInvalidInput       = errors.New("请求参数不合法")
	ErrUserAlreadyExists  = errors.New("用户名或邮箱已存在")
)

type AuthService struct {
	userRepo     *repository.UserRepository
	redisService *RedisService
	jwtSecret    []byte
	jwtTTL       time.Duration
}

type RegisterInput struct {
	Username string
	Email    string
	Password string
}

type LoginInput struct {
	Account  string
	Password string
}

type UpdateCurrentUserInput struct {
	Username string
	Email    string
}

type AuthResult struct {
	Token string      `json:"token"`
	User  UserProfile `json:"user"`
}

type UserProfile struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type TokenClaims struct {
	UserID   string `json:"sub"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

func NewAuthService(userRepo *repository.UserRepository, redisService *RedisService, jwtSecret string, jwtTTL time.Duration) *AuthService {
	return &AuthService{
		userRepo:     userRepo,
		redisService: redisService,
		jwtSecret:    []byte(jwtSecret),
		jwtTTL:       jwtTTL,
	}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	if input.Username == "" || input.Email == "" || input.Password == "" {
		return nil, ErrInvalidInput
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &model.User{
		Username: input.Username,
		Email:    input.Email,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrUserAlreadyExists
		}

		return nil, err
	}

	token, err := s.createToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  toUserProfile(user),
	}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	input.Account = strings.TrimSpace(input.Account)
	if input.Account == "" || input.Password == "" {
		return nil, ErrInvalidInput
	}

	user, err := s.userRepo.FindByAccount(ctx, input.Account)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrInvalidCredentials
		}

		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	token, err := s.createToken(user)
	if err != nil {
		return nil, err
	}

	return &AuthResult{
		Token: token,
		User:  toUserProfile(user),
	}, nil
}

func (s *AuthService) GetCurrentUser(ctx context.Context, userID string) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	profile := toUserProfile(user)
	return &profile, nil
}

func (s *AuthService) UpdateCurrentUser(ctx context.Context, userID string, input UpdateCurrentUserInput) (*UserProfile, error) {
	id, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	updates := map[string]any{}
	input.Username = strings.TrimSpace(input.Username)
	input.Email = strings.TrimSpace(input.Email)
	if input.Username != "" {
		updates["username"] = input.Username
	}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if len(updates) == 0 {
		return nil, ErrInvalidInput
	}

	if err := s.userRepo.UpdateProfile(ctx, id, updates); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrUserAlreadyExists
		}

		return nil, err
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	profile := toUserProfile(user)
	return &profile, nil
}

func (s *AuthService) DeleteCurrentUser(ctx context.Context, userID string) error {
	id, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}

	return s.userRepo.Delete(ctx, id)
}

func (s *AuthService) createToken(user *model.User) (string, error) {
	now := time.Now()
	claims := TokenClaims{
		UserID:   user.ID.String(),
		Username: user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(s.jwtTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}

	// 存入 Redis：key = token, value = userID, TTL = token 有效期
	if err := s.redisService.Set(context.Background(), tokenString, user.ID.String(), s.jwtTTL); err != nil {
		return "", err
	}

	return tokenString, nil
}

func (s *AuthService) Logout(ctx context.Context, tokenString string) error {
	return s.redisService.Del(ctx, tokenString)
}

func toUserProfile(user *model.User) UserProfile {
	return UserProfile{
		ID:        user.ID.String(),
		Username:  user.Username,
		Email:     user.Email,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}
}
