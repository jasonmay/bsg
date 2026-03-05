package db

import (
	"database/sql"
	"fmt"
)

var migrations = []func(*sql.Tx) error{
	migrateV1RangeColumns,
	migrateV2RangePK,
}

func Migrate(db *sql.DB) error {
	// check if this is a fresh DB (no tables yet) — skip migrations, schema.go handles it
	var tableCount int
	if err := db.QueryRow(`SELECT count(*) FROM sqlite_master WHERE type='table' AND name='code_links'`).Scan(&tableCount); err != nil {
		return fmt.Errorf("check tables: %w", err)
	}
	if tableCount == 0 {
		return nil
	}

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var version int
	err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&version)
	if err == sql.ErrNoRows {
		// fresh DB or pre-migration DB — start at 0
		if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (0)`); err != nil {
			return fmt.Errorf("insert initial version: %w", err)
		}
		version = 0
	} else if err != nil {
		return fmt.Errorf("read schema version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration %d: %w", i+1, err)
		}
		if err := migrations[i](tx); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, i+1); err != nil {
			tx.Rollback()
			return fmt.Errorf("update version to %d: %w", i+1, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %d: %w", i+1, err)
		}
	}
	return nil
}

func LatestSchemaVersion() int {
	return len(migrations)
}

func SetSchemaVersion(db *sql.DB, version int) error {
	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}
	var count int
	if err := db.QueryRow(`SELECT count(*) FROM schema_version`).Scan(&count); err != nil {
		return fmt.Errorf("count schema_version rows: %w", err)
	}
	if count == 0 {
		_, err := db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version)
		return err
	}
	_, err := db.Exec(`UPDATE schema_version SET version = ?`, version)
	return err
}

func migrateV2RangePK(tx *sql.Tx) error {
	stmts := []string{
		`CREATE TABLE code_links_new (
			spec_id    TEXT NOT NULL REFERENCES specs(id),
			file_path  TEXT NOT NULL,
			symbol     TEXT,
			link_type  TEXT NOT NULL,
			start_line INTEGER,
			start_col  INTEGER,
			end_line   INTEGER,
			end_col    INTEGER,
			created_at TEXT NOT NULL
		)`,
		`INSERT INTO code_links_new SELECT spec_id, file_path, symbol, link_type, start_line, start_col, end_line, end_col, created_at FROM code_links`,
		`DROP TABLE code_links`,
		`ALTER TABLE code_links_new RENAME TO code_links`,
		`CREATE UNIQUE INDEX idx_code_links_unique ON code_links (spec_id, file_path, link_type, COALESCE(start_line, 0))`,
	}
	for _, q := range stmts {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("%s: %w", q[:40], err)
		}
	}
	return nil
}

func migrateV1RangeColumns(tx *sql.Tx) error {
	alters := []string{
		`ALTER TABLE code_links ADD COLUMN start_line INTEGER`,
		`ALTER TABLE code_links ADD COLUMN start_col INTEGER`,
		`ALTER TABLE code_links ADD COLUMN end_line INTEGER`,
		`ALTER TABLE code_links ADD COLUMN end_col INTEGER`,
	}
	for _, q := range alters {
		if _, err := tx.Exec(q); err != nil {
			return fmt.Errorf("%s: %w", q, err)
		}
	}
	return nil
}
