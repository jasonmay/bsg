package db

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jasonmay/bsg/internal/model"
)

func TestInitializeAndOpen(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := Initialize(dbPath); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// verify tables exist
	for _, table := range []string{"specs", "edges", "code_links", "history"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != nil {
			t.Fatalf("table %s not found: %v", table, err)
		}
	}
}

func TestSpecCRUD(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	if err := Initialize(dbPath); err != nil {
		t.Fatalf("initialize: %v", err)
	}
	db, err := Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer db.Close()

	// Create
	err = CreateSpec(db, CreateSpecInput{
		ID:   "bsg-0001",
		Name: "Test Spec",
		Type: model.SpecTypeBehavior,
		Body: "must do X",
		Tags: []string{"test", "foo"},
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get
	spec, err := GetSpec(db, "bsg-0001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if spec.Name != "Test Spec" {
		t.Fatalf("name=%q, want %q", spec.Name, "Test Spec")
	}
	if spec.Status != model.StatusDraft {
		t.Fatalf("status=%q, want %q", spec.Status, model.StatusDraft)
	}
	if len(spec.Tags) != 2 || spec.Tags[0] != "test" {
		t.Fatalf("tags=%v, want [test foo]", spec.Tags)
	}

	// Update
	newName := "Updated Spec"
	err = UpdateSpec(db, UpdateSpecInput{ID: "bsg-0001", Name: &newName})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	spec, _ = GetSpec(db, "bsg-0001")
	if spec.Name != "Updated Spec" {
		t.Fatalf("name=%q after update", spec.Name)
	}

	// History
	hist, err := GetHistory(db, "bsg-0001")
	if err != nil {
		t.Fatalf("history: %v", err)
	}
	if len(hist) != 1 || hist[0].Field != "name" {
		t.Fatalf("expected 1 history entry for name, got %d", len(hist))
	}

	// List
	specs, err := ListSpecs(db, ListSpecsInput{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(specs) != 1 {
		t.Fatalf("list count=%d, want 1", len(specs))
	}

	// Delete
	if err := DeleteSpec(db, "bsg-0001"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	_, err = GetSpec(db, "bsg-0001")
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestFindDB(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	bsgDir := filepath.Join(dir, ".bsg")
	os.MkdirAll(bsgDir, 0755)
	Initialize(filepath.Join(bsgDir, "bsg.db"))

	// from a subdirectory
	sub := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(sub, 0755)

	orig, _ := os.Getwd()
	os.Chdir(sub)
	defer os.Chdir(orig)

	path, err := FindDB()
	if err != nil {
		t.Fatalf("finddb: %v", err)
	}
	if filepath.Base(filepath.Dir(path)) != ".bsg" {
		t.Fatalf("unexpected path: %s", path)
	}
}
