package repository

import (
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"wolfden.website/papermind/internal/model"
)

type PaperRepository struct {
	db *gorm.DB
}

func NewPaperRepository(db *gorm.DB) *PaperRepository {
	return &PaperRepository{db: db}
}

func (r *PaperRepository) Create(ctx context.Context, paper *model.Paper) error {
	return r.db.WithContext(ctx).Create(paper).Error
}

func (r *PaperRepository) FindByID(ctx context.Context, id uuid.UUID) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).First(&paper, "id = ?", id).Error
	if err != nil {
		return nil, err
	}

	return &paper, nil
}

func (r *PaperRepository) FindByUserID(ctx context.Context, userID uuid.UUID) ([]model.Paper, error) {
	var papers []model.Paper
	err := r.db.WithContext(ctx).Find(&papers, "user_id = ?", userID).Error
	if err != nil {
		return nil, err
	}

	return papers, nil
}

func (r *PaperRepository) FindByFileHash(ctx context.Context, hash string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).First(&paper, "file_hash = ?", hash).Error
	if err != nil {
		return nil, err
	}

	return &paper, nil
}

func (r *PaperRepository) UpdatePaperInfo(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", id).
		Updates(updates).
		Error
}

func (r *PaperRepository) DeletePaper(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.Paper{}, "id = ?", id).Error
}
