package service

import "context"

type EmbeddingClient interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
	Dimension() int
}

type MockEmbeddingClient struct {
	dim int
}

func NewMockEmbeddingClient(dim int) *MockEmbeddingClient {
	return &MockEmbeddingClient{
		dim: dim,
	}
}

func (mock *MockEmbeddingClient) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embedding := make([]float32, mock.dim)
	for i := range embedding {
		embedding[i] = float32(i) * 0.001
	}
	embeddings := make([][]float32, 0, len(texts))
	for range texts {
		embeddings = append(embeddings, embedding)
	}
	return embeddings, nil
}

func (mock *MockEmbeddingClient) Dimension() int {
	return mock.dim
}
