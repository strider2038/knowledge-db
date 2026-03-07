package fetcher

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/muonsoft/errors"
)

// JinaFetcher извлекает контент через Jina Reader API (https://r.jina.ai/{url}).
// Возвращает markdown-контент с title и метаданными.
type JinaFetcher struct {
	apiKey     string
	httpClient *http.Client
}

// NewJinaFetcher создаёт JinaFetcher.
// apiKey опционален — без него запросы проходят с лимитами бесплатного тарифа.
func NewJinaFetcher(apiKey string, httpClient *http.Client) *JinaFetcher {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &JinaFetcher{apiKey: apiKey, httpClient: httpClient}
}

// Fetch выполняет GET-запрос к https://r.jina.ai/{url} и возвращает FetchResult.
func (f *JinaFetcher) Fetch(ctx context.Context, url string) (*FetchResult, error) {
	jinaURL := "https://r.jina.ai/" + url
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jinaURL, nil)
	if err != nil {
		return nil, errors.Errorf("jina fetch: create request: %w", err)
	}
	req.Header.Set("Accept", "text/plain")
	if f.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+f.apiKey)
	}

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("jina fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("jina fetch: unexpected status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Errorf("jina fetch: read body: %w", err)
	}

	content := strings.TrimSpace(string(body))
	title := extractJinaTitle(content)

	return &FetchResult{
		Title:   title,
		Content: content,
	}, nil
}

// extractJinaTitle извлекает title из первой строки markdown (# Title).
func extractJinaTitle(content string) string {
	for _, line := range strings.SplitN(content, "\n", 10) {
		line = strings.TrimSpace(line)
		if title, ok := strings.CutPrefix(line, "# "); ok {
			return title
		}
	}

	return ""
}
