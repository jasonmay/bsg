package db

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jasonmay/bsg/internal/model"
)

// setupProject creates a temp dir with .bsg/, source files, and an initialized DB.
func setupProject(t *testing.T) (string, string) {
	t.Helper()
	root := t.TempDir()
	bsgDir := filepath.Join(root, ".bsg")
	os.MkdirAll(filepath.Join(bsgDir, "specs"), 0755)

	dbPath := filepath.Join(bsgDir, "bsg.db")
	if err := Initialize(dbPath); err != nil {
		t.Fatalf("initialize: %v", err)
	}

	// Create source files
	srcDir := filepath.Join(root, "cmd")
	os.MkdirAll(srcDir, 0755)
	writeFile(t, filepath.Join(srcDir, "main.go"), "package main\nfunc main() {}\n")
	writeFile(t, filepath.Join(srcDir, "server.go"), "package main\nfunc serve() {}\n")

	libDir := filepath.Join(root, "internal", "core")
	os.MkdirAll(libDir, 0755)
	writeFile(t, filepath.Join(libDir, "core.go"), "package core\nfunc Init() {}\n")

	return root, bsgDir
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func statusPtr(s model.SpecStatus) *model.SpecStatus { return &s }

func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

func TestGetSpecsForDirectory(t *testing.T) {
	t.Parallel()
	_, bsgDir := setupProject(t)

	db, err := Open(filepath.Join(bsgDir, "bsg.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	CreateSpec(db, bsgDir, CreateSpecInput{
		ID: "bsg-0001", Name: "main entry", Type: model.SpecTypeBehavior,
	})
	CreateSpec(db, bsgDir, CreateSpecInput{
		ID: "bsg-0002", Name: "core init", Type: model.SpecTypeBehavior,
	})

	CreateLink(db, bsgDir, CreateLinkInput{
		SpecID: "bsg-0001", FilePath: "cmd/main.go", LinkType: model.LinkImplements, Scope: model.ScopeFile,
	})
	CreateLink(db, bsgDir, CreateLinkInput{
		SpecID: "bsg-0002", FilePath: "internal/core/core.go", LinkType: model.LinkImplements, Scope: model.ScopeFile,
	})

	// All files
	results, err := GetSpecsForDirectory(db, "")
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("all: got %d results, want 2", len(results))
	}

	// Only cmd/
	results, err = GetSpecsForDirectory(db, "cmd")
	if err != nil {
		t.Fatalf("cmd: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("cmd: got %d results, want 1", len(results))
	}
	if results[0].Spec.ID != "bsg-0001" {
		t.Fatalf("cmd: got spec %s, want bsg-0001", results[0].Spec.ID)
	}

	// Only internal/
	results, err = GetSpecsForDirectory(db, "internal")
	if err != nil {
		t.Fatalf("internal: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("internal: got %d results, want 1", len(results))
	}
	if results[0].Spec.ID != "bsg-0002" {
		t.Fatalf("internal: got spec %s, want bsg-0002", results[0].Spec.ID)
	}

	// No match
	results, err = GetSpecsForDirectory(db, "nonexistent")
	if err != nil {
		t.Fatalf("nonexistent: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("nonexistent: got %d results, want 0", len(results))
	}
}

func TestDriftDetection(t *testing.T) {
	root, bsgDir := setupProject(t)
	chdir(t, root)

	db, err := Open(filepath.Join(bsgDir, "bsg.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	CreateSpec(db, bsgDir, CreateSpecInput{
		ID: "bsg-0001", Name: "main entry", Type: model.SpecTypeBehavior,
	})
	CreateLink(db, bsgDir, CreateLinkInput{
		SpecID: "bsg-0001", FilePath: "cmd/main.go", LinkType: model.LinkImplements, Scope: model.ScopeFile,
	})
	// Set file mtime to the past so verification timestamp is clearly after it
	past := time.Now().Add(-2 * time.Second)
	os.Chtimes(filepath.Join(root, "cmd", "main.go"), past, past)

	UpdateSpec(db, bsgDir, UpdateSpecInput{ID: "bsg-0001", Status: statusPtr(model.StatusAccepted)})
	UpdateSpec(db, bsgDir, UpdateSpecInput{ID: "bsg-0001", Status: statusPtr(model.StatusImplemented)})
	UpdateSpec(db, bsgDir, UpdateSpecInput{ID: "bsg-0001", Status: statusPtr(model.StatusVerified)})

	// No drift — file not modified since verification
	stats, err := GetCoverage(db)
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if len(stats.Drifted) != 0 {
		t.Fatalf("expected 0 drifted, got %d", len(stats.Drifted))
	}

	// Touch the file to cause drift
	time.Sleep(1100 * time.Millisecond)
	writeFile(t, filepath.Join(root, "cmd", "main.go"), "package main\nfunc main() { changed() }\n")

	stats, err = GetCoverage(db)
	if err != nil {
		t.Fatalf("coverage after touch: %v", err)
	}
	if len(stats.Drifted) != 1 {
		t.Fatalf("expected 1 drifted, got %d", len(stats.Drifted))
	}
	if stats.Drifted[0].Spec.ID != "bsg-0001" {
		t.Fatalf("drifted spec: got %s, want bsg-0001", stats.Drifted[0].Spec.ID)
	}

	// Re-verify to clear drift — sleep so verify timestamp is after file mtime
	time.Sleep(1100 * time.Millisecond)
	UpdateSpec(db, bsgDir, UpdateSpecInput{ID: "bsg-0001", Status: statusPtr(model.StatusVerified)})

	stats, err = GetCoverage(db)
	if err != nil {
		t.Fatalf("coverage after re-verify: %v", err)
	}
	if len(stats.Drifted) != 0 {
		t.Fatalf("expected 0 drifted after re-verify, got %d", len(stats.Drifted))
	}
}

func TestDriftNonVerifiedSpecsIgnored(t *testing.T) {
	root, bsgDir := setupProject(t)
	chdir(t, root)

	db, err := Open(filepath.Join(bsgDir, "bsg.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	CreateSpec(db, bsgDir, CreateSpecInput{
		ID: "bsg-0001", Name: "draft spec", Type: model.SpecTypeBehavior,
	})
	CreateLink(db, bsgDir, CreateLinkInput{
		SpecID: "bsg-0001", FilePath: "cmd/main.go", LinkType: model.LinkImplements, Scope: model.ScopeFile,
	})

	stats, err := GetCoverage(db)
	if err != nil {
		t.Fatalf("coverage: %v", err)
	}
	if len(stats.Drifted) != 0 {
		t.Fatalf("draft spec should not drift, got %d drifted", len(stats.Drifted))
	}
}
