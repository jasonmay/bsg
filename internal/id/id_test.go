package id

import (
	"strings"
	"testing"
)

func TestGenerate_NoCollision(t *testing.T) {
	t.Parallel()
	got := Generate("test spec", func(string) bool { return false })
	if !strings.HasPrefix(got, "bsg-") {
		t.Fatalf("expected bsg- prefix, got %q", got)
	}
	if len(got) != 8 { // "bsg-" + 4 hex
		t.Fatalf("expected 8 chars, got %d: %q", len(got), got)
	}
}

func TestGenerate_FirstCollision(t *testing.T) {
	t.Parallel()
	calls := 0
	got := Generate("test spec", func(id string) bool {
		calls++
		return calls == 1 // first ID collides
	})
	if !strings.HasPrefix(got, "bsg-") {
		t.Fatalf("expected bsg- prefix, got %q", got)
	}
	if len(got) != 10 { // "bsg-" + 6 hex
		t.Fatalf("expected 10 chars on collision, got %d: %q", len(got), got)
	}
}
