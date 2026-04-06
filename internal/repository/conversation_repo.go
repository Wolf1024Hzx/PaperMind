package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
)

type ConversationRepository struct {
	db *gorm.DB
}

func NewConversationRepository(db *gorm.DB) *ConversationRepository {
	return &ConversationRepository{db: db}
}

// Create 创建新对话
func (r *ConversationRepository) Create(ctx context.Context, conv *model.Conversation) error {
	return r.db.WithContext(ctx).Create(conv).Error
}

// FindByUserID 查询用户的所有对话，按更新时间倒序
func (r *ConversationRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Conversation, error) {
	var convs []model.Conversation
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("updated_at DESC").
		Find(&convs).Error
	return convs, err
}

// CreateMessage 创建消息
func (r *ConversationRepository) CreateMessage(ctx context.Context, msg *model.Message) error {
	return r.db.WithContext(ctx).Create(msg).Error
}

// FindMessages 查询对话的所有消息，按时间正序
func (r *ConversationRepository) FindMessages(ctx context.Context, conversationID uuid.UUID) ([]model.Message, error) {
	var msgs []model.Message
	err := r.db.WithContext(ctx).
		Where("conversation_id = ?", conversationID).
		Order("created_at ASC").
		Find(&msgs).Error
	return msgs, err
}

// Delete 删除对话（messages 会自动级联删除）
func (r *ConversationRepository) Delete(ctx context.Context, conversationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("id = ?", conversationID).
		Delete(&model.Conversation{}).Error
}

// FindByIDAndUserID 根据 ID 和用户 ID 查询对话（用于权限校验）
func (r *ConversationRepository) FindByIDAndUserID(ctx context.Context, conversationID, userID uuid.UUID) (*model.Conversation, error) {
	var conv model.Conversation
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		First(&conv).Error
	if err != nil {
		return nil, err
	}
	return &conv, nil
}

// DeleteByIDAndUserID 删除对话（带用户校验）
func (r *ConversationRepository) DeleteByIDAndUserID(ctx context.Context, conversationID, userID uuid.UUID) error {
	result := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", conversationID, userID).
		Delete(&model.Conversation{})

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// UpdateUpdatedAt 更新对话的 updated_at 时间
func (r *ConversationRepository) UpdateUpdatedAt(ctx context.Context, conversationID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Model(&model.Conversation{}).
		Where("id = ?", conversationID).
		Update("updated_at", time.Now()).Error
}
