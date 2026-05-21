package sqlite

import (
	"context"
	"database/sql"
	"encoding/binary"
	stderrors "errors"
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/muonsoft/errors"
	_ "modernc.org/sqlite"

	"github.com/strider2038/knowledge-db/internal/index"
)

type Store struct {
	db               *sql.DB
	dbPath           string
	keywordIndexMode string
}

func NewStore(ctx context.Context, dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, errors.Errorf("open index database: %w", err)
	}

	store := &Store{db: db, dbPath: dbPath}
	if err := store.migrate(ctx); err != nil {
		db.Close()

		return nil, errors.Errorf("migrate index database: %w", err)
	}

	return store, nil
}

func (s *Store) DataPath() string {
	if s.dbPath == "" || s.dbPath == ":memory:" {
		return ""
	}
	if !strings.Contains(s.dbPath, "/") {
		return ""
	}

	return filepath.Dir(filepath.Dir(s.dbPath))
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) KeywordIndexMode() string {
	if s.keywordIndexMode == "" {
		return index.KeywordIndexModeScan
	}

	return s.keywordIndexMode
}

func (s *Store) SetKeywordIndexModeForTest(mode string) {
	s.keywordIndexMode = mode
}

func (s *Store) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", query, err)
	}

	return rows, nil
}

func (s *Store) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s *Store) InsertEmbedding(ctx context.Context, vector []float32, model string) (int64, error) {
	blob := encodeVector(vector)
	dims := len(vector)

	var id int64
	err := s.QueryRowContext(ctx, `
		INSERT INTO embeddings (vector, model, dimensions) VALUES (?, ?, ?)
		RETURNING id`,
		blob, model, dims,
	).Scan(&id)
	if err != nil {
		return 0, errors.Errorf("insert embedding: %w", err)
	}

	return id, nil
}

func (s *Store) GetAllEmbeddings(ctx context.Context) ([]index.EmbeddingRecord, error) {
	rows, err := s.QueryContext(ctx, `SELECT id, vector, model, dimensions FROM embeddings`)
	if err != nil {
		return nil, errors.Errorf("get all embeddings: %w", err)
	}
	defer rows.Close()

	var records []index.EmbeddingRecord
	for rows.Next() {
		var rec index.EmbeddingRecord
		var blob []byte
		if err := rows.Scan(&rec.ID, &blob, &rec.Model, &rec.Dimensions); err != nil {
			return nil, errors.Errorf("scan embedding: %w", err)
		}
		rec.Vector = decodeVector(blob)
		records = append(records, rec)
	}

	return records, rows.Err()
}

func (s *Store) DeleteEmbedding(ctx context.Context, id int64) error {
	err := s.execContext(ctx, `DELETE FROM embeddings WHERE id = ?`, id)
	if err != nil {
		return errors.Errorf("delete embedding: %w", err)
	}

	return nil
}

func (s *Store) UpsertNode(ctx context.Context, nodeID, path, contentHash, bodyHash string, nodeEmbeddingID int64) error {
	err := s.execContext(ctx, `
		INSERT INTO indexed_nodes (node_id, path, content_hash, body_hash, indexed_at, node_embedding_id)
		VALUES (?, ?, ?, ?, CURRENT_TIMESTAMP, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			path = excluded.path,
			content_hash = excluded.content_hash,
			body_hash = excluded.body_hash,
			indexed_at = CURRENT_TIMESTAMP,
			node_embedding_id = excluded.node_embedding_id`,
		nodeID, path, contentHash, bodyHash, nodeEmbeddingID,
	)
	if err != nil {
		return errors.Errorf("upsert node: %w", err)
	}

	return nil
}

func (s *Store) UpsertNodeSourceURL(ctx context.Context, nodeID, sourceURL string) error {
	sourceURL = strings.TrimSpace(sourceURL)
	if sourceURL == "" {
		return nil
	}
	err := s.execContext(ctx, `
		INSERT INTO node_source_urls (node_id, source_url) VALUES (?, ?)
		ON CONFLICT(node_id) DO UPDATE SET source_url = excluded.source_url`,
		nodeID, sourceURL,
	)
	if err != nil {
		return errors.Errorf("upsert node source url: %w", err)
	}

	return nil
}

