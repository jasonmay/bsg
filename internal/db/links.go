package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/jasonmay/bsg/internal/model"
)

type CreateLinkInput struct {
	SpecID    string
	FilePath  string
	Symbol    string
	LinkType  model.LinkType
	StartLine *int
	StartCol  *int
	EndLine   *int
	EndCol    *int
}

func CreateLink(db *sql.DB, in CreateLinkInput) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT INTO code_links (spec_id, file_path, symbol, link_type, start_line, start_col, end_line, end_col, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.SpecID, in.FilePath, in.Symbol, string(in.LinkType), in.StartLine, in.StartCol, in.EndLine, in.EndCol, now,
	)
	if err != nil {
		return fmt.Errorf("insert code link: %w", err)
	}
	return nil
}

func DeleteLink(db *sql.DB, specID, filePath string) error {
	res, err := db.Exec(
		`DELETE FROM code_links WHERE spec_id = ? AND file_path = ?`,
		specID, filePath,
	)
	if err != nil {
		return fmt.Errorf("delete code link: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("no link found for %s -> %s", specID, filePath)
	}
	return nil
}

func GetLinksBySpec(db *sql.DB, specID string) ([]model.CodeLink, error) {
	rows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, start_line, start_col, end_line, end_col, created_at FROM code_links WHERE spec_id = ? ORDER BY file_path`,
		specID,
	)
	if err != nil {
		return nil, fmt.Errorf("query links: %w", err)
	}
	defer rows.Close()
	return scanLinks(rows)
}

func GetLinksByFile(db *sql.DB, filePath string) ([]model.CodeLink, error) {
	rows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, start_line, start_col, end_line, end_col, created_at FROM code_links WHERE file_path = ? ORDER BY spec_id`,
		filePath,
	)
	if err != nil {
		return nil, fmt.Errorf("query links by file: %w", err)
	}
	defer rows.Close()
	return scanLinks(rows)
}

func GetLinksByFileAndPosition(db *sql.DB, filePath string, line, col int) ([]model.CodeLink, error) {
	rows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, start_line, start_col, end_line, end_col, created_at
		FROM code_links
		WHERE file_path = ?
		  AND (start_line IS NULL OR (start_line <= ? AND (end_line IS NULL OR end_line >= ?)))
		ORDER BY spec_id`,
		filePath, line, line,
	)
	if err != nil {
		return nil, fmt.Errorf("query links by position: %w", err)
	}
	defer rows.Close()
	links, err := scanLinks(rows)
	if err != nil {
		return nil, err
	}
	var matched []model.CodeLink
	for _, l := range links {
		if l.ContainsPosition(line, col) {
			matched = append(matched, l)
		}
	}
	return matched, nil
}

func scanLinks(rows *sql.Rows) ([]model.CodeLink, error) {
	var links []model.CodeLink
	for rows.Next() {
		var l model.CodeLink
		var createdAt string
		var symbol sql.NullString
		var startLine, startCol, endLine, endCol sql.NullInt64
		if err := rows.Scan(&l.SpecID, &l.FilePath, &symbol, (*string)(&l.LinkType), &startLine, &startCol, &endLine, &endCol, &createdAt); err != nil {
			return nil, fmt.Errorf("scan link: %w", err)
		}
		l.Symbol = symbol.String
		if startLine.Valid {
			v := int(startLine.Int64)
			l.StartLine = &v
		}
		if startCol.Valid {
			v := int(startCol.Int64)
			l.StartCol = &v
		}
		if endLine.Valid {
			v := int(endLine.Int64)
			l.EndLine = &v
		}
		if endCol.Valid {
			v := int(endCol.Int64)
			l.EndCol = &v
		}
		l.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		links = append(links, l)
	}
	return links, rows.Err()
}
