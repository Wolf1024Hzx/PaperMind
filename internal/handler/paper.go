package handler

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"wolfden.website/papermind/internal/middleware"
	"wolfden.website/papermind/internal/service"
)

type PaperHandler struct {
	paperService *service.PaperService
}

func NewPaperHandler(paperService *service.PaperService) *PaperHandler {
	return &PaperHandler{paperService: paperService}
}

func (h *PaperHandler) RegisterRoutes(router *gin.RouterGroup, redisService *service.RedisService, jwtSecret []byte, jwtTTL time.Duration) {
	paperGroup := router.Group("/papers")
	paperGroup.Use(middleware.RequireAuth(redisService, jwtSecret, jwtTTL))
	paperGroup.POST("", h.UploadPaper)
	paperGroup.GET("", h.ListPapers)
	paperGroup.GET("/:id", h.GetPaperByID)
	paperGroup.DELETE("/:id", h.DeletePaper)
}

func (h *PaperHandler) UploadPaper(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)

	// 获取上传的文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "缺少文件"})
		return
	}

	// 校验文件类型（只允许 PDF、Markdown、纯文本）
	if !isAllowedFile(file.Filename) {
		c.JSON(http.StatusBadRequest, gin.H{"message": "只支持 PDF、Markdown (.md) 和纯文本 (.txt) 文件"})
		return
	}

	// 校验文件大小（限制 50MB）
	if file.Size > 50*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"message": "文件大小不能超过 50MB"})
		return
	}

	// 读取文件内容
	fileContent, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件"})
		return
	}
	defer fileContent.Close()

	fileData := make([]byte, file.Size)
	if _, err := fileContent.Read(fileData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "无法读取文件内容"})
		return
	}

	// 计算文件哈希（SHA-256）
	hash := sha256.Sum256(fileData)
	fileHash := hex.EncodeToString(hash[:])

	// 获取可选的元数据字段
	title := c.PostForm("title")
	authors := c.PostForm("authors")
	venue := c.PostForm("venue")

	var year *int
	if yearStr := c.PostForm("year"); yearStr != "" {
		yearVal, err := strconv.Atoi(yearStr)
		if err == nil {
			year = &yearVal
		}
	}

	input := service.UploadPaperInput{
		Filename: file.Filename,
		FileData: fileData,
		FileSize: file.Size,
		FileHash: fileHash,
		Title:    title,
		Authors:  authors,
		Year:     year,
		Venue:    venue,
	}

	result, err := h.paperService.UploadPaper(c.Request.Context(), userID, input)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, result)
}

func (h *PaperHandler) ListPapers(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)

	result, err := h.paperService.ListByUser(c.Request.Context(), userID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PaperHandler) GetPaperByID(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	paperID := c.Param("id")

	result, err := h.paperService.GetByID(c.Request.Context(), userID, paperID)
	if err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, result)
}

func (h *PaperHandler) DeletePaper(c *gin.Context) {
	userID := c.GetString(middleware.CurrentUserIDKey)
	paperID := c.Param("id")

	if err := h.paperService.Delete(c.Request.Context(), userID, paperID); err != nil {
		h.handleServiceError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *PaperHandler) handleServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrPaperNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": err.Error()})
	case errors.Is(err, service.ErrFileAlreadyExists):
		c.JSON(http.StatusConflict, gin.H{"message": err.Error()})
	case errors.Is(err, gorm.ErrRecordNotFound):
		c.JSON(http.StatusNotFound, gin.H{"message": "论文不存在"})
	default:
		c.JSON(http.StatusInternalServerError, gin.H{"message": "服务器内部错误"})
	}
}

func isAllowedFile(filename string) bool {
	ext := strings.ToLower(filepath.Ext(filename))
	return ext == ".pdf" || ext == ".md" || ext == ".txt"
}
