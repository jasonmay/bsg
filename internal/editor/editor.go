package editor

import (
	"fmt"
	"os"
	"os/exec"
)

func Open(initial string) (string, error) {
	editor := os.Getenv("EDITOR")
	if editor == "" {
		return "", fmt.Errorf("$EDITOR not set")
	}

	f, err := os.CreateTemp("", "bsg-*.md")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := f.Name()
	defer os.Remove(tmpPath)

	if _, err := f.WriteString(initial); err != nil {
		f.Close()
		return "", fmt.Errorf("write template: %w", err)
	}
	f.Close()

	cmd := exec.Command(editor, tmpPath)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("editor: %w", err)
	}

	content, err := os.ReadFile(tmpPath)
	if err != nil {
		return "", fmt.Errorf("read edited file: %w", err)
	}

	result := string(content)
	if result == initial {
		return "", fmt.Errorf("aborted: no changes made")
	}
	return result, nil
}
