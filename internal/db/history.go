package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jasonmay/bsg/internal/model"
)

func AppendHistory(tx *sql.Tx, specID, field, oldValue, newValue string) error {
	_, err := tx.Exec(
		`INSERT INTO history (spec_id, changed_at, field, old_value, new_value) VALUES (?, ?, ?, ?, ?)`,
		specID, time.Now().UTC().Format(time.RFC3339), field, oldValue, newValue,
	)
	if err != nil {
		return fmt.Errorf("append history: %w", err)
	}
	return nil
}

func GetHistory(db *sql.DB, specID string) ([]model.HistoryEntry, error) {
	rows, err := db.Query(
		`SELECT id, spec_id, changed_at, field, old_value, new_value FROM history WHERE spec_id = ? ORDER BY id`,
		specID,
	)
	if err != nil {
		return nil, fmt.Errorf("query history: %w", err)
	}
	defer rows.Close()

	var entries []model.HistoryEntry
	for rows.Next() {
		var e model.HistoryEntry
		var changedAt string
		var oldVal, newVal sql.NullString
		if err := rows.Scan(&e.ID, &e.SpecID, &changedAt, &e.Field, &oldVal, &newVal); err != nil {
			return nil, fmt.Errorf("scan history: %w", err)
		}
		e.ChangedAt, err = time.Parse(time.RFC3339, changedAt)
		if err != nil {
			return nil, fmt.Errorf("parse changed_at: %w", err)
		}
		e.OldValue = oldVal.String
		e.NewValue = newVal.String
		entries = append(entries, e)
	}
	return entries, rows.Err()
}
