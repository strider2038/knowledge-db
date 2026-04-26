package index

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	"github.com/muonsoft/errors"
	_ "modernc.org/sqlite"
)

// IndexedNode — запись о проиндексированной ноде.
type IndexedNode struct {
	Path          string
	ContentHash   string
	BodyHash      string
	IndexedAt     time.Time
	NodeEmbeddingID int64
}

// Chunk — фрагмент тела статьи.
type Chunk struct {
	ID          int64
	NodePath    string
	ChunkIndex  int
	Heading     string
	Content     string
	EmbeddingID int64
}

// EmbeddingRecord — запись эмбеддинга.
type EmbeddingRecord struct {
	ID         int64
	Vector     []float32
	Model      string
	Dimensions int
}

// IndexStatus — состояние индекса.
type IndexStatus struct {
	TotalNodes   int
	TotalChunks  int
	EmbeddingModel string
	LastIndexedAt time.Time
	Status       string
}

// IndexStore управляет SQLite-индексом эмбеддингов и чанков.
type IndexStore struct {
	db *sql.DB
}

// NewIndexStore создаёт IndexStore и применяет миграции.
func NewIndexStore(dbPath string) (*IndexStore, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, errors.Errorf("open index database: %w", err)
	}

	store := &IndexStore{db: db}
	if err := store.migrate(); err != nil {
		db.Close()

		return nil, errors.Errorf("migrate index database: %w", err)
	}

	return store, nil
}

// Close закрывает соединение с базой.
func (s *IndexStore) Close() error {
	return s.db.Close()
}

func (s *IndexStore) migrate() error {
	_, err := s.db.Exec("PRAGMA journal_mode=WAL")
	if err != nil {
		return errors.Errorf("set WAL mode: %w", err)
	}

	schema := `
	CREATE TABLE IF NOT EXISTS embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vector BLOB NOT NULL,
		model TEXT NOT NULL,
		dimensions INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS indexed_nodes (
		path TEXT PRIMARY KEY,
		content_hash TEXT NOT NULL,
		body_hash TEXT NOT NULL DEFAULT '',
		indexed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		node_embedding_id INTEGER NOT NULL,
		FOREIGN KEY (node_embedding_id) REFERENCES embeddings(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_path TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		heading TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL,
		embedding_id INTEGER NOT NULL,
		UNIQUE(node_path, chunk_index),
		FOREIGN KEY (node_path) REFERENCES indexed_nodes(path) ON DELETE CASCADE,
		FOREIGN KEY (embedding_id) REFERENCES embeddings(id) ON DELETE CASCADE
	);`

	if _, err := s.db.Exec(schema); err != nil {
		return errors.Errorf("create schema: %w", err)
	}

	return nil
}

// InsertEmbedding вставляет эмбеддинг и возвращает его ID.
func (s *IndexStore) InsertEmbedding(ctx context.Context, vector []float32, model string) (int64, error) {
	blob := encodeVector(vector)
	dims := len(vector)

	var id int64
	err := s.queryRowContext(ctx, `
		INSERT INTO embeddings (vector, model, dimensions) VALUES (?, ?, ?)
		RETURNING id`,
		blob, model, dims,
	).Scan(&id)
	if err != nil {
		return 0, errors.Errorf("insert embedding: %w", err)
	}

	return id, nil
}

// GetAllEmbeddings возвращает все эмбеддинги.
func (s *IndexStore) GetAllEmbeddings(ctx context.Context) ([]EmbeddingRecord, error) {
	rows, err := s.queryContext(ctx, `SELECT id, vector, model, dimensions FROM embeddings`)
	if err != nil {
		return nil, errors.Errorf("get all embeddings: %w", err)
	}
	defer rows.Close()

	var records []EmbeddingRecord
	for rows.Next() {
		var rec EmbeddingRecord
		var blob []byte
		if err := rows.Scan(&rec.ID, &blob, &rec.Model, &rec.Dimensions); err != nil {
			return nil, errors.Errorf("scan embedding: %w", err)
		}
		rec.Vector = decodeVector(blob)
		records = append(records, rec)
	}

	return records, rows.Err()
}

// DeleteEmbedding удаляет эмбеддинг по ID.
func (s *IndexStore) DeleteEmbedding(ctx context.Context, id int64) error {
	_, err := s.execContext(ctx, `DELETE FROM embeddings WHERE id = ?`, id)
	if err != nil {
		return errors.Errorf("delete embedding: %w", err)
	}

	return nil
}