func (s *Store) GetNodeByID(ctx context.Context, nodeID string) (*index.IndexedNode, error) {
	var node index.IndexedNode
	err := s.QueryRowContext(ctx, `
		SELECT node_id, path, content_hash, body_hash, indexed_at, node_embedding_id
		FROM indexed_nodes WHERE node_id = ?`, nodeID,
	).Scan(&node.NodeID, &node.Path, &node.ContentHash, &node.BodyHash, &node.IndexedAt, &node.NodeEmbeddingID)
	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (s *Store) UpdateNodePath(ctx context.Context, nodeID, newPath string) error {
	err := s.execContext(ctx, `UPDATE indexed_nodes SET path = ? WHERE node_id = ?`, newPath, nodeID)
	if err != nil {
		return errors.Errorf("update node path: %w", err)
	}
	if err := s.execContext(ctx, `UPDATE node_search SET path = ? WHERE node_id = ?`, newPath, nodeID); err != nil {
		return errors.Errorf("update node search path: %w", err)
	}
	if err := s.execContext(ctx, `UPDATE chunks SET node_path = ? WHERE node_id = ?`, newPath, nodeID); err != nil {
		return errors.Errorf("update chunks path: %w", err)
	}
	if err := s.execContext(ctx, `UPDATE chunk_search SET node_path = ? WHERE node_id = ?`, newPath, nodeID); err != nil {
		return errors.Errorf("update chunk search path: %w", err)
	}
	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		// FTS rows are keyed by path/node_path; rebuild on next index touch is acceptable after move.
		_ = s.execContext(ctx, `DELETE FROM node_search_fts WHERE path IN (SELECT path FROM node_search WHERE node_id = ?)`, nodeID)
		_ = s.execContext(ctx, `DELETE FROM chunk_search_fts WHERE node_path = ?`, newPath)
	}

	return nil
}

func (s *Store) FindBySourceURL(ctx context.Context, normalizedURL string) (*index.NodeSourceMatch, error) {
	normalizedURL = strings.TrimSpace(strings.ToLower(normalizedURL))
	if normalizedURL == "" {
		return nil, sql.ErrNoRows
	}
	var match index.NodeSourceMatch
	err := s.QueryRowContext(ctx, `
		SELECT u.node_id, n.path
		FROM node_source_urls u
		JOIN indexed_nodes n ON n.node_id = u.node_id
		WHERE u.source_url = ?`, normalizedURL,
	).Scan(&match.NodeID, &match.Path)
	if err != nil {
		return nil, err
	}

	return &match, nil
}

func (s *Store) DeleteNodeByID(ctx context.Context, nodeID string) error {
	node, err := s.GetNodeByID(ctx, nodeID)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return errors.Errorf("delete node by id: %w", err)
	}

	return s.deleteNodeRecord(ctx, node.NodeID, node.Path)
}

