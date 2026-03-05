package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}
	if err := Migrate(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	bsgDir := filepath.Dir(path)
	if NeedsSync(bsgDir) {
		if err := SyncFromFiles(db, bsgDir); err != nil {
			db.Close()
			return nil, fmt.Errorf("sync from files: %w", err)
		}
	}

	return db, nil
}

func Initialize(path string) error {
	db, err := Open(path)
	if err != nil {
		return err
	}
	defer db.Close()
	if _, err := db.Exec(Schema); err != nil {
		return fmt.Errorf("create schema: %w", err)
	}
	if err := SetSchemaVersion(db, LatestSchemaVersion()); err != nil {
		return fmt.Errorf("set schema version: %w", err)
	}
	return nil
}

func FindDB() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for {
		candidate := filepath.Join(dir, ".bsg", "bsg.db")
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		specsDir := filepath.Join(dir, ".bsg", "specs")
		if info, err := os.Stat(specsDir); err == nil && info.IsDir() {
			if err := Initialize(candidate); err != nil {
				return "", fmt.Errorf("initialize db from specs: %w", err)
			}
			return candidate, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf(".bsg/bsg.db not found")
		}
		dir = parent
	}
}