// UpsertNode вставляет или обновляет ноду в индексе.
func (s *IndexStore) UpsertNode(ctx context.Context, path, contentHash, bodyHash string, nodeEmbeddingID int64) error {
	_, err := s.execContext(ctx, `
		INSERT INTO indexed_nodes (path, content_hash, body_hash, indexed_at, node_embedding_id)
		VALUES (?, ?, ?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(path) DO UPDATE SET
			content_hash = excluded.content_hash,
			body_hash = excluded.body_hash,
			indexed_at = CURRENT_TIMESTAMP,
			node_embedding_id = excluded.node_embedding_id`,
		path, contentHash, bodyHash, nodeEmbeddingID,
	)
	if err != nil {
		return errors.Errorf("upsert node: %w", err)
	}

	return nil
}

// DeleteNode удаляет ноду и её чанки из индекса.
func (s *IndexStore) DeleteNode(ctx context.Context, path string) error {
	_, err := s.execContext(ctx, `DELETE FROM indexed_nodes WHERE path = ?`, path)
	if err != nil {
		return errors.Errorf("delete node: %w", err)
	}

	return nil
}

// GetNodeByPath возвращает проиндексированную ноду по пути.
func (s *IndexStore) GetNodeByPath(ctx context.Context, path string) (*IndexedNode, error) {
	var node IndexedNode
	err := s.queryRowContext(ctx, `
		SELECT path, content_hash, body_hash, indexed_at, node_embedding_id
		FROM indexed_nodes WHERE path = ?`, path,
	).Scan(&node.Path, &node.ContentHash, &node.BodyHash, &node.IndexedAt, &node.NodeEmbeddingID)
	if err != nil {
		return nil, err
	}

	return &node, nil
}

