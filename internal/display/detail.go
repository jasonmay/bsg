package display

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/jasonmay/bsg/internal/model"
)

func ShowSpec(w io.Writer, s *model.Spec, history []model.HistoryEntry) {
	fmt.Fprintf(w, "%-10s %s\n", "ID:", s.ID)
	fmt.Fprintf(w, "%-10s %s\n", "Name:", s.Name)
	fmt.Fprintf(w, "%-10s %s\n", "Type:", s.Type)
	fmt.Fprintf(w, "%-10s %s\n", "Status:", s.Status)
	if len(s.Tags) > 0 {
		fmt.Fprintf(w, "%-10s %s\n", "Tags:", strings.Join(s.Tags, ", "))
	}
	fmt.Fprintf(w, "%-10s %s\n", "Created:", s.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "%-10s %s\n", "Updated:", s.UpdatedAt.Format("2006-01-02 15:04:05"))
	fmt.Fprintf(w, "\n%s\n", s.Body)

	if len(history) > 0 {
		fmt.Fprintf(w, "\nHistory:\n")
		for _, h := range history {
			fmt.Fprintf(w, "  %s  %s: %s -> %s\n",
				h.ChangedAt.Format("2006-01-02 15:04"),
				h.Field, h.OldValue, h.NewValue)
		}
	}
}

type SpecJSON struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	Type      string   `json:"type"`
	Status    string   `json:"status"`
	Body      string   `json:"body"`
	Tags      []string `json:"tags,omitempty"`
	CreatedAt string   `json:"created_at"`
	UpdatedAt string   `json:"updated_at"`
}

func ShowSpecJSON(w io.Writer, s *model.Spec) error {
	out := SpecJSON{
		ID:        s.ID,
		Name:      s.Name,
		Type:      string(s.Type),
		Status:    string(s.Status),
		Body:      s.Body,
		Tags:      s.Tags,
		CreatedAt: s.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt: s.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
