package sqlite

import (
	"context"
	"database/sql"

	"github.com/muonsoft/errors"
)

const schemaVersionV2 = 2

func (s *Store) migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, "PRAGMA journal_mode=WAL"); err != nil {
		return errors.Errorf("set WAL mode: %w", err)
	}

	if err := s.ensureSchemaMeta(ctx); err != nil {
		return err
	}
	version, err := s.currentSchemaVersion(ctx)
	if err != nil {
		return err
	}
	if version < schemaVersionV2 {
		if err := s.migrateToV2(ctx); err != nil {
			return err
		}
	}
	if err := s.createV2Schema(ctx); err != nil {
		return err
	}
	if err := s.setSchemaVersion(ctx, schemaVersionV2); err != nil {
		return err
	}

	return s.migrateFTS(ctx)
}

func (s *Store) ensureSchemaMeta(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_meta (
			key TEXT PRIMARY KEY,
			value INTEGER NOT NULL
		)`)

	return err
}

func (s *Store) currentSchemaVersion(ctx context.Context) (int, error) {
	var version sql.NullInt64
	err := s.QueryRowContext(ctx, `SELECT value FROM schema_meta WHERE key = 'version'`).Scan(&version)
	if errors.Is(err, sql.ErrNoRows) {
		if s.hasLegacyPathPKSchema(ctx) {
			return 1, nil
		}

		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}

	return int(version.Int64), nil
}

func (s *Store) setSchemaVersion(ctx context.Context, version int) error {
	return s.execContext(ctx, `
		INSERT INTO schema_meta (key, value) VALUES ('version', ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, version)
}

func (s *Store) hasLegacyPathPKSchema(ctx context.Context) bool {
	rows, err := s.db.QueryContext(ctx, `PRAGMA table_info(indexed_nodes)`)
	if err != nil {
		return false
	}
	defer rows.Close()
	hasPathPK := false
	hasNodeID := false
	for rows.Next() {
		var cid int
		var name, colType string
		var notNull, pk int
		var dfltValue sql.NullString
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			return false
		}
		switch name {
		case "path":
			if pk == 1 {
				hasPathPK = true
			}
		case "node_id":
			hasNodeID = true
		}
	}
	if err := rows.Err(); err != nil {
		return false
	}

	return hasPathPK && !hasNodeID
}

func (s *Store) migrateToV2(ctx context.Context) error {
	tables := []string{
		"chunk_search_fts", "node_search_fts",
		"chunk_search", "node_search", "node_source_urls",
		"chunks", "indexed_nodes",
	}
	for _, table := range tables {
		if err := s.execContext(ctx, `DROP TABLE IF EXISTS `+table); err != nil {
			return errors.Errorf("drop %s: %w", table, err)
		}
	}

	return nil
}

func (s *Store) createV2Schema(ctx context.Context) error {
	schema := `
	CREATE TABLE IF NOT EXISTS embeddings (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		vector BLOB NOT NULL,
		model TEXT NOT NULL,
		dimensions INTEGER NOT NULL
	);
	CREATE TABLE IF NOT EXISTS indexed_nodes (
		node_id TEXT PRIMARY KEY,
		path TEXT NOT NULL UNIQUE,
		content_hash TEXT NOT NULL,
		body_hash TEXT NOT NULL DEFAULT '',
		indexed_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
		node_embedding_id INTEGER NOT NULL,
		FOREIGN KEY (node_embedding_id) REFERENCES embeddings(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS chunks (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		node_id TEXT NOT NULL,
		node_path TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		heading TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL,
		embedding_id INTEGER NOT NULL,
		UNIQUE(node_id, chunk_index),
		FOREIGN KEY (node_id) REFERENCES indexed_nodes(node_id) ON DELETE CASCADE,
		FOREIGN KEY (embedding_id) REFERENCES embeddings(id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS node_search (
		node_id TEXT PRIMARY KEY,
		path TEXT NOT NULL,
		title TEXT NOT NULL DEFAULT '',
		type TEXT NOT NULL DEFAULT '',
		aliases TEXT NOT NULL DEFAULT '',
		annotation TEXT NOT NULL DEFAULT '',
		keywords TEXT NOT NULL DEFAULT '',
		source_url TEXT NOT NULL DEFAULT '',
		manual_processed INTEGER NOT NULL DEFAULT 0,
		body TEXT NOT NULL DEFAULT '',
		searchable_text TEXT NOT NULL DEFAULT '',
		FOREIGN KEY (node_id) REFERENCES indexed_nodes(node_id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS chunk_search (
		node_id TEXT NOT NULL,
		node_path TEXT NOT NULL,
		chunk_index INTEGER NOT NULL,
		heading TEXT NOT NULL DEFAULT '',
		content TEXT NOT NULL DEFAULT '',
		searchable_text TEXT NOT NULL DEFAULT '',
		PRIMARY KEY (node_id, chunk_index),
		FOREIGN KEY (node_id) REFERENCES indexed_nodes(node_id) ON DELETE CASCADE
	);
	CREATE TABLE IF NOT EXISTS node_source_urls (
		node_id TEXT NOT NULL,
		source_url TEXT NOT NULL,
		PRIMARY KEY (node_id),
		UNIQUE(source_url),
		FOREIGN KEY (node_id) REFERENCES indexed_nodes(node_id) ON DELETE CASCADE
	);`

	_, err := s.db.ExecContext(ctx, schema)

	return err
}
