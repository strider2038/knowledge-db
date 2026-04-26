package index

import "context"

// EmbeddingProvider генерирует векторные представления текста.
type EmbeddingProvider interface {
	Embed(ctx context.Context, texts []string) ([][]float32, error)
}
