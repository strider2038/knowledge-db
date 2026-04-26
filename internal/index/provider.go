package index

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/muonsoft/errors"
)

// APIProvider генерирует эмбеддинги через OpenAI-совместимое API.
type APIProvider struct {
	apiURL string
	apiKey string
	model  string
	client *http.Client
}

// NewAPIProvider создаёт APIProvider.
func NewAPIProvider(apiURL, apiKey, model string) *APIProvider {
	return &APIProvider{
		apiURL: apiURL,
		apiKey: apiKey,
		model:  model,
		client: &http.Client{},
	}
}

type embeddingRequest struct {
	Input interface{} `json:"input"`
	Model string      `json:"model"`
}

type embeddingResponse struct {
	Data []struct {
		Embedding []float32 `json:"embedding"`
		Index     int       `json:"index"`
	} `json:"data"`
	Model string `json:"model"`
	Usage struct {
		TotalTokens int `json:"total_tokens"`
	} `json:"usage"`
}

type apiError struct {
	Error struct {
		Message string `json:"message"`
	} `json:"error"`
}

// Embed отправляет тексты на эмбеддинг API и возвращает векторы.
func (p *APIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	reqBody := embeddingRequest{
		Input: texts,
		Model: p.model,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, errors.Errorf("marshal embedding request: %w", err)
	}

	endpoint := p.apiURL + "/v1/embeddings"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, errors.Errorf("create embedding request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.Errorf("embedding API request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("read embedding response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var apiErr apiError
		if jsonErr := json.Unmarshal(body, &apiErr); jsonErr == nil && apiErr.Error.Message != "" {
			return nil, errors.Errorf("embedding API error (status %d): %s", resp.StatusCode, apiErr.Error.Message)
		}

		return nil, errors.Errorf("embedding API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result embeddingResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, errors.Errorf("unmarshal embedding response: %w", err)
	}

	if len(result.Data) != len(texts) {
		return nil, errors.Errorf("embedding API returned %d results, expected %d", len(result.Data), len(texts))
	}

	embeddings := make([][]float32, len(texts))
	for i, d := range result.Data {
		if d.Index != i {
			return nil, fmt.Errorf("embedding API returned out-of-order result: index %d at position %d", d.Index, i)
		}
		embeddings[i] = d.Embedding
	}

	return embeddings, nil
}
