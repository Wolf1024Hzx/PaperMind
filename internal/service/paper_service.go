package service

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
	"wolfden.website/papermind/internal/repository"
)

type UploadPaperInput struct {
	Filename string
	FileData []byte
	FileSize int64
	FileHash string
	Title    string
	Authors  string
	Year     *int
	Venue    string
}

type PaperDTO struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	FileSize   int64  `json:"fileSize"`
	Title      string `json:"title"`
	Authors    string `json:"authors"`
	Year       *int   `json:"year"`
	Venue      string `json:"venue"`
	Status     string `json:"status"`
	ChunkCount int    `json:"chunkCount"`
	CreatedAt  string `json:"createdAt"`
	UpdatedAt  string `json:"updatedAt"`
}

type PaperService struct {
	paperRepo *repository.PaperRepository
	uploadDir string
}

func NewPaperService(paperRepo *repository.PaperRepository, uploadDir string) *PaperService {
	return &PaperService{
		paperRepo: paperRepo,
		uploadDir: uploadDir,
	}
}

func (s *PaperService) UploadPaper(ctx context.Context, userID string, input UploadPaperInput) (*PaperDTO, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	// 检查文件是否已存在（去重）
	_, err = s.paperRepo.FindByFileHash(ctx, input.FileHash)
	if err == nil {
		return nil, ErrFileAlreadyExists
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	paper := &model.Paper{
		UserID:   userUUID,
		Filename: input.Filename,
		FileSize: input.FileSize,
		FileHash: input.FileHash,
		Title:    input.Title,
		Authors:  input.Authors,
		Year:     input.Year,
		Venue:    input.Venue,
		Status:   "pending",
	}

	// 先创建数据库记录（获取 UUID）
	if err := s.paperRepo.Create(ctx, paper); err != nil {
		return nil, err
	}

	// 再保存文件
	filePath := fmt.Sprintf("%s/%s.pdf", s.uploadDir, paper.ID.String())
	if err := os.WriteFile(filePath, input.FileData, 0644); err != nil {
		// 文件保存失败，删除数据库记录（补偿操作）
		s.paperRepo.DeletePaper(ctx, paper.ID)
		return nil, err
	}

	return toPaperDTO(paper), nil
}

func (s *PaperService) ListByUser(ctx context.Context, userID string) ([]PaperDTO, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	papers, err := s.paperRepo.FindByUserID(ctx, userUUID)
	if err != nil {
		return nil, err
	}
	result := []PaperDTO{}
	for _, paper := range papers {
		result = append(result, *toPaperDTO(&paper))
	}
	return result, nil
}

func (s *PaperService) GetByID(ctx context.Context, userID, paperID string) (*PaperDTO, error) {
	paperUUID, err := uuid.Parse(paperID)
	if err != nil {
		return nil, ErrInvalidInput
	}

	paper, err := s.paperRepo.FindByID(ctx, paperUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrPaperNotFound
		}
		return nil, err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, ErrInvalidInput
	}
	if paper.UserID != userUUID {
		return nil, ErrPaperNotFound
	}

	return toPaperDTO(paper), nil
}

func (s *PaperService) Delete(ctx context.Context, userID, paperID string) error {
	paperUUID, err := uuid.Parse(paperID)
	if err != nil {
		return ErrInvalidInput
	}

	paper, err := s.paperRepo.FindByID(ctx, paperUUID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrPaperNotFound
		}
		return err
	}

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return ErrInvalidInput
	}
	if paper.UserID != userUUID {
		return ErrPaperNotFound
	}

	// 删除数据库记录
	if err := s.paperRepo.DeletePaper(ctx, paperUUID); err != nil {
		return err
	}

	// 删除文件
	filePath := fmt.Sprintf("%s/%s.pdf", s.uploadDir, paperUUID.String())
	os.Remove(filePath) // 忽略文件删除失败（文件可能不存在）

	return nil
}

func toPaperDTO(paper *model.Paper) *PaperDTO {
	return &PaperDTO{
		ID:         paper.ID.String(),
		Filename:   paper.Filename,
		FileSize:   paper.FileSize,
		Title:      paper.Title,
		Authors:    paper.Authors,
		Year:       paper.Year,
		Venue:      paper.Venue,
		Status:     paper.Status,
		ChunkCount: paper.ChunkCount,
		CreatedAt:  paper.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  paper.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
}
