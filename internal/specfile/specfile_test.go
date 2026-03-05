package specfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jasonmay/bsg/internal/model"
)

func setupBsgDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bsgDir := filepath.Join(dir, ".bsg")
	os.MkdirAll(filepath.Join(bsgDir, "specs"), 0755)
	return bsgDir
}

func writeTestSpec(t *testing.T, bsgDir string, spec *model.Spec, links []model.CodeLink) {
	t.Helper()
	if err := WriteSpec(bsgDir, spec, links, nil); err != nil {
		t.Fatalf("WriteSpec(%s): %v", spec.ID, err)
	}
}

func summarize(t *testing.T, bsgDir string) string {
	t.Helper()
	s, err := Summarize(bsgDir)
	if err != nil {
		t.Fatalf("Summarize: %v", err)
	}
	return s
}

func TestSummarize_NoSpecs(t *testing.T) {
	t.Parallel()
	bsgDir := setupBsgDir(t)
	out := summarize(t, bsgDir)
	if !strings.Contains(out, "No specs yet.") {
		t.Fatalf("expected 'No specs yet.', got:\n%s", out)
	}
}

func TestSummarize_GroupsByStatus(t *testing.T) {
	t.Parallel()
	bsgDir := setupBsgDir(t)

	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0001", Name: "Draft Spec", Type: model.SpecTypeBehavior, Status: model.StatusDraft,
	}, nil)
	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0002", Name: "Accepted Spec", Type: model.SpecTypeConstraint, Status: model.StatusAccepted,
		Tags: []string{"auth", "input"},
	}, nil)
	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0003", Name: "Another Draft", Type: model.SpecTypeBehavior, Status: model.StatusDraft,
	}, nil)

	out := summarize(t, bsgDir)

	// Status summary
	if !strings.Contains(out, "draft") || !strings.Contains(out, "2") {
		t.Errorf("expected draft count 2 in summary")
	}
	if !strings.Contains(out, "accepted") || !strings.Contains(out, "1") {
		t.Errorf("expected accepted count 1 in summary")
	}

	// Section headers
	if !strings.Contains(out, "Draft") {
		t.Errorf("expected Draft section")
	}
	if !strings.Contains(out, "Accepted") {
		t.Errorf("expected Accepted section")
	}
	if strings.Contains(out, "Implemented") {
		t.Errorf("should not have Implemented section")
	}

	// Tags
	if !strings.Contains(out, "[auth, input]") {
		t.Errorf("expected tags '[auth, input]'")
	}

	// Sorted by ID within group
	idx1 := strings.Index(out, "bsg-0001")
	idx3 := strings.Index(out, "bsg-0003")
	if idx1 > idx3 {
		t.Errorf("expected bsg-0001 before bsg-0003 (sorted by ID)")
	}
}

func TestSummarize_FilesIndex(t *testing.T) {
	t.Parallel()
	bsgDir := setupBsgDir(t)

	startLine := 10
	endLine := 25
	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0001", Name: "Weight Validation", Type: model.SpecTypeBehavior, Status: model.StatusDraft,
	}, []model.CodeLink{
		{FilePath: "src/weight.go", LinkType: model.LinkImplements, StartLine: &startLine, EndLine: &endLine},
	})
	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0002", Name: "Auth Check", Type: model.SpecTypeBehavior, Status: model.StatusDraft,
	}, []model.CodeLink{
		{FilePath: "src/weight.go", LinkType: model.LinkImplements},
		{FilePath: "src/auth.go", LinkType: model.LinkImplements},
	})

	out := summarize(t, bsgDir)

	if !strings.Contains(out, "Files") {
		t.Fatalf("expected Files section")
	}
	if !strings.Contains(out, "src/weight.go: bsg-0001, bsg-0002") {
		t.Errorf("expected weight.go with both specs")
	}
	if !strings.Contains(out, "src/auth.go: bsg-0002") {
		t.Errorf("expected auth.go with bsg-0002")
	}

	// Link counts
	if !strings.Contains(out, "(1 file)") {
		t.Errorf("expected '(1 file)' for bsg-0001")
	}
	if !strings.Contains(out, "(2 files)") {
		t.Errorf("expected '(2 files)' for bsg-0002")
	}
}

func TestSummarize_StatusChange(t *testing.T) {
	t.Parallel()
	bsgDir := setupBsgDir(t)

	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0001", Name: "First", Type: model.SpecTypeBehavior, Status: model.StatusDraft,
	}, nil)

	out := summarize(t, bsgDir)
	if !strings.Contains(out, "Draft") {
		t.Fatalf("expected Draft section")
	}

	writeTestSpec(t, bsgDir, &model.Spec{
		ID: "bsg-0001", Name: "First", Type: model.SpecTypeBehavior, Status: model.StatusAccepted,
	}, nil)

	out = summarize(t, bsgDir)
	if strings.Contains(out, "── Draft") {
		t.Errorf("should not have Draft section after status change")
	}
	if !strings.Contains(out, "Accepted") {
		t.Errorf("expected Accepted section after status change")
	}
}
