package db

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/jasonmay/bsg/internal/model"
	"github.com/jasonmay/bsg/internal/specfile"
)

type CreateLinkInput struct {
	SpecID    string
	FilePath  string
	Symbol    string
	LinkType  model.LinkType
	Scope     model.LinkScope
	StartLine *int
	StartCol  *int
	EndLine   *int
	EndCol    *int
}

func CreateLink(db *sql.DB, bsgDir string, in CreateLinkInput) error {
	now := time.Now().UTC().Format(time.RFC3339)
	scope := in.Scope
	if scope == "" {
		scope = model.ScopeFile
	}
	_, err := db.Exec(
		`INSERT INTO code_links (spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.SpecID, in.FilePath, in.Symbol, string(in.LinkType), string(scope), in.StartLine, in.StartCol, in.EndLine, in.EndCol, now,
	)
	if err != nil {
		return fmt.Errorf("insert code link: %w", err)
	}
	return exportSpecFile(db, bsgDir, in.SpecID)
}

func DeleteLink(db *sql.DB, bsgDir string, specID, filePath string) error {
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
	return exportSpecFile(db, bsgDir, specID)
}

func exportSpecFile(db *sql.DB, bsgDir string, specID string) error {
	spec, err := GetSpec(db, specID)
	if err != nil {
		return fmt.Errorf("get spec for export: %w", err)
	}
	links, err := GetLinksBySpec(db, specID)
	if err != nil {
		return fmt.Errorf("get links for export: %w", err)
	}
	edges, err := GetEdgesBySpec(db, specID)
	if err != nil {
		return fmt.Errorf("get edges for export: %w", err)
	}
	return specfile.WriteSpec(bsgDir, spec, links, edges)
}

func GetLinksBySpec(db *sql.DB, specID string) ([]model.CodeLink, error) {
	rows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at FROM code_links WHERE spec_id = ? ORDER BY file_path`,
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
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at FROM code_links WHERE file_path = ? ORDER BY spec_id`,
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
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
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

type ScopedResult struct {
	Scope model.LinkScope
	Spec  model.Spec
	Link  model.CodeLink
}

func GetSpecsForLocation(db *sql.DB, filePath string, line, col int) ([]ScopedResult, error) {
	var results []ScopedResult

	// 1. Codebase-scoped links
	codebaseRows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
		FROM code_links WHERE scope = 'codebase' ORDER BY spec_id`)
	if err != nil {
		return nil, fmt.Errorf("query codebase links: %w", err)
	}
	codebaseLinks, err := scanLinks(codebaseRows)
	codebaseRows.Close()
	if err != nil {
		return nil, err
	}
	for _, l := range codebaseLinks {
		spec, err := GetSpec(db, l.SpecID)
		if err != nil {
			continue
		}
		results = append(results, ScopedResult{Scope: model.ScopeCodebase, Spec: *spec, Link: l})
	}

	// 2. Directory-scoped links (filePath starts with link's file_path)
	dirRows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
		FROM code_links WHERE scope = 'directory' ORDER BY spec_id`)
	if err != nil {
		return nil, fmt.Errorf("query directory links: %w", err)
	}
	dirLinks, err := scanLinks(dirRows)
	dirRows.Close()
	if err != nil {
		return nil, err
	}
	for _, l := range dirLinks {
		if strings.HasPrefix(filePath, l.FilePath) {
			spec, err := GetSpec(db, l.SpecID)
			if err != nil {
				continue
			}
			results = append(results, ScopedResult{Scope: model.ScopeDirectory, Spec: *spec, Link: l})
		}
	}

	// 3. File-scoped links (exact match)
	fileRows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
		FROM code_links WHERE scope = 'file' AND file_path = ? ORDER BY spec_id`, filePath)
	if err != nil {
		return nil, fmt.Errorf("query file links: %w", err)
	}
	fileLinks, err := scanLinks(fileRows)
	fileRows.Close()
	if err != nil {
		return nil, err
	}
	for _, l := range fileLinks {
		spec, err := GetSpec(db, l.SpecID)
		if err != nil {
			continue
		}
		results = append(results, ScopedResult{Scope: model.ScopeFile, Spec: *spec, Link: l})
	}

	// 4. Range-scoped links (file match + position in range)
	if line > 0 {
		rangeRows, err := db.Query(
			`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
			FROM code_links WHERE scope = 'range' AND file_path = ?
			  AND start_line <= ? AND (end_line IS NULL OR end_line >= ?)
			ORDER BY spec_id`, filePath, line, line)
		if err != nil {
			return nil, fmt.Errorf("query range links: %w", err)
		}
		rangeLinks, err := scanLinks(rangeRows)
		rangeRows.Close()
		if err != nil {
			return nil, err
		}
		for _, l := range rangeLinks {
			if l.ContainsPosition(line, col) {
				spec, err := GetSpec(db, l.SpecID)
				if err != nil {
					continue
				}
				results = append(results, ScopedResult{Scope: model.ScopeRange, Spec: *spec, Link: l})
			}
		}
	}

	// 5. Symbol-scoped links (exact file match)
	symRows, err := db.Query(
		`SELECT spec_id, file_path, symbol, link_type, scope, start_line, start_col, end_line, end_col, created_at
		FROM code_links WHERE scope = 'symbol' AND file_path = ? ORDER BY spec_id`, filePath)
	if err != nil {
		return nil, fmt.Errorf("query symbol links: %w", err)
	}
	symLinks, err := scanLinks(symRows)
	symRows.Close()
	if err != nil {
		return nil, err
	}
	for _, l := range symLinks {
		spec, err := GetSpec(db, l.SpecID)
		if err != nil {
			continue
		}
		results = append(results, ScopedResult{Scope: model.ScopeSymbol, Spec: *spec, Link: l})
	}

	return results, nil
}

func scanLinks(rows *sql.Rows) ([]model.CodeLink, error) {
	var links []model.CodeLink
	for rows.Next() {
		var l model.CodeLink
		var createdAt string
		var symbol sql.NullString
		var startLine, startCol, endLine, endCol sql.NullInt64
		if err := rows.Scan(&l.SpecID, &l.FilePath, &symbol, (*string)(&l.LinkType), (*string)(&l.Scope), &startLine, &startCol, &endLine, &endCol, &createdAt); err != nil {
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
