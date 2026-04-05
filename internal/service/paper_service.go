package service

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/model"
	"wolfden.website/papermind/internal/pkg/extractor"
	"wolfden.website/papermind/internal/pkg/parser"
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

	// 启动异步处理（不阻塞上传响应）
	go s.processPaper(paper.ID, filePath)

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

// 论文上传后的异步处理流水线
func (s *PaperService) processPaper(paperID uuid.UUID, filePath string) {
	// 防止 panic 扩散
	defer func() {
		if r := recover(); r != nil {
			log.Printf("处理论文 %s 时发生 panic: %v", paperID, r)
			s.updateStatus(context.Background(), paperID, "failed")
		}
	}()

	// 使用 background context，因为请求可能已经结束
	ctx := context.Background()

	// ========== 阶段 1: 文本提取 ==========
	s.updateStatus(ctx, paperID, "extracting")

	pdfContent, err := extractor.ExtractPDF(filePath)
	if err != nil {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s PDF 提取失败: %v", paperID, err)
		return
	}

	// 检查是否提取出了有效文本
	if strings.TrimSpace(pdfContent.FullText) == "" {
		s.updateStatus(ctx, paperID, "failed")
		log.Printf("论文 %s PDF 未提取出文本（可能是扫描件）", paperID)
		return
	}

	// 用提取到的元数据更新论文记录
	s.updateMetadataFromPDF(ctx, paperID, pdfContent.Metadata)

	// ========== 阶段 2: 结构解析 ==========
	s.updateStatus(ctx, paperID, "parsing")

	sections := parser.ParseSections(pdfContent.Pages)

	// 打印解析结果到日志
	log.Printf("论文 %s 解析完成，识别出 %d 个章节:", paperID, len(sections))
	for i, sec := range sections {
		log.Printf("  [%d] type=%s, title=%s, page=%d, contentLen=%d",
			i, sec.Type, sec.Title, sec.StartPage, len(sec.Content))
	}

	// ========== 阶段 3: 切片 + Embedding（后续实现）==========
	// s.updateStatus(ctx, paperID, "chunking")
	// s.updateStatus(ctx, paperID, "embedding")

	// 临时：直接标记为 completed
	s.updateStatus(ctx, paperID, "completed")
}

// updateStatus 更新论文的处理状态
func (s *PaperService) updateStatus(ctx context.Context, paperID uuid.UUID, status string) {
	err := s.paperRepo.UpdateStatus(ctx, paperID, status)
	if err != nil {
		log.Printf("更新论文 %s 状态为 %s 失败: %v", paperID, status, err)
	}
}

// updateMetadataFromPDF 用 PDF 元数据更新论文记录
func (s *PaperService) updateMetadataFromPDF(ctx context.Context, paperID uuid.UUID, meta extractor.PDFMetadata) {
	updates := make(map[string]interface{})

	if meta.Title != "" {
		updates["title"] = meta.Title
	}
	if meta.Author != "" {
		updates["authors"] = meta.Author
	}

	if len(updates) > 0 {
		s.paperRepo.UpdateMetadataIfEmpty(ctx, paperID, updates)
	}
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
