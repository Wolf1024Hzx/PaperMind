package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/dto"
	"wolfden.website/papermind/internal/middleware"
	"wolfden.website/papermind/internal/repository"
	"wolfden.website/papermind/internal/service"
)

type ChatHandler struct {
	chatService      *service.ChatService
	conversationRepo *repository.ConversationRepository
}

func NewChatHandler(chatService *service.ChatService, conversationRepo *repository.ConversationRepository) *ChatHandler {
	return &ChatHandler{
		chatService:      chatService,
		conversationRepo: conversationRepo,
	}
}

func (h *ChatHandler) RegisterRoutes(router *gin.RouterGroup, redisService *service.RedisService, jwtSecret []byte, jwtTTL time.Duration) {
	authMiddleware := middleware.RequireAuth(redisService, jwtSecret, jwtTTL)

	// 问答接口
	router.POST("/chat", authMiddleware, h.Ask)

	// 对话管理接口
	router.GET("/conversations", authMiddleware, h.GetConversations)
	router.GET("/conversations/:id/messages", authMiddleware, h.GetMessages)
	router.DELETE("/conversations/:id", authMiddleware, h.DeleteConversation)
}

func (h *ChatHandler) Ask(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)

	var req dto.ChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "请求格式错误"})
		return
	}

	if req.Question == "" {
		c.JSON(http.StatusBadRequest, gin.H{"message": "问题不能为空"})
		return
	}

	result, err := h.chatService.Ask(c.Request.Context(), uuid.MustParse(userID), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "问答失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, result)
}

// GetConversations 获取用户的对话列表
func (h *ChatHandler) GetConversations(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)

	convs, err := h.conversationRepo.FindByUserID(c.Request.Context(), uuid.MustParse(userID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"conversations": convs})
}

// GetMessages 获取对话的消息历史
func (h *ChatHandler) GetMessages(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	conversationIDStr := c.Param("id")

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的对话 ID"})
		return
	}

	// 验证对话归属
	_, err = h.conversationRepo.FindByIDAndUserID(c.Request.Context(), conversationID, uuid.MustParse(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "对话不存在或无权访问"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "查询失败"})
		}
		return
	}

	// 查询消息
	msgs, err := h.conversationRepo.FindMessages(c.Request.Context(), conversationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "查询消息失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"messages": msgs})
}

// DeleteConversation 删除对话
func (h *ChatHandler) DeleteConversation(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	conversationIDStr := c.Param("id")

	conversationID, err := uuid.Parse(conversationIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "无效的对话 ID"})
		return
	}

	// 删除（带用户校验）
	err = h.conversationRepo.DeleteByIDAndUserID(c.Request.Context(), conversationID, uuid.MustParse(userID))
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "对话不存在或无权访问"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "删除失败"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "删除成功"})
}
