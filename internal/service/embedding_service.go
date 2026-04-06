package service

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// EmbeddingClient 定义 Embedding API 的统一接口
// 方便后续切换不同的 Embedding 提供商
type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

// MockEmbeddingClient 用于开发阶段测试，生成确定性向量
type MockEmbeddingClient struct {
	dim int
}

func NewMockEmbeddingClient(dim int) *MockEmbeddingClient {
	return &MockEmbeddingClient{
		dim: dim,
	}
}

func (c *MockEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embedding := make([]float32, c.dim)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}
	embeddings := make([][]float32, 0, len(texts))
	for range texts {
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func (c *MockEmbeddingClient) Dimension() int {
	return c.dim
}

// QwenEmbeddingClient 阿里云通义千问 Embedding API 客户端
type QwenEmbeddingClient struct {
	apiKey     string
	model      string
	dimension  int
	httpClient *http.Client
}

func NewQwenEmbeddingClient(apiKey, model string) *QwenEmbeddingClient {
	return &QwenEmbeddingClient{
		apiKey:    apiKey,
		model:     model,
		dimension: 1024, // 固定 1024 维，与数据库一致
		httpClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *QwenEmbeddingClient) Dimension() int {
	return c.dimension
}

// Embed 调用通义千问 Embedding API
// texts 长度建议不超过 20（API 限制）
func (c *QwenEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// 构建 contents 数组
	contents := make([]map[string]string, len(texts))
	for i, text := range texts {
		contents[i] = map[string]string{"text": text}
	}

	// 构建请求体
	reqBody := map[string]any{
		"model": c.model,
		"input": map[string]any{
			"contents": contents,
		},
		"parameters": map[string]any{
			"dimension": c.dimension,
		},
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求
	url := "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 Embedding API 失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Embedding API 返回错误 %d: %s", resp.StatusCode, string(respBytes))
	}

	// 解析响应
	var result qwenEmbeddingResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 按 index 排序提取向量
	embeddings := make([][]float32, len(texts))
	for _, item := range result.Output.Embeddings {
		if item.Index < len(embeddings) {
			embeddings[item.Index] = item.Embedding
		}
	}

	// 校验每个向量都有值
	for i, emb := range embeddings {
		if len(emb) == 0 {
			return nil, fmt.Errorf("第 %d 个文本的 Embedding 为空", i)
		}
	}

	return embeddings, nil
}

// qwenEmbeddingResponse 通义千问 Embedding API 响应结构
type qwenEmbeddingResponse struct {
	Output struct {
		Embeddings []struct {
			Index     int       `json:"index"`
			Embedding []float32 `json:"embedding"`
			Type      string    `json:"type"`
		} `json:"embeddings"`
	} `json:"output"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
	RequestID string `json:"request_id"`
}
