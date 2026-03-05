package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/jasonmay/bsg/internal/model"
	"github.com/jasonmay/bsg/internal/specfile"
)

type edgeKey struct {
	fromID, toID string
	relation     string
}

func SyncFromFiles(db *sql.DB, bsgDir string) error {
	specs, err := specfile.ReadAll(bsgDir)
	if err != nil {
		return fmt.Errorf("read spec files: %w", err)
	}
	if len(specs) == 0 {
		return nil
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)

	fileIDs := make(map[string]bool)
	for _, sf := range specs {
		fileIDs[sf.ID] = true

		_, err := tx.Exec(
			`INSERT OR REPLACE INTO specs (id, name, type, status, body, tags, created_at, updated_at)
			 VALUES (?, ?, ?, ?, ?, ?, COALESCE((SELECT created_at FROM specs WHERE id = ?), ?), ?)`,
			sf.ID, sf.Name, sf.Type, sf.Status, sf.Body, marshalTags(sf.Tags), sf.ID, now, now,
		)
		if err != nil {
			return fmt.Errorf("upsert spec %s: %w", sf.ID, err)
		}

		if _, err := tx.Exec(`DELETE FROM code_links WHERE spec_id = ?`, sf.ID); err != nil {
			return fmt.Errorf("delete links for %s: %w", sf.ID, err)
		}

		for _, l := range sf.Links {
			linkType := l.Type
			if linkType == "" {
				linkType = string(model.LinkImplements)
			}
			scope := l.Scope
			if scope == "" {
				scope = string(model.ScopeFile)
			}
			_, err := tx.Exec(
				`INSERT INTO code_links (spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at)
				 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
				sf.ID, l.File, l.Symbol, linkType, scope, l.StartLine, l.StartCol, l.EndLine, l.EndCol, now,
			)
			if err != nil {
				return fmt.Errorf("insert link for %s: %w", sf.ID, err)
			}
		}
	}

	// Sync edges: collect all "out" edges from JSON, replace DB edges
	fileEdges := make(map[edgeKey]bool)
	for _, sf := range specs {
		for _, ef := range sf.Edges {
			if ef.Dir != "out" {
				continue
			}
			k := edgeKey{fromID: sf.ID, toID: ef.Spec, relation: ef.Relation}
			fileEdges[k] = true
		}
	}
	if _, err := tx.Exec(`DELETE FROM edges`); err != nil {
		return fmt.Errorf("clear edges: %w", err)
	}
	for k := range fileEdges {
		_, err := tx.Exec(
			`INSERT INTO edges (from_id, to_id, relation, created_at) VALUES (?, ?, ?, ?)`,
			k.fromID, k.toID, k.relation, now,
		)
		if err != nil {
			return fmt.Errorf("insert edge %s->%s: %w", k.fromID, k.toID, err)
		}
	}

	rows, err := tx.Query(`SELECT id FROM specs`)
	if err != nil {
		return fmt.Errorf("list db specs: %w", err)
	}
	var toDelete []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan spec id: %w", err)
		}
		if !fileIDs[id] {
			toDelete = append(toDelete, id)
		}
	}
	rows.Close()

	for _, id := range toDelete {
		if err := cascadeDeleteTx(tx, id); err != nil {
			return fmt.Errorf("cascade delete %s: %w", id, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit sync: %w", err)
	}

	return writeSyncMarker(bsgDir)
}

func NeedsSync(bsgDir string) bool {
	specsDir := filepath.Join(bsgDir, "specs")
	entries, err := os.ReadDir(specsDir)
	if err != nil {
		return false
	}

	var hasJSON bool
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".json" {
			hasJSON = true
			break
		}
	}
	if !hasJSON {
		return false
	}

	markerPath := filepath.Join(specsDir, ".synced")
	markerInfo, err := os.Stat(markerPath)
	if err != nil {
		return true
	}
	markerTime := markerInfo.ModTime()

	for _, e := range entries {
		if filepath.Ext(e.Name()) != ".json" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().After(markerTime) {
			return true
		}
	}
	return false
}

func writeSyncMarker(bsgDir string) error {
	markerPath := filepath.Join(bsgDir, "specs", ".synced")
	return os.WriteFile(markerPath, []byte(time.Now().UTC().Format(time.RFC3339)+"\n"), 0644)
}

func marshalTags(tags []string) *string {
	if len(tags) == 0 {
		return nil
	}
	b, _ := json.Marshal(tags)
	s := string(b)
	return &s
}
