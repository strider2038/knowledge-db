package fetcher

import (
	"context"
	"net/url"
	"strings"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	htmltomarkdown "github.com/JohannesKaufmann/html-to-markdown/v2"
	"github.com/muonsoft/errors"
)

// ReadabilityFetcher извлекает контент через HTTP GET + go-readability + html-to-markdown.
// Используется как fallback при недоступности Jina.
type ReadabilityFetcher struct {
	timeout time.Duration
}

// NewReadabilityFetcher создаёт ReadabilityFetcher с указанным таймаутом.
func NewReadabilityFetcher(timeout time.Duration) *ReadabilityFetcher {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}

	return &ReadabilityFetcher{timeout: timeout}
}

// Fetch выполняет HTTP GET, извлекает основной контент и конвертирует в markdown.
func (f *ReadabilityFetcher) Fetch(ctx context.Context, rawURL string) (*FetchResult, error) {
	_, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.Errorf("readability fetch: parse url: %w", err)
	}

	article, err := readability.FromURL(rawURL, f.timeout)
	if err != nil {
		return nil, errors.Errorf("readability fetch: %w", err)
	}

	var htmlBuf strings.Builder
	if err := article.RenderHTML(&htmlBuf); err != nil {
		return nil, errors.Errorf("readability fetch: render html: %w", err)
	}

	mdContent, err := htmltomarkdown.ConvertString(htmlBuf.String())
	if err != nil {
		return nil, errors.Errorf("readability fetch: html to markdown: %w", err)
	}

	result := &FetchResult{
		Title:   article.Title(),
		Content: strings.TrimSpace(mdContent),
		Author:  article.Byline(),
	}
	if t, err := article.PublishedTime(); err == nil {
		result.SourceDate = &t
	}

	return result, nil
}
