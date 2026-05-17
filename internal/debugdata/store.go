package debugdata

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

const (
	IssueStatusNew           = "new"
	IssueStatusInvestigating = "investigating"
	IssueStatusFixed         = "fixed"
)

var errInvalidIssueFormat = errors.New("invalid issue format")

type IssuePayload struct {
	Title       string         `json:"title"`
	Description string         `json:"description"`
	Page        string         `json:"page"`
	Context     map[string]any `json:"context"`
}

type Issue struct {
	ID          string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	Title       string
	Description string
	Page        string
	Context     map[string]any
	Body        string
}

type TelegramRawRecord struct {
	ReceivedAt time.Time       `json:"received_at"`
	UpdateID   int             `json:"update_id"`
	Payload    json.RawMessage `json:"payload"`
}

type Store struct {
	dataPath string
}

func NewStore(dataPath string) *Store {
	return &Store{dataPath: strings.TrimSpace(dataPath)}
}

func (s *Store) WriteIssue(_ context.Context, payload IssuePayload) (Issue, error) {
	now := time.Now().UTC()
	id := "issue-" + now.Format("20060102-150405.000000000")
	relDir := filepath.Join(".kb", "issues", now.Format("2006"), now.Format("01"), now.Format("02"))
	dir := filepath.Join(s.dataPath, relDir)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return Issue{}, fmt.Errorf("create issues dir: %w", err)
	}

	contextJSON, err := json.MarshalIndent(payload.Context, "", "  ")
	if err != nil {
		return Issue{}, fmt.Errorf("marshal context: %w", err)
	}

	frontmatter := map[string]any{
		"id":          id,
		"status":      IssueStatusNew,
		"created_at":  now.Format(time.RFC3339),
		"updated_at":  now.Format(time.RFC3339),
		"title":       payload.Title,
		"page":        payload.Page,
		"description": payload.Description,
	}
	fm, err := yaml.Marshal(frontmatter)
	if err != nil {
		return Issue{}, fmt.Errorf("marshal frontmatter: %w", err)
	}

	body := fmt.Sprintf("---\n%s---\n\n## Context\n\n```json\n%s\n```\n", string(fm), string(contextJSON))
	path := filepath.Join(dir, id+".md")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		return Issue{}, fmt.Errorf("write issue file: %w", err)
	}

	return Issue{
		ID:          id,
		Status:      IssueStatusNew,
		CreatedAt:   now,
		UpdatedAt:   now,
		Title:       payload.Title,
		Description: payload.Description,
		Page:        payload.Page,
		Context:     payload.Context,
		Body:        body,
	}, nil
}

func (s *Store) UpdateIssueStatus(_ context.Context, issueID, status string) (Issue, error) {
	issue, path, err := s.findIssueByID(issueID)
	if err != nil {
		return Issue{}, err
	}
	now := time.Now().UTC()
	issue.Status = status
	issue.UpdatedAt = now
	contextJSON, err := json.MarshalIndent(issue.Context, "", "  ")
	if err != nil {
		return Issue{}, fmt.Errorf("marshal context: %w", err)
	}
	frontmatter := map[string]any{
		"id":          issue.ID,
		"status":      issue.Status,
		"created_at":  issue.CreatedAt.Format(time.RFC3339),
		"updated_at":  issue.UpdatedAt.Format(time.RFC3339),
		"title":       issue.Title,
		"page":        issue.Page,
		"description": issue.Description,
	}
	fm, err := yaml.Marshal(frontmatter)
	if err != nil {
		return Issue{}, fmt.Errorf("marshal frontmatter: %w", err)
	}
	issue.Body = fmt.Sprintf("---\n%s---\n\n## Context\n\n```json\n%s\n```\n", string(fm), string(contextJSON))
	if err := os.WriteFile(path, []byte(issue.Body), 0o644); err != nil {
		return Issue{}, fmt.Errorf("write issue file: %w", err)
	}

	return issue, nil
}

func (s *Store) ListIssues(_ context.Context, limit int) ([]Issue, error) {
	baseDir := filepath.Join(s.dataPath, ".kb", "issues")
	if limit <= 0 {
		limit = 20
	}
	paths := make([]string, 0, limit)
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		paths = append(paths, path)

		return nil
	})
	sort.Strings(paths)
	for i, j := 0, len(paths)-1; i < j; i, j = i+1, j-1 {
		paths[i], paths[j] = paths[j], paths[i]
	}
	issues := make([]Issue, 0, limit)
	for _, path := range paths {
		issue, err := s.readIssue(path)
		if err != nil {
			continue
		}
		issues = append(issues, issue)
		if len(issues) >= limit {
			break
		}
	}

	return issues, nil
}