// ListAllIndexed возвращает все проиндексированные ноды.
func (s *IndexStore) ListAllIndexed(ctx context.Context) ([]IndexedNode, error) {
	rows, err := s.queryContext(ctx, `
		SELECT path, content_hash, body_hash, indexed_at, node_embedding_id
		FROM indexed_nodes`)
	if err != nil {
		return nil, errors.Errorf("list indexed nodes: %w", err)
	}
	defer rows.Close()

	var nodes []IndexedNode
	for rows.Next() {
		var node IndexedNode
		if err := rows.Scan(&node.Path, &node.ContentHash, &node.BodyHash, &node.IndexedAt, &node.NodeEmbeddingID); err != nil {
			return nil, errors.Errorf("scan indexed node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

// UpsertChunks заменяет все чанки ноды.
func (s *IndexStore) UpsertChunks(ctx context.Context, nodePath string, chunks []Chunk) error {
	if err := s.DeleteChunks(ctx, nodePath); err != nil {
		return err
	}

	for _, c := range chunks {
		_, err := s.execContext(ctx, `
			INSERT INTO chunks (node_path, chunk_index, heading, content, embedding_id)
			VALUES (?, ?, ?, ?, ?)`,
			nodePath, c.ChunkIndex, c.Heading, c.Content, c.EmbeddingID,
		)
		if err != nil {
			return errors.Errorf("insert chunk: %w", err)
		}
	}

	return nil
}

// DeleteChunks удаляет все чанки ноды.
func (s *IndexStore) DeleteChunks(ctx context.Context, nodePath string) error {
	_, err := s.execContext(ctx, `DELETE FROM chunks WHERE node_path = ?`, nodePath)
	if err != nil {
		return errors.Errorf("delete chunks: %w", err)
	}

	return nil
}

// ListChunksByNode возвращает все чанки ноды.
func (s *IndexStore) ListChunksByNode(ctx context.Context, nodePath string) ([]Chunk, error) {
	rows, err := s.queryContext(ctx, `
		SELECT id, node_path, chunk_index, heading, content, embedding_id
		FROM chunks WHERE node_path = ? ORDER BY chunk_index`, nodePath,
	)
	if err != nil {
		return nil, errors.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []Chunk
	for rows.Next() {
		var c Chunk
		if err := rows.Scan(&c.ID, &c.NodePath, &c.ChunkIndex, &c.Heading, &c.Content, &c.EmbeddingID); err != nil {
			return nil, errors.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

// GetAllChunkEmbeddings возвращает все чанки с их эмбеддингами.
func (s *IndexStore) GetAllChunkEmbeddings(ctx context.Context) ([]ChunkEmbedding, error) {
	rows, err := s.queryContext(ctx, `
		SELECT c.id, c.node_path, c.chunk_index, c.heading, c.content, e.vector
		FROM chunks c
		JOIN embeddings e ON c.embedding_id = e.id`)
	if err != nil {
		return nil, errors.Errorf("get chunk embeddings: %w", err)
	}
	defer rows.Close()

	var results []ChunkEmbedding
	for rows.Next() {
		var ce ChunkEmbedding
		var blob []byte
		if err := rows.Scan(&ce.ID, &ce.NodePath, &ce.ChunkIndex, &ce.Heading, &ce.Content, &blob); err != nil {
			return nil, errors.Errorf("scan chunk embedding: %w", err)
		}
		ce.Vector = decodeVector(blob)
		results = append(results, ce)
	}

	return results, rows.Err()
}

// GetAllNodeEmbeddings возвращает все ноды с их эмбеддингами.
func (s *IndexStore) GetAllNodeEmbeddings(ctx context.Context) ([]NodeEmbedding, error) {
	rows, err := s.queryContext(ctx, `
		SELECT n.path, e.vector
		FROM indexed_nodes n
		JOIN embeddings e ON n.node_embedding_id = e.id`)
	if err != nil {
		return nil, errors.Errorf("get node embeddings: %w", err)
	}
	defer rows.Close()

	var results []NodeEmbedding
	for rows.Next() {
		var ne NodeEmbedding
		var blob []byte
		if err := rows.Scan(&ne.Path, &blob); err != nil {
			return nil, errors.Errorf("scan node embedding: %w", err)
		}
		ne.Vector = decodeVector(blob)
		results = append(results, ne)
	}

	return results, rows.Err()
}

// GetStatus возвращает статус индекса.
func (s *IndexStore) GetStatus(ctx context.Context, model string) (*IndexStatus, error) {
	status := &IndexStatus{EmbeddingModel: model, Status: "ready"}

	err := s.queryRowContext(ctx, `SELECT COUNT(*) FROM indexed_nodes`).Scan(&status.TotalNodes)
	if err != nil {
		return nil, errors.Errorf("count nodes: %w", err)
	}

	err = s.queryRowContext(ctx, `SELECT COUNT(*) FROM chunks`).Scan(&status.TotalChunks)
	if err != nil {
		return nil, errors.Errorf("count chunks: %w", err)
	}

	var lastIndexed sql.NullString
	err = s.queryRowContext(ctx, `SELECT MAX(indexed_at) FROM indexed_nodes`).Scan(&lastIndexed)
	if err != nil {
		return nil, errors.Errorf("last indexed: %w", err)
	}
	if lastIndexed.Valid {
		status.LastIndexedAt, _ = time.Parse("2006-01-02 15:04:05", lastIndexed.String)
	}

	return status, nil
}

// ClearAll удаляет все данные из индекса.
func (s *IndexStore) ClearAll(ctx context.Context) error {
	if _, err := s.execContext(ctx, `DELETE FROM chunks`); err != nil {
		return errors.Errorf("clear chunks: %w", err)
	}
	if _, err := s.execContext(ctx, `DELETE FROM indexed_nodes`); err != nil {
		return errors.Errorf("clear nodes: %w", err)
	}
	if _, err := s.execContext(ctx, `DELETE FROM embeddings`); err != nil {
		return errors.Errorf("clear embeddings: %w", err)
	}

	return nil
}

func (s *IndexStore) execContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := s.db.ExecContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query, err)
	}

	return result, nil
}

func (s *IndexStore) queryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query, err)
	}

	return rows, nil
}

func (s *IndexStore) queryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func encodeVector(v []float32) []byte {
	buf := make([]byte, len(v)*4)
	for i, f := range v {
		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(f))
	}

	return buf
}

func decodeVector(b []byte) []float32 {
	n := len(b) / 4
	v := make([]float32, n)
	for i := range v {
		v[i] = math.Float32frombits(binary.LittleEndian.Uint32(b[i*4:]))
	}

	return v
}

// NodeEmbedding — нода с эмбеддингом для поиска.
type NodeEmbedding struct {
	Path   string
	Vector []float32
}

// ChunkEmbedding — чанк с эмбеддингом для поиска.
type ChunkEmbedding struct {
	ID         int64
	NodePath   string
	ChunkIndex int
	Heading    string
	Content    string
	Vector     []float32
}
