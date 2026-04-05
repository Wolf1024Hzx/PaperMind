package extractor

import (
	"fmt"
	"strings"

	pdflib "github.com/ledongthuc/pdf"
)

// PageText 表示 PDF 中一页的文本内容
type PageText struct {
	PageNumber int    // 页码，从 1 开始
	Text       string // 该页的全部文本
}

// PDFContent 表示从 PDF 中提取出的完整内容
type PDFContent struct {
	Pages    []PageText  // 逐页文本
	FullText string      // 所有页文本拼接（用于后续结构解析）
	Metadata PDFMetadata // PDF 内嵌元数据
}

// PDFMetadata 表示从 PDF 文件属性中提取的信息
type PDFMetadata struct {
	Title  string
	Author string
}

// ExtractPDF 从 PDF 文件中提取逐页文本和元数据
func ExtractPDF(filePath string) (*PDFContent, error) {
	// 1. 打开 PDF 文件
	f, r, err := pdflib.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("打开 PDF 失败: %w", err)
	}
	defer f.Close()

	// 2. 提取元数据（可能为空，不影响主流程）
	metadata := extractMetadata(r)

	// 3. 逐页提取文本
	totalPages := r.NumPage()
	if totalPages == 0 {
		return nil, fmt.Errorf("PDF 页数为 0，可能是损坏的文件")
	}

	pages := make([]PageText, 0, totalPages)
	var fullTextBuilder strings.Builder

	for i := 1; i <= totalPages; i++ {
		page := r.Page(i)
		if page.V.IsNull() {
			// 某些页可能无法解析，跳过但记录
			pages = append(pages, PageText{
				PageNumber: i,
				Text:       "",
			})
			continue
		}

		text, err := extractPageText(page)
		if err != nil {
			// 单页提取失败不中断整个流程，记录空文本
			pages = append(pages, PageText{
				PageNumber: i,
				Text:       "",
			})
			continue
		}

		pages = append(pages, PageText{
			PageNumber: i,
			Text:       text,
		})

		fullTextBuilder.WriteString(text)
		fullTextBuilder.WriteString("\n\n") // 页与页之间用双换行分隔
	}

	return &PDFContent{
		Pages:    pages,
		FullText: fullTextBuilder.String(),
		Metadata: metadata,
	}, nil
}

// extractPageText 提取单页文本
// 使用 GetTextByRow 并在单词之间智能添加空格
func extractPageText(page pdflib.Page) (string, error) {
	rows, err := page.GetTextByRow()
	if err != nil {
		return "", err
	}

	var builder strings.Builder
	for _, row := range rows {
		for i, word := range row.Content {
			builder.WriteString(word.S)
			// 如果不是最后一个单词，且当前单词不以标点结尾，添加空格
			if i < len(row.Content)-1 {
				// 检查当前单词是否以标点结尾
				if len(word.S) > 0 {
					lastChar := word.S[len(word.S)-1]
					if lastChar != '.' && lastChar != ',' && lastChar != ';' &&
						lastChar != ':' && lastChar != '!' && lastChar != '?' &&
						lastChar != ')' && lastChar != ']' && lastChar != '}' {
						builder.WriteString(" ")
					}
				}
			}
		}
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String()), nil
}

// extractMetadata 从 PDF 文件属性中提取 title 和 author
func extractMetadata(r *pdflib.Reader) PDFMetadata {
	metadata := PDFMetadata{}

	trailer := r.Trailer()
	infoRef := trailer.Key("Info")
	if infoRef.IsNull() {
		return metadata
	}

	title := infoRef.Key("Title")
	if !title.IsNull() {
		metadata.Title = strings.TrimSpace(title.String())
	}

	author := infoRef.Key("Author")
	if !author.IsNull() {
		metadata.Author = strings.TrimSpace(author.String())
	}

	return metadata
}
