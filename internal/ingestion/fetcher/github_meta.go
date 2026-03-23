package fetcher

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/muonsoft/errors"
)

const githubAPIDefaultBaseURL = "https://api.github.com"

// GitHubMetaFetcher извлекает метаданные репозитория через GitHub API.
type GitHubMetaFetcher struct {
	httpClient *http.Client
	apiBaseURL string
}

// NewGitHubMetaFetcher создаёт GitHubMetaFetcher.
func NewGitHubMetaFetcher(httpClient *http.Client) *GitHubMetaFetcher {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &GitHubMetaFetcher{
		httpClient: httpClient,
		apiBaseURL: githubAPIDefaultBaseURL,
	}
}

// FetchMeta возвращает URLMeta для URL репозитория GitHub.
func (f *GitHubMetaFetcher) FetchMeta(ctx context.Context, rawURL string) (*URLMeta, error) {
	owner, repo, err := parseGitHubRepo(rawURL)
	if err != nil {
		return nil, errors.Errorf("github meta fetch: parse repo: %w", err)
	}
	if owner == "" || repo == "" {
		return nil, ErrURLMetaNotSupported
	}

	apiURL := strings.TrimSuffix(f.apiBaseURL, "/") + "/repos/" + owner + "/" + repo
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, errors.Errorf("github meta fetch: create request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "knowledge-db")

	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, errors.Errorf("github meta fetch: request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
		return nil, ErrURLMetaNotSupported
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("github meta fetch: unexpected status %d", resp.StatusCode)
	}

	//nolint:tagliatelle // GitHub API fields are snake_case.
	var payload struct {
		FullName    string   `json:"full_name"`
		Description string   `json:"description"`
		Homepage    string   `json:"homepage"`
		Language    string   `json:"language"`
		Topics      []string `json:"topics"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, errors.Errorf("github meta fetch: decode response: %w", err)
	}

	title := "GitHub - " + owner + "/" + repo
	if payload.FullName != "" {
		title = "GitHub - " + payload.FullName
	}

	var parts []string
	if payload.Description != "" {
		parts = append(parts, normalizeMetaValue(payload.Description))
	}
	if payload.Language != "" {
		parts = append(parts, "Основной язык: "+payload.Language)
	}
	if payload.Homepage != "" {
		parts = append(parts, "Сайт проекта: "+payload.Homepage)
	}
	if len(payload.Topics) > 0 {
		topics := payload.Topics
		if len(topics) > 5 {
			topics = topics[:5]
		}
		parts = append(parts, "Темы: "+strings.Join(topics, ", "))
	}

	return &URLMeta{
		Title:       title,
		Description: strings.Join(parts, ". "),
	}, nil
}

func parseGitHubRepo(rawURL string) (string, string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", errors.Errorf("parse github url: %w", err)
	}
	if !strings.EqualFold(u.Hostname(), "github.com") {
		return "", "", nil
	}

	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(parts) < 2 {
		return "", "", nil
	}
	owner := strings.TrimSpace(parts[0])
	repo := strings.TrimSpace(parts[1])
	if owner == "" || repo == "" {
		return "", "", nil
	}

	repo = strings.TrimSuffix(repo, ".git")
	if isGitHubReservedOwner(owner) || isGitHubReservedRepoSegment(repo) {
		return "", "", nil
	}

	return owner, repo, nil
}

func isGitHubReservedOwner(owner string) bool {
	switch strings.ToLower(owner) {
	case "features", "topics", "collections", "trending", "events", "sponsors", "organizations",
		"orgs", "site", "about", "pricing", "readme", "search", "marketplace", "login", "signup":
		return true
	default:
		return false
	}
}

func isGitHubReservedRepoSegment(segment string) bool {
	switch strings.ToLower(segment) {
	case "", "pulls", "issues", "explore", "new", "notifications", "settings":
		return true
	default:
		return false
	}
}
