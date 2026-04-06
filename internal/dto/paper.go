package dto

// UploadPaperInput 上传论文输入（handler 通过 multipart form 获取）
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

// PaperDTO 论文返回结果
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
