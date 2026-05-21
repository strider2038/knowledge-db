package index

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

const (
	KeywordIndexModeFTS5 = "fts5"
	KeywordIndexModeScan = "scan"

	keywordIndexModeFTS5 = "fts5"
	keywordIndexModeScan = "scan"
)

type IndexedNode struct {
	NodeID          string
	Path            string
	ContentHash     string
	BodyHash        string
	IndexedAt       time.Time
	NodeEmbeddingID int64
}

type NodeSourceMatch struct {
	NodeID string
	Path   string
}

type Chunk struct {
	ID          int64
	NodeID      string
	NodePath    string
	ChunkIndex  int
	Heading     string
	Content     string
	EmbeddingID int64
}

type NodeSearchDocument struct {
	NodeID          string
	Path            string
	Title           string
	Type            string
	Aliases         []string
	Annotation      string
	Keywords        []string
	SourceURL       string
	SourceKind      string
	ContentProfile  string
	ManualProcessed bool
	Body            string
}

type ChunkSearchDocument struct {
	NodeID     string
	NodePath   string
	ChunkIndex int
	Heading    string
	Content    string
}

type EmbeddingRecord struct {
	ID         int64
	Vector     []float32
	Model      string
	Dimensions int
}

type IndexStatus struct {
	TotalNodes     int
	TotalChunks    int
	EmbeddingModel string
	KeywordIndex   string
	LastIndexedAt  time.Time
	Status         string
}

type NodeEmbedding struct {
	NodeID string
	Path   string
	Vector []float32
}

type ChunkEmbedding struct {
	ID         int64
	NodePath   string
	ChunkIndex int
	Heading    string
	Content    string
	Vector     []float32
}

//nolint:interfacebloat
type Store interface {
	Close() error
	DataPath() string
	KeywordIndexMode() string

	InsertEmbedding(ctx context.Context, vector []float32, model string) (int64, error)
	GetAllEmbeddings(ctx context.Context) ([]EmbeddingRecord, error)
	DeleteEmbedding(ctx context.Context, id int64) error

	UpsertNode(ctx context.Context, nodeID, path, contentHash, bodyHash string, nodeEmbeddingID int64) error
	UpsertNodeSearch(ctx context.Context, doc NodeSearchDocument) error
	UpsertNodeSourceURL(ctx context.Context, nodeID, sourceURL string) error
	DeleteNode(ctx context.Context, path string) error
	DeleteNodeByID(ctx context.Context, nodeID string) error
	GetNodeByPath(ctx context.Context, path string) (*IndexedNode, error)
	GetNodeByID(ctx context.Context, nodeID string) (*IndexedNode, error)
	UpdateNodePath(ctx context.Context, nodeID, newPath string) error
	FindBySourceURL(ctx context.Context, normalizedURL string) (*NodeSourceMatch, error)
	ListAllIndexed(ctx context.Context) ([]IndexedNode, error)

	UpsertChunks(ctx context.Context, nodeID, nodePath string, chunks []Chunk) error
	UpsertChunkSearch(ctx context.Context, doc ChunkSearchDocument) error
	DeleteChunks(ctx context.Context, nodeID, nodePath string) error
	ListChunksByNode(ctx context.Context, nodePath string) ([]Chunk, error)
	GetAllChunkEmbeddings(ctx context.Context) ([]ChunkEmbedding, error)
	GetAllNodeEmbeddings(ctx context.Context) ([]NodeEmbedding, error)

	GetStatus(ctx context.Context, model string) (*IndexStatus, error)
	ClearAll(ctx context.Context) error
	SearchVocabulary(ctx context.Context, opts SearchVocabularyOptions) ([]string, error)

	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func JoinSearchText(parts ...string) string {
	var builder strings.Builder
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if builder.Len() > 0 {
			builder.WriteByte(' ')
		}
		builder.WriteString(part)
	}

	return builder.String()
}
