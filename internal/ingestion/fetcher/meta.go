package fetcher

import (
	"context"
	"io"
	"net/http"
	"strings"

	"github.com/muonsoft/errors"
	"golang.org/x/net/html"
)

// FetchURLMeta выполняет лёгкий HTTP GET и извлекает title + description из <meta> тегов.
func FetchURLMeta(ctx context.Context, rawURL string) (*URLMeta, error) {
	return fetchURLMeta(ctx, rawURL, http.DefaultClient)
}

// HTMLMetaFetcher извлекает title/description из HTML-метаданных страницы.
type HTMLMetaFetcher struct {
	httpClient *http.Client
}

// NewHTMLMetaFetcher создаёт fetcher метаданных из HTML.
func NewHTMLMetaFetcher(httpClient *http.Client) *HTMLMetaFetcher {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &HTMLMetaFetcher{httpClient: httpClient}
}

// FetchMeta извлекает URLMeta из HTML-страницы.
func (f *HTMLMetaFetcher) FetchMeta(ctx context.Context, rawURL string) (*URLMeta, error) {
	return fetchURLMeta(ctx, rawURL, f.httpClient)
}

func fetchURLMeta(ctx context.Context, rawURL string, httpClient *http.Client) (*URLMeta, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.Errorf("fetch url meta: create request: %w", err)
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("fetch url meta: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	limited := io.LimitReader(resp.Body, 512*1024)
	doc, err := html.Parse(limited)
	if err != nil {
		return nil, errors.Errorf("fetch url meta: parse html: %w", err)
	}

	meta := &URLMeta{}
	extractMeta(doc, meta)
	meta.Title = normalizeMetaValue(meta.Title)
	meta.Description = normalizeMetaValue(meta.Description)

	return meta, nil
}

func extractMeta(n *html.Node, meta *URLMeta) {
	if n.Type == html.ElementNode {
		switch strings.ToLower(n.Data) {
		case "title":
			if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
				if meta.Title == "" {
					meta.Title = strings.TrimSpace(n.FirstChild.Data)
				}
			}
		case "meta":
			name := attrVal(n, "name")
			property := attrVal(n, "property")
			itemprop := attrVal(n, "itemprop")
			content := attrVal(n, "content")
			switch {
			case strings.EqualFold(name, "description") ||
				strings.EqualFold(property, "og:description") ||
				strings.EqualFold(name, "twitter:description") ||
				strings.EqualFold(itemprop, "description"):
				if meta.Description == "" {
					meta.Description = content
				}
			case strings.EqualFold(property, "og:title") ||
				strings.EqualFold(name, "twitter:title") ||
				strings.EqualFold(itemprop, "name") ||
				strings.EqualFold(name, "application-name"):
				if meta.Title == "" {
					meta.Title = content
				}
			}
		case "body":
			return
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		extractMeta(c, meta)
	}
}

func attrVal(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if strings.EqualFold(a.Key, key) {
			return a.Val
		}
	}

	return ""
}

func normalizeMetaValue(v string) string {
	return strings.Join(strings.Fields(v), " ")
}
