package fetcher

import (
	"context"
	"time"
)

// ContentFetcher — интерфейс для извлечения контента из URL.
type ContentFetcher interface {
	Fetch(ctx context.Context, url string) (*FetchResult, error)
}

// FetchResult — результат извлечения контента.
type FetchResult struct {
	Title      string
	Content    string
	SourceDate *time.Time
	Author     string
}

// URLMeta — лёгкие метаданные страницы (title + description из <meta>).
type URLMeta struct {
	Title       string
	Description string
}
