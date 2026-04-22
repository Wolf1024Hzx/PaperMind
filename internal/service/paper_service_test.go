package service

import (
	"context"
	"errors"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/dto"
	"wolfden.website/papermind/internal/repository"
)

// setupTestDB 创建测试用的 mock 数据库连接
func setupTestDB(t *testing.T) (*gorm.DB, sqlmock.Sqlmock) {
	sqlDB, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: sqlDB,
	}), &gorm.Config{
		SkipDefaultTransaction: true,
	})
	if err != nil {
		t.Fatalf("创建 gorm DB 失败: %v", err)
	}

	return gormDB, mock
}

// TestUploadPaper_DifferentUsersCanUploadSameFile 测试不同用户可以上传相同文件
// 这是修复的核心测试用例
func TestUploadPaper_DifferentUsersCanUploadSameFile(t *testing.T) {
	db, mock := setupTestDB(t)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()
	fileHash := "abc123def456"
	paperIDA := uuid.New()
	paperIDB := uuid.New()

	paperRepo := repository.NewPaperRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	embeddingClient := NewMockEmbeddingClient(1024)

	svc := NewPaperService(
		paperRepo,
		chunkRepo,
		embeddingClient,
		t.TempDir(),
		20,
		4,
	)

	input := dto.UploadPaperInput{
		Filename: "test.pdf",
		FileData: []byte("test content"),
		FileSize: 12,
		FileHash: fileHash,
	}

	// 用户A上传文件 - 期望成功
	// 1. 查询用户A是否已上传该文件
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userA, fileHash, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))
	// 2. 创建论文记录 (SkipDefaultTransaction=true 时不期望事务)
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "papers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paperIDA))

	_, err := svc.UploadPaper(ctx, userA.String(), input)
	if err != nil {
		t.Errorf("用户A上传文件失败: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("用户A上传未满足的期望: %v", err)
	}

	// 用户B上传相同文件 - 期望成功（这是修复的核心场景）
	// 1. 查询用户B是否已上传该文件（应该返回未找到）
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userB, fileHash, 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))
	// 2. 创建论文记录
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "papers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paperIDB))

	_, err = svc.UploadPaper(ctx, userB.String(), input)
	if err != nil {
		t.Errorf("用户B上传相同文件失败: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("用户B上传未满足的期望: %v", err)
	}
}