func (s *Store) ReadIssue(_ context.Context, issueID string) (Issue, error) {
	issue, _, err := s.findIssueByID(issueID)

	return issue, err
}

func (s *Store) AppendTelegramRaw(_ context.Context, record TelegramRawRecord) error {
	day := record.ReceivedAt.UTC().Format("2006-01-02")
	dir := filepath.Join(s.dataPath, ".kb", "telegram-raw")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create telegram raw dir: %w", err)
	}
	path := filepath.Join(dir, day+".ndjson")
	line, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal telegram record: %w", err)
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open telegram raw file: %w", err)
	}
	defer func() { _ = f.Close() }()
	if _, err := f.Write(append(line, '\n')); err != nil {
		return fmt.Errorf("append telegram raw file: %w", err)
	}

	return nil
}

func (s *Store) CleanupTelegramRaw(_ context.Context, now time.Time, ttl time.Duration) error {
	dir := filepath.Join(s.dataPath, ".kb", "telegram-raw")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}

		return fmt.Errorf("read telegram raw dir: %w", err)
	}
	cutoff := now.UTC().Add(-ttl)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".ndjson" {
			continue
		}
		day := strings.TrimSuffix(e.Name(), ".ndjson")
		t, err := time.Parse("2006-01-02", day)
		if err != nil {
			continue
		}
		if t.Before(cutoff) {
			if err := os.Remove(filepath.Join(dir, e.Name())); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove old telegram raw: %w", err)
			}
		}
	}

	return nil
}

func (s *Store) ReadLastTelegramRaw(_ context.Context, limit int) ([]TelegramRawRecord, error) {
	if limit <= 0 {
		limit = 20
	}
	dir := filepath.Join(s.dataPath, ".kb", "telegram-raw")
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("read telegram raw dir: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".ndjson" {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	records := make([]TelegramRawRecord, 0, limit)
	for i := len(names) - 1; i >= 0 && len(records) < limit; i-- {
		path := filepath.Join(dir, names[i])
		fileRecords, err := readNDJSON(path)
		if err != nil {
			continue
		}
		for j := len(fileRecords) - 1; j >= 0 && len(records) < limit; j-- {
			records = append(records, fileRecords[j])
		}
	}

	return records, nil
}

func readNDJSON(path string) ([]TelegramRawRecord, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := make([]TelegramRawRecord, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var rec TelegramRawRecord
		if err := json.Unmarshal([]byte(line), &rec); err != nil {
			continue
		}
		out = append(out, rec)
	}

	return out, scanner.Err()
}

func (s *Store) findIssueByID(issueID string) (Issue, string, error) {
	baseDir := filepath.Join(s.dataPath, ".kb", "issues")
	var found string
	_ = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		if strings.TrimSuffix(filepath.Base(path), ".md") == issueID {
			found = path
		}

		return nil
	})
	if found == "" {
		return Issue{}, "", os.ErrNotExist
	}
	issue, err := s.readIssue(found)
	if err != nil {
		return Issue{}, "", err
	}

	return issue, found, nil
}

func (s *Store) readIssue(path string) (Issue, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Issue{}, err
	}
	text := string(data)
	parts := strings.SplitN(text, "---\n", 3)
	if len(parts) < 3 {
		return Issue{}, fmt.Errorf("%w: %s", errInvalidIssueFormat, path)
	}
	front := parts[1]
	var fm map[string]string
	if err := yaml.Unmarshal([]byte(front), &fm); err != nil {
		return Issue{}, err
	}
	issue := Issue{
		ID:          fm["id"],
		Status:      fm["status"],
		Title:       fm["title"],
		Page:        fm["page"],
		Description: fm["description"],
		Body:        text,
	}
	issue.CreatedAt, _ = time.Parse(time.RFC3339, fm["created_at"])
	issue.UpdatedAt, _ = time.Parse(time.RFC3339, fm["updated_at"])
	if idx := strings.Index(text, "```json\n"); idx >= 0 {
		end := strings.Index(text[idx+8:], "\n```")
		if end > 0 {
			var ctx map[string]any
			raw := text[idx+8 : idx+8+end]
			_ = json.Unmarshal([]byte(raw), &ctx)
			issue.Context = ctx
		}
	}

	return issue, nil
}
