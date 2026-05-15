package chat

import (
	"context"
	"time"
)

//nolint:interfacebloat
type Store interface {
	Close() error
	CreateSession(ctx context.Context, id, title string) (Session, error)
	ListSessions(ctx context.Context) ([]Session, error)
	GetSession(ctx context.Context, id string) (SessionDetails, error)
	RenameSession(ctx context.Context, id, title string) error
	DeleteSession(ctx context.Context, id string) error
	EnsureSession(ctx context.Context, id string) error
	AddMessage(ctx context.Context, sessionID, role, content string, isSummary bool) error
	BuildPromptMessages(ctx context.Context, sessionID string) ([]map[string]string, error)
	SummarizeAndTrim(ctx context.Context, sessionID string) error
	CleanupExpired(ctx context.Context) error
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
