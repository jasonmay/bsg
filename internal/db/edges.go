package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jasonmay/bsg/internal/model"
)

func CreateEdge(db *sql.DB, bsgDir string, fromID, toID string, relation model.Relation) error {
	if !IDExists(db, fromID) {
		return fmt.Errorf("spec %s not found", fromID)
	}
	if !IDExists(db, toID) {
		return fmt.Errorf("spec %s not found", toID)
	}

	if relation == model.RelDependsOn {
		if err := checkCycle(db, fromID, toID); err != nil {
			return err
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO edges (from_id, to_id, relation, created_at) VALUES (?, ?, ?, ?)`,
		fromID, toID, string(relation), now,
	)
	if err != nil {
		return fmt.Errorf("insert edge: %w", err)
	}

	if err := exportSpecFile(db, bsgDir, fromID); err != nil {
		return fmt.Errorf("export from spec: %w", err)
	}
	if err := exportSpecFile(db, bsgDir, toID); err != nil {
		return fmt.Errorf("export to spec: %w", err)
	}
	return nil
}

func DeleteEdge(db *sql.DB, bsgDir string, fromID, toID string, relation *model.Relation) error {
	var res sql.Result
	var err error
	if relation != nil {
		res, err = db.Exec(
			`DELETE FROM edges WHERE from_id = ? AND to_id = ? AND relation = ?`,
			fromID, toID, string(*relation),
		)
	} else {
		res, err = db.Exec(
			`DELETE FROM edges WHERE from_id = ? AND to_id = ?`,
			fromID, toID,
		)
	}
	if err != nil {
		return fmt.Errorf("delete edge: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("no edge found between %s and %s", fromID, toID)
	}

	if err := exportSpecFile(db, bsgDir, fromID); err != nil {
		return fmt.Errorf("export from spec: %w", err)
	}
	if err := exportSpecFile(db, bsgDir, toID); err != nil {
		return fmt.Errorf("export to spec: %w", err)
	}
	return nil
}

func GetEdgesBySpec(db *sql.DB, specID string) ([]model.Edge, error) {
	rows, err := db.Query(
		`SELECT from_id, to_id, relation, created_at FROM edges WHERE from_id = ? OR to_id = ? ORDER BY created_at`,
		specID, specID,
	)
	if err != nil {
		return nil, fmt.Errorf("query edges: %w", err)
	}
	defer rows.Close()

	var edges []model.Edge
	for rows.Next() {
		var e model.Edge
		var createdAt string
		if err := rows.Scan(&e.FromID, &e.ToID, (*string)(&e.Relation), &createdAt); err != nil {
			return nil, fmt.Errorf("scan edge: %w", err)
		}
		e.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		edges = append(edges, e)
	}
	return edges, rows.Err()
}

func checkCycle(db *sql.DB, fromID, toID string) error {
	visited := make(map[string]bool)
	stack := []string{toID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if current == fromID {
			return fmt.Errorf("cycle detected: %s depends_on %s creates a cycle", fromID, toID)
		}

		if visited[current] {
			continue
		}
		visited[current] = true

		rows, err := db.Query(
			`SELECT to_id FROM edges WHERE from_id = ? AND relation = ?`,
			current, string(model.RelDependsOn),
		)
		if err != nil {
			return fmt.Errorf("query edges for cycle check: %w", err)
		}
		for rows.Next() {
			var next string
			if err := rows.Scan(&next); err != nil {
				rows.Close()
				return fmt.Errorf("scan edge for cycle check: %w", err)
			}
			stack = append(stack, next)
		}
		rows.Close()
	}
	return nil
}
