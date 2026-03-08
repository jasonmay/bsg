package db

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/jasonmay/bsg/internal/model"
)

type CoverageStats struct {
	Total         int
	WithLinks     int
	Verified      int
	Drifted       []DriftedSpec
	ReadyToImpl   []model.Spec
	ReadyToVerify []model.Spec
}

type DriftedSpec struct {
	Spec         model.Spec
	DriftedFiles []DriftedFile
}

type DriftedFile struct {
	FilePath   string
	ModifiedAt time.Time
	DriftDays  int
}

func GetCoverage(db *sql.DB) (*CoverageStats, error) {
	var stats CoverageStats

	if err := db.QueryRow(`SELECT COUNT(*) FROM specs`).Scan(&stats.Total); err != nil {
		return nil, fmt.Errorf("count specs: %w", err)
	}

	if err := db.QueryRow(`SELECT COUNT(DISTINCT spec_id) FROM code_links`).Scan(&stats.WithLinks); err != nil {
		return nil, fmt.Errorf("count linked: %w", err)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM specs WHERE status = 'verified'`).Scan(&stats.Verified); err != nil {
		return nil, fmt.Errorf("count verified: %w", err)
	}

	drifted, err := getDrifted(db)
	if err != nil {
		return nil, err
	}
	stats.Drifted = drifted

	ready, err := getReadyToImpl(db)
	if err != nil {
		return nil, err
	}
	stats.ReadyToImpl = ready

	readyVerify, err := getReadyToVerify(db)
	if err != nil {
		return nil, err
	}
	stats.ReadyToVerify = readyVerify

	return &stats, nil
}

func getDrifted(db *sql.DB) ([]DriftedSpec, error) {
	rows, err := db.Query(`
		SELECT s.id, s.name, s.type, s.status, s.body, s.tags, s.created_at, s.updated_at,
		       cl.file_path,
		       h.changed_at AS verify_time
		FROM specs s
		JOIN code_links cl ON cl.spec_id = s.id
		JOIN (
			SELECT spec_id, MAX(changed_at) AS changed_at
			FROM history
			WHERE field = 'status' AND new_value = 'verified'
			GROUP BY spec_id
		) h ON h.spec_id = s.id
		WHERE s.status = 'verified'
		ORDER BY s.id, cl.file_path
	`)
	if err != nil {
		return nil, fmt.Errorf("query drifted: %w", err)
	}
	defer rows.Close()

	specMap := make(map[string]*DriftedSpec)
	var order []string

	for rows.Next() {
		var s model.Spec
		var tagsJSON, filePath, verifyTimeStr sql.NullString
		var createdAt, updatedAt string

		if err := rows.Scan(
			&s.ID, &s.Name, (*string)(&s.Type), (*string)(&s.Status),
			&s.Body, &tagsJSON, &createdAt, &updatedAt,
			&filePath, &verifyTimeStr,
		); err != nil {
			return nil, fmt.Errorf("scan drifted: %w", err)
		}
		s.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		s.UpdatedAt, _ = time.Parse(time.RFC3339, updatedAt)

		verifyTime, _ := time.Parse(time.RFC3339, verifyTimeStr.String)

		info, err := os.Stat(filePath.String)
		if err != nil {
			continue
		}
		mtime := info.ModTime()
		if !mtime.After(verifyTime) {
			continue
		}

		ds, ok := specMap[s.ID]
		if !ok {
			ds = &DriftedSpec{Spec: s}
			specMap[s.ID] = ds
			order = append(order, s.ID)
		}
		ds.DriftedFiles = append(ds.DriftedFiles, DriftedFile{
			FilePath:   filePath.String,
			ModifiedAt: mtime,
			DriftDays:  int(mtime.Sub(verifyTime).Hours() / 24),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	var result []DriftedSpec
	for _, id := range order {
		result = append(result, *specMap[id])
	}
	return result, nil
}

func getReadyToImpl(db *sql.DB) ([]model.Spec, error) {
	rows, err := db.Query(`
		SELECT s.id, s.name, s.type, s.status, s.body, s.tags, s.created_at, s.updated_at
		FROM specs s
		WHERE s.status = 'accepted'
		  AND s.id NOT IN (SELECT DISTINCT spec_id FROM code_links)
		ORDER BY s.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("query ready: %w", err)
	}
	defer rows.Close()

	var specs []model.Spec
	for rows.Next() {
		s, err := scanSpec(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ready: %w", err)
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

func getReadyToVerify(db *sql.DB) ([]model.Spec, error) {
	rows, err := db.Query(`
		SELECT s.id, s.name, s.type, s.status, s.body, s.tags, s.created_at, s.updated_at
		FROM specs s
		WHERE s.status = 'implemented'
		  AND s.id IN (SELECT DISTINCT spec_id FROM code_links)
		ORDER BY s.created_at
	`)
	if err != nil {
		return nil, fmt.Errorf("query ready to verify: %w", err)
	}
	defer rows.Close()

	var specs []model.Spec
	for rows.Next() {
		s, err := scanSpec(rows)
		if err != nil {
			return nil, fmt.Errorf("scan ready to verify: %w", err)
		}
		specs = append(specs, s)
	}
	return specs, rows.Err()
}

func GetDriftedIDs(db *sql.DB) map[string]bool {
	coverage, err := GetCoverage(db)
	if err != nil {
		return nil
	}
	ids := make(map[string]bool, len(coverage.Drifted))
	for _, d := range coverage.Drifted {
		ids[d.Spec.ID] = true
	}
	return ids
}