// TestUploadPaper_SameUserCannotUploadSameFileTwice 测试同一用户不能重复上传相同文件
func TestUploadPaper_SameUserCannotUploadSameFileTwice(t *testing.T) {
	db, mock := setupTestDB(t)
	ctx := context.Background()

	userA := uuid.New()
	fileHash := "abc123def456"
	existingPaperID := uuid.New()

	paperRepo := repository.NewPaperRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	embeddingClient := NewMockEmbeddingClient(1024)

	svc := NewPaperService(
		paperRepo,
		chunkRepo,
		embeddingClient,
		t.TempDir(),
		20,
		4,
	)

	input := dto.UploadPaperInput{
		Filename: "test.pdf",
		FileData: []byte("test content"),
		FileSize: 12,
		FileHash: fileHash,
	}

	// 用户A尝试上传已存在的文件
	// 查询返回已存在的论文
	rows := sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}).
		AddRow(existingPaperID, userA, "test.pdf", 12, fileHash, "", "", nil, "", "", 0, "pending", nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userA, fileHash, 1).
		WillReturnRows(rows)

	_, err := svc.UploadPaper(ctx, userA.String(), input)
	if err == nil {
		t.Error("期望返回 ErrFileAlreadyExists 错误，但返回 nil")
	}
	if !errors.Is(err, ErrFileAlreadyExists) {
		t.Errorf("期望错误为 ErrFileAlreadyExists，实际为: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

// TestUploadPaper_DifferentFilesFromSameUser 测试同一用户可以上传不同文件
func TestUploadPaper_DifferentFilesFromSameUser(t *testing.T) {
	db, mock := setupTestDB(t)
	ctx := context.Background()

	userA := uuid.New()
	paperID1 := uuid.New()
	paperID2 := uuid.New()

	paperRepo := repository.NewPaperRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	embeddingClient := NewMockEmbeddingClient(1024)

	svc := NewPaperService(
		paperRepo,
		chunkRepo,
		embeddingClient,
		t.TempDir(),
		20,
		4,
	)

	input1 := dto.UploadPaperInput{
		Filename: "test1.pdf",
		FileData: []byte("content 1"),
		FileSize: 9,
		FileHash: "hash1",
	}

	input2 := dto.UploadPaperInput{
		Filename: "test2.pdf",
		FileData: []byte("content 2"),
		FileSize: 9,
		FileHash: "hash2",
	}

	// 用户A上传第一个文件
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userA, "hash1", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "papers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paperID1))

	_, err := svc.UploadPaper(ctx, userA.String(), input1)
	if err != nil {
		t.Errorf("上传第一个文件失败: %v", err)
	}

	// 用户A上传第二个文件（不同哈希）
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1 AND file_hash = $2 ORDER BY "papers"."id" LIMIT $3`)).
		WithArgs(userA, "hash2", 1).
		WillReturnRows(sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}))
	mock.ExpectQuery(regexp.QuoteMeta(`INSERT INTO "papers"`)).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(paperID2))

	_, err = svc.UploadPaper(ctx, userA.String(), input2)
	if err != nil {
		t.Errorf("上传第二个文件失败: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

// TestListByUser 测试用户只能看到自己的论文
func TestListByUser(t *testing.T) {
	db, mock := setupTestDB(t)
	ctx := context.Background()

	userA := uuid.New()
	paperID1 := uuid.New()
	paperID2 := uuid.New()

	paperRepo := repository.NewPaperRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	embeddingClient := NewMockEmbeddingClient(1024)

	svc := NewPaperService(
		paperRepo,
		chunkRepo,
		embeddingClient,
		t.TempDir(),
		20,
		4,
	)

	// 查询用户A的论文
	rows := sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}).
		AddRow(paperID1, userA, "paper1.pdf", 1024, "hash1", "Paper 1", "Author", 2024, "Venue", "", 10, "completed", nil, nil).
		AddRow(paperID2, userA, "paper2.pdf", 2048, "hash2", "Paper 2", "Author", 2024, "Venue", "", 20, "completed", nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE user_id = $1`)).
		WithArgs(userA).
		WillReturnRows(rows)

	papers, err := svc.ListByUser(ctx, userA.String())
	if err != nil {
		t.Errorf("查询论文列表失败: %v", err)
	}
	if len(papers) != 2 {
		t.Errorf("期望返回 2 篇论文，实际返回 %d 篇", len(papers))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}

// TestGetByID_UserIsolation 测试用户无法访问其他用户的论文
func TestGetByID_UserIsolation(t *testing.T) {
	db, mock := setupTestDB(t)
	ctx := context.Background()

	userA := uuid.New()
	userB := uuid.New()
	paperID := uuid.New()

	paperRepo := repository.NewPaperRepository(db)
	chunkRepo := repository.NewChunkRepository(db)
	embeddingClient := NewMockEmbeddingClient(1024)

	svc := NewPaperService(
		paperRepo,
		chunkRepo,
		embeddingClient,
		t.TempDir(),
		20,
		4,
	)

	// 用户A的论文
	rows := sqlmock.NewRows([]string{"id", "user_id", "filename", "file_size", "file_hash", "title", "authors", "year", "venue", "abstract", "chunk_count", "status", "created_at", "updated_at"}).
		AddRow(paperID, userA, "paper.pdf", 1024, "hash", "Paper", "Author", 2024, "Venue", "", 10, "completed", nil, nil)

	mock.ExpectQuery(regexp.QuoteMeta(`SELECT * FROM "papers" WHERE id = $1 ORDER BY "papers"."id" LIMIT $2`)).
		WithArgs(paperID, 1).
		WillReturnRows(rows)

	// 用户B尝试访问用户A的论文
	_, err := svc.GetByID(ctx, userB.String(), paperID.String())
	if err == nil {
		t.Error("期望返回 ErrPaperNotFound 错误，但返回 nil")
	}
	if !errors.Is(err, ErrPaperNotFound) {
		t.Errorf("期望错误为 ErrPaperNotFound，实际为: %v", err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("未满足的期望: %v", err)
	}
}