func (s *Store) UpsertNodeSearch(ctx context.Context, doc index.NodeSearchDocument) error {
	aliases := strings.Join(doc.Aliases, " ")
	keywords := strings.Join(doc.Keywords, " ")
	searchableText := index.JoinSearchText(
		doc.Path, doc.Title, doc.Type, aliases, doc.Annotation, keywords, doc.SourceURL, doc.SourceKind, doc.ContentProfile, doc.Body,
	)
	manualProcessed := 0
	if doc.ManualProcessed {
		manualProcessed = 1
	}

	err := s.execContext(ctx, `
		INSERT INTO node_search (
			node_id, path, title, type, aliases, annotation, keywords, source_url, manual_processed, body, searchable_text
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			path = excluded.path,
			title = excluded.title,
			type = excluded.type,
			aliases = excluded.aliases,
			annotation = excluded.annotation,
			keywords = excluded.keywords,
			source_url = excluded.source_url,
			manual_processed = excluded.manual_processed,
			body = excluded.body,
			searchable_text = excluded.searchable_text`,
		doc.NodeID, doc.Path, doc.Title, doc.Type, aliases, doc.Annotation, keywords, doc.SourceURL, manualProcessed, doc.Body, searchableText,
	)
	if err != nil {
		return errors.Errorf("upsert node search: %w", err)
	}

	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		if err := s.execContext(ctx, `DELETE FROM node_search_fts WHERE path = ?`, doc.Path); err != nil {
			return errors.Errorf("delete node search fts: %w", err)
		}
		err := s.execContext(ctx, `
			INSERT INTO node_search_fts (
				path, title, type, aliases, annotation, keywords, source_url, body, searchable_text
			) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			doc.Path, doc.Title, doc.Type, aliases, doc.Annotation, keywords, doc.SourceURL, doc.Body, searchableText,
		)
		if err != nil {
			return errors.Errorf("upsert node search fts: %w", err)
		}
	}

	return nil
}

func (s *Store) DeleteNode(ctx context.Context, path string) error {
	node, err := s.GetNodeByPath(ctx, path)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			return nil
		}

		return errors.Errorf("delete node: %w", err)
	}

	return s.deleteNodeRecord(ctx, node.NodeID, node.Path)
}

//nolint:funcorder // internal delete helper grouped with DeleteNode
func (s *Store) deleteNodeRecord(ctx context.Context, nodeID, path string) error {
	if err := s.DeleteChunks(ctx, nodeID, path); err != nil {
		return err
	}
	if err := s.execContext(ctx, `DELETE FROM node_source_urls WHERE node_id = ?`, nodeID); err != nil {
		return errors.Errorf("delete node source url: %w", err)
	}
	if err := s.execContext(ctx, `DELETE FROM node_search WHERE node_id = ?`, nodeID); err != nil {
		return errors.Errorf("delete node search: %w", err)
	}
	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		if err := s.execContext(ctx, `DELETE FROM node_search_fts WHERE path = ?`, path); err != nil {
			return errors.Errorf("delete node search fts: %w", err)
		}
	}

	err := s.execContext(ctx, `DELETE FROM indexed_nodes WHERE node_id = ?`, nodeID)
	if err != nil {
		return errors.Errorf("delete node: %w", err)
	}

	return nil
}

func (s *Store) GetNodeByPath(ctx context.Context, path string) (*index.IndexedNode, error) {
	var node index.IndexedNode
	err := s.QueryRowContext(ctx, `
		SELECT node_id, path, content_hash, body_hash, indexed_at, node_embedding_id
		FROM indexed_nodes WHERE path = ?`, path,
	).Scan(&node.NodeID, &node.Path, &node.ContentHash, &node.BodyHash, &node.IndexedAt, &node.NodeEmbeddingID)
	if err != nil {
		return nil, err
	}

	return &node, nil
}

func (s *Store) ListAllIndexed(ctx context.Context) ([]index.IndexedNode, error) {
	rows, err := s.QueryContext(ctx, `
		SELECT node_id, path, content_hash, body_hash, indexed_at, node_embedding_id
		FROM indexed_nodes`)
	if err != nil {
		return nil, errors.Errorf("list indexed nodes: %w", err)
	}
	defer rows.Close()

	var nodes []index.IndexedNode
	for rows.Next() {
		var node index.IndexedNode
		if err := rows.Scan(&node.NodeID, &node.Path, &node.ContentHash, &node.BodyHash, &node.IndexedAt, &node.NodeEmbeddingID); err != nil {
			return nil, errors.Errorf("scan indexed node: %w", err)
		}
		nodes = append(nodes, node)
	}

	return nodes, rows.Err()
}

func (s *Store) UpsertChunks(ctx context.Context, nodeID, nodePath string, chunks []index.Chunk) error {
	if err := s.DeleteChunks(ctx, nodeID, nodePath); err != nil {
		return err
	}

	for _, c := range chunks {
		err := s.execContext(ctx, `
			INSERT INTO chunks (node_id, node_path, chunk_index, heading, content, embedding_id)
			VALUES (?, ?, ?, ?, ?, ?)`,
			nodeID, nodePath, c.ChunkIndex, c.Heading, c.Content, c.EmbeddingID,
		)
		if err != nil {
			return errors.Errorf("insert chunk: %w", err)
		}

		if err := s.UpsertChunkSearch(ctx, index.ChunkSearchDocument{
			NodeID:     nodeID,
			NodePath:   nodePath,
			ChunkIndex: c.ChunkIndex,
			Heading:    c.Heading,
			Content:    c.Content,
		}); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) UpsertChunkSearch(ctx context.Context, doc index.ChunkSearchDocument) error {
	searchableText := index.JoinSearchText(doc.NodePath, doc.Heading, doc.Content)
	err := s.execContext(ctx, `
		INSERT INTO chunk_search (node_id, node_path, chunk_index, heading, content, searchable_text)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id, chunk_index) DO UPDATE SET
			node_path = excluded.node_path,
			heading = excluded.heading,
			content = excluded.content,
			searchable_text = excluded.searchable_text`,
		doc.NodeID, doc.NodePath, doc.ChunkIndex, doc.Heading, doc.Content, searchableText,
	)
	if err != nil {
		return errors.Errorf("upsert chunk search: %w", err)
	}

	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		if err := s.execContext(ctx, `DELETE FROM chunk_search_fts WHERE node_path = ? AND chunk_index = ?`, doc.NodePath, doc.ChunkIndex); err != nil {
			return errors.Errorf("delete chunk search fts: %w", err)
		}
		err := s.execContext(ctx, `
			INSERT INTO chunk_search_fts (node_path, chunk_index, heading, content, searchable_text)
			VALUES (?, ?, ?, ?, ?)`,
			doc.NodePath, doc.ChunkIndex, doc.Heading, doc.Content, searchableText,
		)
		if err != nil {
			return errors.Errorf("upsert chunk search fts: %w", err)
		}
	}

	return nil
}

func (s *Store) DeleteChunks(ctx context.Context, nodeID, nodePath string) error {
	if err := s.execContext(ctx, `DELETE FROM chunk_search WHERE node_id = ?`, nodeID); err != nil {
		return errors.Errorf("delete chunk search: %w", err)
	}
	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		if err := s.execContext(ctx, `DELETE FROM chunk_search_fts WHERE node_path = ?`, nodePath); err != nil {
			return errors.Errorf("delete chunk search fts: %w", err)
		}
	}

	err := s.execContext(ctx, `DELETE FROM chunks WHERE node_id = ?`, nodeID)
	if err != nil {
		return errors.Errorf("delete chunks: %w", err)
	}

	return nil
}

func (s *Store) ListChunksByNode(ctx context.Context, nodePath string) ([]index.Chunk, error) {
	rows, err := s.QueryContext(ctx, `
		SELECT id, node_id, node_path, chunk_index, heading, content, embedding_id
		FROM chunks WHERE node_path = ? ORDER BY chunk_index`, nodePath,
	)
	if err != nil {
		return nil, errors.Errorf("list chunks: %w", err)
	}
	defer rows.Close()

	var chunks []index.Chunk
	for rows.Next() {
		var c index.Chunk
		if err := rows.Scan(&c.ID, &c.NodeID, &c.NodePath, &c.ChunkIndex, &c.Heading, &c.Content, &c.EmbeddingID); err != nil {
			return nil, errors.Errorf("scan chunk: %w", err)
		}
		chunks = append(chunks, c)
	}

	return chunks, rows.Err()
}

func (s *Store) GetAllChunkEmbeddings(ctx context.Context) ([]index.ChunkEmbedding, error) {
	rows, err := s.QueryContext(ctx, `
		SELECT c.id, c.node_path, c.chunk_index, c.heading, c.content, e.vector
		FROM chunks c
		JOIN embeddings e ON c.embedding_id = e.id`)
	if err != nil {
		return nil, errors.Errorf("get chunk embeddings: %w", err)
	}
	defer rows.Close()

	var results []index.ChunkEmbedding
	for rows.Next() {
		var ce index.ChunkEmbedding
		var blob []byte
		if err := rows.Scan(&ce.ID, &ce.NodePath, &ce.ChunkIndex, &ce.Heading, &ce.Content, &blob); err != nil {
			return nil, errors.Errorf("scan chunk embedding: %w", err)
		}
		ce.Vector = decodeVector(blob)
		results = append(results, ce)
	}

	return results, rows.Err()
}

func (s *Store) GetAllNodeEmbeddings(ctx context.Context) ([]index.NodeEmbedding, error) {
	rows, err := s.QueryContext(ctx, `
		SELECT n.node_id, n.path, e.vector
		FROM indexed_nodes n
		JOIN embeddings e ON n.node_embedding_id = e.id`)
	if err != nil {
		return nil, errors.Errorf("get node embeddings: %w", err)
	}
	defer rows.Close()

	var results []index.NodeEmbedding
	for rows.Next() {
		var ne index.NodeEmbedding
		var blob []byte
		if err := rows.Scan(&ne.NodeID, &ne.Path, &blob); err != nil {
			return nil, errors.Errorf("scan node embedding: %w", err)
		}
		ne.Vector = decodeVector(blob)
		results = append(results, ne)
	}

	return results, rows.Err()
}

func (s *Store) GetStatus(ctx context.Context, model string) (*index.IndexStatus, error) {
	status := &index.IndexStatus{EmbeddingModel: model, KeywordIndex: s.KeywordIndexMode(), Status: "ready"}

	err := s.QueryRowContext(ctx, `SELECT COUNT(*) FROM indexed_nodes`).Scan(&status.TotalNodes)
	if err != nil {
		return nil, errors.Errorf("count nodes: %w", err)
	}

	err = s.QueryRowContext(ctx, `SELECT COUNT(*) FROM chunks`).Scan(&status.TotalChunks)
	if err != nil {
		return nil, errors.Errorf("count chunks: %w", err)
	}

	var lastIndexed sql.NullString
	err = s.QueryRowContext(ctx, `SELECT MAX(indexed_at) FROM indexed_nodes`).Scan(&lastIndexed)
	if err != nil {
		return nil, errors.Errorf("last indexed: %w", err)
	}
	if lastIndexed.Valid {
		status.LastIndexedAt, _ = time.Parse("2006-01-02 15:04:05", lastIndexed.String)
	}

	return status, nil
}

func (s *Store) ClearAll(ctx context.Context) error {
	if err := s.execContext(ctx, `DELETE FROM chunk_search`); err != nil {
		return errors.Errorf("clear chunk search: %w", err)
	}
	if err := s.execContext(ctx, `DELETE FROM node_search`); err != nil {
		return errors.Errorf("clear node search: %w", err)
	}
	if s.KeywordIndexMode() == index.KeywordIndexModeFTS5 {
		if err := s.execContext(ctx, `DELETE FROM chunk_search_fts`); err != nil {
			return errors.Errorf("clear chunk search fts: %w", err)
		}
		if err := s.execContext(ctx, `DELETE FROM node_search_fts`); err != nil {
			return errors.Errorf("clear node search fts: %w", err)
		}
	}
	if err := s.execContext(ctx, `DELETE FROM chunks`); err != nil {
		return errors.Errorf("clear chunks: %w", err)
	}
	if err := s.execContext(ctx, `DELETE FROM node_source_urls`); err != nil {
		return errors.Errorf("clear node source urls: %w", err)
	}
	if err := s.execContext(ctx, `DELETE FROM indexed_nodes`); err != nil {
		return errors.Errorf("clear nodes: %w", err)
	}
	if err := s.execContext(ctx, `DELETE FROM embeddings`); err != nil {
		return errors.Errorf("clear embeddings: %w", err)
	}

	return nil
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) execContext(ctx context.Context, query string, args ...any) error {
	_, err := s.db.ExecContext(ctx, query, args...)

	return err
}

func (s *Store) migrateFTS(ctx context.Context) error {
	if !s.detectFTS5(ctx) {
		s.keywordIndexMode = index.KeywordIndexModeScan

		return nil
	}

	schema := `
	CREATE VIRTUAL TABLE IF NOT EXISTS node_search_fts USING fts5(
		path UNINDEXED,
		title,
		type,
		aliases,
		annotation,
		keywords,
		source_url,
		body,
		searchable_text
	);
	CREATE VIRTUAL TABLE IF NOT EXISTS chunk_search_fts USING fts5(
		node_path UNINDEXED,
		chunk_index UNINDEXED,
		heading,
		content,
		searchable_text
	);`
	if _, err := s.db.ExecContext(ctx, schema); err != nil {
		return errors.Errorf("create fts schema: %w", err)
	}

	s.keywordIndexMode = index.KeywordIndexModeFTS5

	return nil
}

func (s *Store) detectFTS5(ctx context.Context) bool {
	_, err := s.db.ExecContext(ctx, `
		CREATE VIRTUAL TABLE IF NOT EXISTS __kb_fts5_probe USING fts5(value);
		DROP TABLE IF EXISTS __kb_fts5_probe;`)

	return err == nil
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
