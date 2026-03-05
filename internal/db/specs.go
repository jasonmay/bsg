package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/jasonmay/bsg/internal/model"
	"github.com/jasonmay/bsg/internal/specfile"
)

type CreateSpecInput struct {
	ID   string
	Name string
	Type model.SpecType
	Body string
	Tags []string
}

func CreateSpec(db *sql.DB, bsgDir string, in CreateSpecInput) error {
	now := time.Now().UTC().Format(time.RFC3339)
	var tagsJSON *string
	if len(in.Tags) > 0 {
		b, err := json.Marshal(in.Tags)
		if err != nil {
			return fmt.Errorf("marshal tags: %w", err)
		}
		s := string(b)
		tagsJSON = &s
	}
	_, err := db.Exec(
		`INSERT INTO specs (id, name, type, status, body, tags, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		in.ID, in.Name, string(in.Type), string(model.StatusDraft), in.Body, tagsJSON, now, now,
	)
	if err != nil {
		return fmt.Errorf("insert spec: %w", err)
	}

	spec := &model.Spec{
		ID:     in.ID,
		Name:   in.Name,
		Type:   in.Type,
		Status: model.StatusDraft,
		Body:   in.Body,
		Tags:   in.Tags,
	}
	if err := specfile.WriteSpec(bsgDir, spec, nil); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}
	return nil
}

func GetSpec(db *sql.DB, id string) (*model.Spec, error) {
	var s model.Spec
	var tagsJSON sql.NullString
	var createdAt, updatedAt string
	err := db.QueryRow(
		`SELECT id, name, type, status, body, tags, created_at, updated_at FROM specs WHERE id = ?`,
		id,
	).Scan(&s.ID, &s.Name, (*string)(&s.Type), (*string)(&s.Status), &s.Body, &tagsJSON, &createdAt, &updatedAt)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("spec %s not found", id)
	}
	if err != nil {
		return nil, fmt.Errorf("get spec: %w", err)
	}
	s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
	if tagsJSON.Valid {
		if err := json.Unmarshal([]byte(tagsJSON.String), &s.Tags); err != nil {
			return nil, fmt.Errorf("unmarshal tags: %w", err)
		}
	}
	return &s, nil
}

func IDExists(db *sql.DB, id string) bool {
	var count int
	db.QueryRow(`SELECT COUNT(*) FROM specs WHERE id = ?`, id).Scan(&count)
	return count > 0
}

type UpdateSpecInput struct {
	ID     string
	Name   *string
	Body   *string
	Tags   *[]string
	Status *model.SpecStatus
}

func UpdateSpec(db *sql.DB, bsgDir string, in UpdateSpecInput) error {
	existing, err := GetSpec(db, in.ID)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC().Format(time.RFC3339)
	sets := []string{"updated_at = ?"}
	args := []any{now}

	if in.Name != nil && *in.Name != existing.Name {
		if err := AppendHistory(tx, in.ID, "name", existing.Name, *in.Name); err != nil {
			return err
		}
		sets = append(sets, "name = ?")
		args = append(args, *in.Name)
	}

	if in.Body != nil && *in.Body != existing.Body {
		if err := AppendHistory(tx, in.ID, "body", existing.Body, *in.Body); err != nil {
			return err
		}
		sets = append(sets, "body = ?")
		args = append(args, *in.Body)
	}

	if in.Tags != nil {
		oldTags, _ := json.Marshal(existing.Tags)
		newTags, _ := json.Marshal(*in.Tags)
		if string(oldTags) != string(newTags) {
			if err := AppendHistory(tx, in.ID, "tags", string(oldTags), string(newTags)); err != nil {
				return err
			}
			sets = append(sets, "tags = ?")
			args = append(args, string(newTags))
		}
	}

	if in.Status != nil && *in.Status != existing.Status {
		if err := model.ValidateTransition(existing.Status, *in.Status); err != nil {
			return err
		}
		if err := AppendHistory(tx, in.ID, "status", string(existing.Status), string(*in.Status)); err != nil {
			return err
		}
		sets = append(sets, "status = ?")
		args = append(args, string(*in.Status))
	}

	args = append(args, in.ID)
	query := fmt.Sprintf("UPDATE specs SET %s WHERE id = ?", strings.Join(sets, ", "))
	if _, err := tx.Exec(query, args...); err != nil {
		return fmt.Errorf("update spec: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return err
	}

	spec, err := GetSpec(db, in.ID)
	if err != nil {
		return fmt.Errorf("re-read spec for export: %w", err)
	}
	links, err := GetLinksBySpec(db, in.ID)
	if err != nil {
		return fmt.Errorf("get links for export: %w", err)
	}
	if err := specfile.WriteSpec(bsgDir, spec, links); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}
	return nil
}

func DeleteSpec(db *sql.DB, bsgDir string, id string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	for _, table := range []string{"history", "code_links", "edges"} {
		var query string
		if table == "edges" {
			query = fmt.Sprintf("DELETE FROM %s WHERE from_id = ? OR to_id = ?", table)
			if _, err := tx.Exec(query, id, id); err != nil {
				return fmt.Errorf("delete from %s: %w", table, err)
			}
		} else {
			query = fmt.Sprintf("DELETE FROM %s WHERE spec_id = ?", table)
			if _, err := tx.Exec(query, id); err != nil {
				return fmt.Errorf("delete from %s: %w", table, err)
			}
		}
	}

	res, err := tx.Exec(`DELETE FROM specs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete spec: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("spec %s not found", id)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return specfile.DeleteSpec(bsgDir, id)
}

type ListSpecsInput struct {
	Status *model.SpecStatus
	Type   *model.SpecType
	Tag    *string
}

func ListSpecs(db *sql.DB, in ListSpecsInput) ([]model.Spec, error) {
	query := `SELECT id, name, type, status, body, tags, created_at, updated_at FROM specs WHERE 1=1`
	var args []any

	if in.Status != nil {
		query += ` AND status = ?`
		args = append(args, string(*in.Status))
	}
	if in.Type != nil {
		query += ` AND type = ?`
		args = append(args, string(*in.Type))
	}
	if in.Tag != nil {
		query += ` AND tags LIKE ?`
		args = append(args, `%"`+*in.Tag+`"%`)
	}

	query += ` ORDER BY created_at`

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list specs: %w", err)
	}
	defer rows.Close()

	var specs []model.Spec
	for rows.Next() {
		var s model.Spec
		var tagsJSON sql.NullString
		var createdAt, updatedAt string
		if err := rows.Scan(&s.ID, &s.Name, (*string)(&s.Type), (*string)(&s.Status), &s.Body, &tagsJSON, &createdAt, &updatedAt); err != nil {
			return nil, fmt.Errorf("scan spec: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)
		if tagsJSON.Valid {
			json.Unmarshal([]byte(tagsJSON.String), &s.Tags)
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}
