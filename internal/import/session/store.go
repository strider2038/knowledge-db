package session

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/gofrs/uuid/v5"
	"github.com/muonsoft/errors"
	"github.com/strider2038/knowledge-db/internal/import/telegram"
	"github.com/strider2038/knowledge-db/internal/ingestion"
	"github.com/strider2038/knowledge-db/internal/kb"
)

// ErrImportNotConfigured — импорт не настроен (KB_UPLOADS_DIR не задан).
var ErrImportNotConfigured = errors.New("import not configured")

// ErrSessionNotFound — сессия не найдена.
var ErrSessionNotFound = errors.New("session not found")

// ErrNoCurrentItem — нет текущей записи (сессия завершена).
var ErrNoCurrentItem = errors.New("no current item")

// ImportItem — запись для импорта (алиас для совместимости).
type ImportItem = telegram.ImportItem

// Session — состояние сессии импорта.
type Session struct {
	SessionID    string       `json:"session_id"`
	CreatedAt    string       `json:"created_at"`
	Total        int          `json:"total"`
	CurrentIndex int          `json:"current_index"`
	ProcessedIDs []int64      `json:"processed_ids"`
	RejectedIDs  []int64      `json:"rejected_ids"`
	Items        []ImportItem `json:"items"`
}

// CurrentItem возвращает текущую запись или nil.
func (s *Session) CurrentItem() *ImportItem {
	if s.CurrentIndex < 0 || s.CurrentIndex >= len(s.Items) {
		return nil
	}
	item := s.Items[s.CurrentIndex]

	return &item
}

// SessionStore — хранилище сессий импорта.
type SessionStore interface {
	Create(ctx context.Context, items []ImportItem) (*Session, error)
	Get(ctx context.Context, id string) (*Session, error)
	Accept(ctx context.Context, id string, typeHint string) (*kb.Node, *ImportItem, error)
	Reject(ctx context.Context, id string) (*ImportItem, error)
}

type fileStore struct {
	uploadsDir string
	ingester   ingestion.Ingester
}

// NewFileStore создаёт SessionStore с хранением в файлах.
func NewFileStore(uploadsDir string, ingester ingestion.Ingester) SessionStore {
	return &fileStore{uploadsDir: uploadsDir, ingester: ingester}
}

func (s *fileStore) Create(ctx context.Context, items []ImportItem) (*Session, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, err
	}

	sessionID := uuid.Must(uuid.NewV4()).String()
	now := formatISO8601()
	sess := &Session{
		SessionID:    sessionID,
		CreatedAt:    now,
		Total:        len(items),
		CurrentIndex: 0,
		ProcessedIDs: nil,
		RejectedIDs:  nil,
		Items:        items,
	}

	dir := filepath.Dir(s.sessionPath(sessionID))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, errors.Errorf("create session dir: %w", err)
	}

	if err := s.save(sess); err != nil {
		return nil, errors.Errorf("save session: %w", err)
	}

	return sess, nil
}

func (s *fileStore) Get(ctx context.Context, id string) (*Session, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(s.sessionPath(id))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.Errorf("get session: %w", ErrSessionNotFound)
		}

		return nil, errors.Errorf("read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, errors.Errorf("parse session: %w", err)
	}

	return &sess, nil
}

func (s *fileStore) Accept(ctx context.Context, id string, typeHint string) (*kb.Node, *ImportItem, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, nil, err
	}

	sess, err := s.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}

	item := sess.CurrentItem()
	if item == nil {
		return nil, nil, errors.Errorf("accept: %w", ErrNoCurrentItem)
	}

	node, err := s.ingester.IngestText(ctx, ingestion.IngestRequest{
		Text:         item.Text,
		SourceURL:    item.SourceURL,
		SourceAuthor: item.SourceAuthor,
		TypeHint:     typeHint,
	})
	if err != nil {
		return nil, nil, errors.Errorf("ingest: %w", err)
	}

	sess.ProcessedIDs = append(sess.ProcessedIDs, item.ID)
	sess.CurrentIndex++

	if err := s.save(sess); err != nil {
		return nil, nil, errors.Errorf("save session: %w", err)
	}

	next := sess.CurrentItem()

	return node, next, nil
}

func (s *fileStore) Reject(ctx context.Context, id string) (*ImportItem, error) {
	if err := s.checkConfigured(); err != nil {
		return nil, err
	}

	sess, err := s.Get(ctx, id)
	if err != nil {
		return nil, err
	}

	item := sess.CurrentItem()
	if item == nil {
		return nil, errors.Errorf("reject: %w", ErrNoCurrentItem)
	}

	sess.RejectedIDs = append(sess.RejectedIDs, item.ID)
	sess.CurrentIndex++

	if err := s.save(sess); err != nil {
		return nil, errors.Errorf("save session: %w", err)
	}

	next := sess.CurrentItem()

	return next, nil
}

func (s *fileStore) save(sess *Session) error {
	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.sessionPath(sess.SessionID), data, 0o644)
}

func formatISO8601() string {
	return time.Now().UTC().Format(time.RFC3339)
}

func (s *fileStore) checkConfigured() error {
	if s.uploadsDir == "" {
		return ErrImportNotConfigured
	}

	return nil
}

func (s *fileStore) sessionPath(id string) string {
	return filepath.Join(s.uploadsDir, "telegram-import-sessions", id+".json")
}
