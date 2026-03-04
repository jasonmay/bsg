package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var setupRemove bool

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Configure integrations",
}

var setupClaudeCmd = &cobra.Command{
	Use:   "claude",
	Short: "Install Claude Code hooks for BSG",
	RunE: func(cmd *cobra.Command, args []string) error {
		settingsPath := filepath.Join(".claude", "settings.json")

		if setupRemove {
			return removeBSGHooks(settingsPath)
		}
		return installBSGHooks(settingsPath)
	},
}

func installBSGHooks(path string) error {
	settings, err := readSettings(path)
	if err != nil {
		return err
	}

	hooks := map[string]any{
		"SessionStart": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "bsg prime 2>/dev/null || true",
					},
				},
			},
		},
		"PreCompact": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "bsg prime --compact 2>/dev/null || true",
					},
				},
			},
		},
		"PostToolUse": []any{
			map[string]any{
				"matcher": "Edit|Write",
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "jq -r '.tool_input.file_path // empty' | xargs -I{} bsg check-file {} 2>/dev/null || true",
					},
				},
			},
		},
		"Stop": []any{
			map[string]any{
				"hooks": []any{
					map[string]any{
						"type":    "command",
						"command": "bsg prime --compact 2>/dev/null || true",
					},
				},
			},
		},
	}

	settings["hooks"] = hooks

	return writeSettings(path, settings)
}

func removeBSGHooks(path string) error {
	settings, err := readSettings(path)
	if err != nil {
		return err
	}

	delete(settings, "hooks")

	if len(settings) == 0 {
		os.Remove(path)
		fmt.Println("removed BSG hooks (settings file empty, deleted)")
		return nil
	}

	if err := writeSettings(path, settings); err != nil {
		return err
	}
	fmt.Println("removed BSG hooks")
	return nil
}

func readSettings(path string) (map[string]any, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]any), nil
		}
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var settings map[string]any
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	return settings, nil
}

func writeSettings(path string, settings map[string]any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(settings); err != nil {
		return fmt.Errorf("marshal settings: %w", err)
	}
	data := buf.Bytes()
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	fmt.Printf("installed BSG hooks in %s\n", path)
	return nil
}

func init() {
	setupClaudeCmd.Flags().BoolVar(&setupRemove, "remove", false, "remove BSG hooks")
	setupCmd.AddCommand(setupClaudeCmd)
	rootCmd.AddCommand(setupCmd)
}
