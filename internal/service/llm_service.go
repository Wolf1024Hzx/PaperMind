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

// ChatMessage 表示一条对话消息
type ChatMessage struct {
	Role    string `json:"role"` // system / user / assistant
	Content string `json:"content"`
}

// ChatResponse LLM 的回复结果
type ChatResponse struct {
	Content          string // LLM 生成的文本
	PromptTokens     int    // 输入消耗的 token
	CompletionTokens int    // 输出消耗的 token
}

// LLMClient LLM API 的统一接口，方便切换不同的提供商
type LLMClient interface {
	Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error)
}

// MockLLMClient 用于开发阶段测试，返回固定回复
type MockLLMClient struct {
	response string
}

func NewMockLLMClient(response string) *MockLLMClient {
	return &MockLLMClient{response: response}
}

func (c *MockLLMClient) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	return &ChatResponse{
		Content:          c.response,
		PromptTokens:     10,
		CompletionTokens: 20,
	}, nil
}

// QwenLLMClient 阿里云通义千问 LLM API 客户端
type QwenLLMClient struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

func NewQwenLLMClient(apiKey, model string) *QwenLLMClient {
	if model == "" {
		model = "qwen3.5-plus" // 默认模型
	}
	return &QwenLLMClient{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // LLM 调用可能较慢
		},
	}
}

// Chat 调用通义千问 LLM API
func (c *QwenLLMClient) Chat(ctx context.Context, messages []ChatMessage) (*ChatResponse, error) {
	// 构建请求体（OpenAI 兼容格式）
	reqBody := map[string]any{
		"model":    c.model,
		"messages": messages,
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("序列化请求失败: %w", err)
	}

	// 创建 HTTP 请求（OpenAI 兼容端点）
	url := "https://dashscope.aliyuncs.com/compatible-mode/v1/chat/completions"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 LLM API 失败: %w", err)
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("LLM API 返回错误 %d: %s", resp.StatusCode, string(respBytes))
	}

	// 解析响应
	var result qwenChatResponse
	if err := json.Unmarshal(respBytes, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	// 提取回复内容
	content := ""
	if len(result.Choices) > 0 {
		content = result.Choices[0].Message.Content
	}

	return &ChatResponse{
		Content:          content,
		PromptTokens:     result.Usage.PromptTokens,
		CompletionTokens: result.Usage.CompletionTokens,
	}, nil
}

// qwenChatResponse 通义千问 Chat API 响应结构（OpenAI 兼容格式）
type qwenChatResponse struct {
	Choices []struct {
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
	Model string `json:"model"`
	ID    string `json:"id"`
}
