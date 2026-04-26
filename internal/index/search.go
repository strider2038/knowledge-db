package index

import (
	"context"
	"math"
	"sort"
)

// SearchResult — результат векторного поиска.
type SearchResult struct {
	Path       string
	Title      string
	Annotation string
	Score      float32
}

// ChunkSearchResult — результат поиска по чанкам.
type ChunkSearchResult struct {
	NodePath string
	Heading  string
	Content  string
	Score    float32
}

// VectorSearch выполняет поиск по node-level эмбеддингам.
func VectorSearch(ctx context.Context, store *IndexStore, provider EmbeddingProvider, query string, topK int) ([]SearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	embeddings, err := store.GetAllNodeEmbeddings(ctx)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 {
		return nil, nil
	}

	vec, err := provider.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]

	type scored struct {
		path  string
		score float32
	}
	scores := make([]scored, 0, len(embeddings))
	for _, ne := range embeddings {
		s := cosineSimilarity(queryVec, ne.Vector)
		scores = append(scores, scored{path: ne.Path, score: s})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > topK {
		scores = scores[:topK]
	}

	results := make([]SearchResult, len(scores))
	for i, s := range scores {
		results[i] = SearchResult{Path: s.path, Score: s.score}
	}

	return results, nil
}

// ChunkSearch выполняет поиск по chunk-level эмбеддингам.
func ChunkSearch(ctx context.Context, store *IndexStore, provider EmbeddingProvider, query string, topK int) ([]ChunkSearchResult, error) {
	if topK <= 0 {
		topK = 5
	}

	chunks, err := store.GetAllChunkEmbeddings(ctx)
	if err != nil {
		return nil, err
	}
	if len(chunks) == 0 {
		return nil, nil
	}

	vec, err := provider.Embed(ctx, []string{query})
	if err != nil {
		return nil, err
	}
	queryVec := vec[0]

	type scored struct {
		chunk ChunkEmbedding
		score float32
	}
	scores := make([]scored, 0, len(chunks))
	for _, c := range chunks {
		s := cosineSimilarity(queryVec, c.Vector)
		scores = append(scores, scored{chunk: c, score: s})
	}

	sort.Slice(scores, func(i, j int) bool {
		return scores[i].score > scores[j].score
	})

	if len(scores) > topK {
		scores = scores[:topK]
	}

	results := make([]ChunkSearchResult, len(scores))
	for i, s := range scores {
		results[i] = ChunkSearchResult{
			NodePath: s.chunk.NodePath,
			Heading:  s.chunk.Heading,
			Content:  s.chunk.Content,
			Score:    s.score,
		}
	}

	return results, nil
}

func cosineSimilarity(a, b []float32) float32 {
	if len(a) != len(b) {
		return 0
	}

	var dot, normA, normB float32
	for i := range a {
		dot += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dot / float32(math.Sqrt(float64(normA))*math.Sqrt(float64(normB)))
}
