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

func (r *PaperRepository) FindByUserIDAndFileHash(ctx context.Context, userID uuid.UUID, hash string) (*model.Paper, error) {
	var paper model.Paper
	err := r.db.WithContext(ctx).First(&paper, "user_id = ? AND file_hash = ?", userID, hash).Error
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

// 更新论文的处理状态
func (r *PaperRepository) UpdateStatus(ctx context.Context, paperID uuid.UUID, status string) error {
	return r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", paperID).
		Update("status", status).Error
}

// 仅当字段为空时更新元数据
func (r *PaperRepository) UpdateMetadataIfEmpty(ctx context.Context, paperID uuid.UUID, updates map[string]interface{}) error {
	// 先查出当前记录
	var paper model.Paper
	if err := r.db.WithContext(ctx).First(&paper, "id = ?", paperID).Error; err != nil {
		return err
	}

	// 只更新当前为空的字段
	finalUpdates := make(map[string]interface{})
	if v, ok := updates["title"]; ok && paper.Title == "" {
		if title, ok := v.(string); ok && title != "" {
			finalUpdates["title"] = title
		}
	}
	if v, ok := updates["authors"]; ok && paper.Authors == "" {
		if authors, ok := v.(string); ok && authors != "" {
			finalUpdates["authors"] = authors
		}
	}

	if len(finalUpdates) == 0 {
		return nil
	}

	return r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", paperID).
		Updates(finalUpdates).Error
}

// UpdateChunkCount 更新论文的 chunk 数量
func (r *PaperRepository) UpdateChunkCount(ctx context.Context, paperID uuid.UUID, count int) error {
	return r.db.WithContext(ctx).
		Model(&model.Paper{}).
		Where("id = ?", paperID).
		Update("chunk_count", count).Error
}
