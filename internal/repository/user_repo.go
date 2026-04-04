package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
)

type UserRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{db: db}
}

func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) FindByAccount(ctx context.Context, account string) (*model.User, error) {
	var user model.User
	err := r.db.WithContext(ctx).
		Where("username = ? OR email = ?", account, account).
		First(&user).
		Error
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&model.User{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.User{}, "id = ?", id).Error
}
