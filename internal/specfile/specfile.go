package specfile

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/model"
)

type SpecFile struct {
	ID     string     `json:"id"`
	Name   string     `json:"name"`
	Type   string     `json:"type"`
	Status string     `json:"status"`
	Body   string     `json:"body"`
	Tags   []string   `json:"tags,omitempty"`
	Links  []LinkFile `json:"links,omitempty"`
}

type LinkFile struct {
	File      string `json:"file"`
	Symbol    string `json:"symbol,omitempty"`
	Type      string `json:"type"`
	StartLine *int   `json:"start_line,omitempty"`
	StartCol  *int   `json:"start_col,omitempty"`
	EndLine   *int   `json:"end_line,omitempty"`
	EndCol    *int   `json:"end_col,omitempty"`
}

func SpecDir(bsgDir string) (string, error) {
	dir := filepath.Join(bsgDir, "specs")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create specs dir: %w", err)
	}
	return dir, nil
}

func WriteSpec(bsgDir string, spec *model.Spec, links []model.CodeLink) error {
	dir, err := SpecDir(bsgDir)
	if err != nil {
		return err
	}

	sf := SpecFile{
		ID:     spec.ID,
		Name:   spec.Name,
		Type:   string(spec.Type),
		Status: string(spec.Status),
		Body:   spec.Body,
		Tags:   spec.Tags,
	}

	for _, l := range links {
		lf := LinkFile{
			File:      l.FilePath,
			Symbol:    l.Symbol,
			Type:      string(l.LinkType),
			StartLine: l.StartLine,
			StartCol:  l.StartCol,
			EndLine:   l.EndLine,
			EndCol:    l.EndCol,
		}
		sf.Links = append(sf.Links, lf)
	}

	data, err := json.MarshalIndent(sf, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal spec file: %w", err)
	}
	data = append(data, '\n')

	path := filepath.Join(dir, spec.ID+".json")
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write spec file: %w", err)
	}
	return nil
}

func ReadSpec(path string) (*SpecFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read spec file: %w", err)
	}
	var sf SpecFile
	if err := json.Unmarshal(data, &sf); err != nil {
		return nil, fmt.Errorf("unmarshal spec file %s: %w", path, err)
	}
	return &sf, nil
}

func ReadAll(bsgDir string) ([]SpecFile, error) {
	dir := filepath.Join(bsgDir, "specs")
	matches, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, fmt.Errorf("glob spec files: %w", err)
	}
	var specs []SpecFile
	for _, path := range matches {
		sf, err := ReadSpec(path)
		if err != nil {
			return nil, err
		}
		specs = append(specs, *sf)
	}
	return specs, nil
}

func DeleteSpec(bsgDir string, id string) error {
	dir := filepath.Join(bsgDir, "specs")
	path := filepath.Join(dir, id+".json")
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove spec file: %w", err)
	}
	return nil
}
