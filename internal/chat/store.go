package chat

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const (
	defaultSessionTTL      = 7 * 24 * time.Hour
	defaultMaxMessages     = 40
	defaultMaxContextRunes = 24000
)

type Store struct {
	db              *sql.DB
	sessionTTL      time.Duration
	maxMessages     int
	maxContextRunes int
}

type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Message struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

type SessionDetails struct {
	Session  Session   `json:"session"`
	Messages []Message `json:"messages"`
}

func NewStore(path string) (*Store, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite pragmas: %w", err)
	}
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS chat_sessions (
	id TEXT PRIMARY KEY,
	title TEXT NOT NULL,
	summary TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	updated_at TEXT NOT NULL,
	expires_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS chat_messages (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	session_id TEXT NOT NULL REFERENCES chat_sessions(id) ON DELETE CASCADE,
	role TEXT NOT NULL,
	content TEXT NOT NULL,
	is_summary INTEGER NOT NULL DEFAULT 0,
	created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_updated ON chat_sessions(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_chat_sessions_expires ON chat_sessions(expires_at);
CREATE INDEX IF NOT EXISTS idx_chat_messages_session ON chat_messages(session_id, id);
`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("sqlite schema: %w", err)
	}

	return &Store{db: db, sessionTTL: defaultSessionTTL, maxMessages: defaultMaxMessages, maxContextRunes: defaultMaxContextRunes}, nil
}

func (s *Store) Close() error { return s.db.Close() }

func (s *Store) CreateSession(ctx context.Context, id, title string) (Session, error) {
	now := time.Now().UTC()
	if strings.TrimSpace(title) == "" {
		title = "Новый чат"
	}
	expires := now.Add(s.sessionTTL)
	_, err := s.db.ExecContext(ctx, `INSERT INTO chat_sessions(id, title, created_at, updated_at, expires_at) VALUES (?, ?, ?, ?, ?)`,
		id, title, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), expires.Format(time.RFC3339Nano))
	if err != nil {
		return Session{}, err
	}
	return Session{ID: id, Title: title, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *Store) ListSessions(ctx context.Context) ([]Session, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, title, created_at, updated_at FROM chat_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]Session, 0)
	for rows.Next() {
		var i Session
		var created, updated string
		if err := rows.Scan(&i.ID, &i.Title, &created, &updated); err != nil {
			return nil, err
		}
		i.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
		i.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)
		items = append(items, i)
	}
	return items, rows.Err()
}

func (s *Store) GetSession(ctx context.Context, id string) (SessionDetails, error) {
	out := SessionDetails{
		Messages: make([]Message, 0),
	}
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, title, created_at, updated_at FROM chat_sessions WHERE id = ?`, id).Scan(&out.Session.ID, &out.Session.Title, &created, &updated)
	if err != nil {
		return out, err
	}
	out.Session.CreatedAt, _ = time.Parse(time.RFC3339Nano, created)
	out.Session.UpdatedAt, _ = time.Parse(time.RFC3339Nano, updated)

	rows, err := s.db.QueryContext(ctx, `SELECT id, role, content, created_at FROM chat_messages WHERE session_id = ? AND is_summary = 0 ORDER BY id ASC`, id)
	if err != nil {
		return out, err
	}
	defer rows.Close()
	for rows.Next() {
		var m Message
		var ts string
		if err := rows.Scan(&m.ID, &m.Role, &m.Content, &ts); err != nil {
			return out, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339Nano, ts)
		out.Messages = append(out.Messages, m)
	}
	return out, rows.Err()
}

func (s *Store) RenameSession(ctx context.Context, id, title string) error {
	title = strings.TrimSpace(title)
	if title == "" {
		return fmt.Errorf("title is required")
	}
	res, err := s.db.ExecContext(ctx, `UPDATE chat_sessions SET title = ?, updated_at = ? WHERE id = ?`, title, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) DeleteSession(ctx context.Context, id string) error {
	res, err := s.db.ExecContext(ctx, `DELETE FROM chat_sessions WHERE id = ?`, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (s *Store) EnsureSession(ctx context.Context, id string) error {
	_, err := s.GetSession(ctx, id)
	return err
}

func (s *Store) AddMessage(ctx context.Context, sessionID, role, content string, isSummary bool) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	summaryInt := 0
	if isSummary {
		summaryInt = 1
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO chat_messages(session_id, role, content, is_summary, created_at) VALUES (?, ?, ?, ?, ?)`, sessionID, role, content, summaryInt, now)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `UPDATE chat_sessions SET updated_at = ?, expires_at = ? WHERE id = ?`, now, time.Now().UTC().Add(s.sessionTTL).Format(time.RFC3339Nano), sessionID)
	if err != nil {
		return err
	}
	if role == "user" && !isSummary {
		if err := s.autoRenameSessionFromFirstMessage(ctx, sessionID, content, now); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) autoRenameSessionFromFirstMessage(ctx context.Context, sessionID, content, now string) error {
	title := deriveTitleFromMessage(content)
	if title == "" {
		return nil
	}
	res, err := s.db.ExecContext(ctx, `
UPDATE chat_sessions
SET title = ?, updated_at = ?
WHERE id = ?
  AND title = 'Новый чат'
  AND (
    SELECT COUNT(*)
    FROM chat_messages
    WHERE session_id = ?
      AND role = 'user'
      AND is_summary = 0
  ) = 1
`, title, now, sessionID, sessionID)
	if err != nil {
		return err
	}
	_, _ = res.RowsAffected()
	return nil
}

func deriveTitleFromMessage(content string) string {
	title := strings.TrimSpace(content)
	if title == "" {
		return ""
	}
	title = strings.Join(strings.Fields(title), " ")
	r := []rune(title)
	if len(r) > 60 {
		title = strings.TrimSpace(string(r[:60])) + "…"
	}
	return title
}

func (s *Store) BuildPromptMessages(ctx context.Context, sessionID string) ([]map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT role, content, is_summary FROM chat_messages WHERE session_id = ? ORDER BY is_summary DESC, id ASC`, sessionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]map[string]string, 0)
	for rows.Next() {
		var role, content string
		var isSummary int
		if err := rows.Scan(&role, &content, &isSummary); err != nil {
			return nil, err
		}
		if isSummary == 1 {
			items = append(items, map[string]string{"role": "system", "content": "Summary of previous dialog: " + content})
			continue
		}
		items = append(items, map[string]string{"role": role, "content": content})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return s.compactPrompt(items), nil
}

func (s *Store) compactPrompt(items []map[string]string) []map[string]string {
	if len(items) <= s.maxMessages {
		return compactByRunes(items, s.maxContextRunes)
	}
	start := len(items) - s.maxMessages
	return compactByRunes(items[start:], s.maxContextRunes)
}

func compactByRunes(items []map[string]string, budget int) []map[string]string {
	if budget <= 0 {
		return items
	}
	total := 0
	for i := len(items) - 1; i >= 0; i-- {
		total += len([]rune(items[i]["content"]))
		if total > budget {
			if i+1 < len(items) {
				return items[i+1:]
			}
			break
		}
	}
	return items
}

func (s *Store) SummarizeAndTrim(ctx context.Context, sessionID string) error {
	rows, err := s.db.QueryContext(ctx, `SELECT id, role, content FROM chat_messages WHERE session_id = ? AND is_summary = 0 ORDER BY id ASC`, sessionID)
	if err != nil {
		return err
	}
	defer rows.Close()
	var messages []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.role, &r.content); err != nil {
			return err
		}
		messages = append(messages, r)
	}
	trimCount := s.summaryTrimCount(messages)
	if trimCount == 0 {
		return nil
	}
	toSummarize := messages[:trimCount]
	parts := make([]string, 0, len(toSummarize))
	for _, m := range toSummarize {
		parts = append(parts, fmt.Sprintf("%s: %s", m.role, truncateRunes(strings.TrimSpace(m.content), 180)))
	}
	summary := strings.Join(parts, "\n")
	if len(summary) > 1200 {
		summary = truncateRunes(summary, 1200)
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `INSERT INTO chat_messages(session_id, role, content, is_summary, created_at) VALUES (?, 'system', ?, 1, ?)`, sessionID, summary, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	ids := make([]int64, 0, len(toSummarize))
	for _, m := range toSummarize {
		ids = append(ids, m.id)
	}
	where := make([]string, 0, len(ids))
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		where = append(where, "?")
		args = append(args, id)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM chat_messages WHERE id IN (`+strings.Join(where, ",")+`)`, args...); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

func (s *Store) summaryTrimCount(messages []rec) int {
	if len(messages) <= 1 {
		return 0
	}

	trimCount := 0
	if len(messages) > s.maxMessages {
		trimCount = len(messages) - s.maxMessages
	}
	if runeTrimCount := trimCountByRunes(messages, s.maxContextRunes); runeTrimCount > trimCount {
		trimCount = runeTrimCount
	}
	if trimCount == 0 {
		return 0
	}
	if trimCount < 4 {
		trimCount = min(4, len(messages)-1)
	}

	return min(trimCount, len(messages)-1)
}

type rec struct {
	id            int64
	role, content string
}

func trimCountByRunes(messages []rec, budget int) int {
	if budget <= 0 {
		return 0
	}

	total := 0
	for i := len(messages) - 1; i >= 0; i-- {
		total += len([]rune(messages[i].content))
		if total > budget {
			return min(i+1, len(messages)-1)
		}
	}

	return 0
}

func (s *Store) CleanupExpired(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `DELETE FROM chat_sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339Nano)); err != nil {
		return err
	}
	_, _ = s.db.ExecContext(ctx, `VACUUM`)
	return nil
}

func truncateRunes(v string, max int) string {
	r := []rune(v)
	if len(r) <= max {
		return v
	}
	return string(r[:max])
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
