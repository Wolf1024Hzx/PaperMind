package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	apiKey := os.Getenv("DASHSCOPE_API_KEY")
	if apiKey == "" {
		fmt.Println("请设置环境变量 DASHSCOPE_API_KEY")
		return
	}

	// 请求体
	reqBody := map[string]any{
		"model": "qwen3-vl-embedding",
		"input": map[string]any{
			"contents": []map[string]string{
				{"text": "Attention Is All You Need"},
			},
		},
		"parameters": map[string]any{
			"dimension": 1024,
		},
	}

	bodyBytes, _ := json.Marshal(reqBody)

	// 创建请求
	url := "https://dashscope.aliyuncs.com/api/v1/services/embeddings/multimodal-embedding/multimodal-embedding"
	req, _ := http.NewRequest("POST", url, bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// 发送请求
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Printf("请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()

	respBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != 200 {
		fmt.Printf("API 返回错误 %d: %s\n", resp.StatusCode, string(respBytes))
		return
	}

	// 解析响应
	var result struct {
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

	json.Unmarshal(respBytes, &result)

	fmt.Printf("请求成功! Request ID: %s\n", result.RequestID)
	fmt.Printf("Token 用量: %d\n", result.Usage.TotalTokens)
	if len(result.Output.Embeddings) > 0 {
		emb := result.Output.Embeddings[0]
		fmt.Printf("向量维度: %d\n", len(emb.Embedding))
		fmt.Printf("向量前5个值: %v\n", emb.Embedding[:5])
	}
}
