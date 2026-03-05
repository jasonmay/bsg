package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
		"PreToolUse": []any{
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

	existingHooks, _ := settings["hooks"].(map[string]any)
	if existingHooks == nil {
		existingHooks = make(map[string]any)
	}

	for hookType, entries := range hooks {
		bsgEntries, _ := entries.([]any)
		existingEntries, _ := existingHooks[hookType].([]any)

		// Remove existing BSG entries, then append new ones
		var filtered []any
		for _, entry := range existingEntries {
			if !isBSGHookEntry(entry) {
				filtered = append(filtered, entry)
			}
		}
		existingHooks[hookType] = append(filtered, bsgEntries...)
	}

	settings["hooks"] = existingHooks

	if err := writeSettings(path, settings); err != nil {
		return err
	}

	return addClaudeMDReference()
}

func isBSGHookEntry(entry any) bool {
	m, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hooks, _ := m["hooks"].([]any)
	for _, h := range hooks {
		hm, _ := h.(map[string]any)
		cmd, _ := hm["command"].(string)
		if strings.Contains(cmd, "bsg ") {
			return true
		}
	}
	return false
}

const bsgClaudeMDLine = "Refer to .bsg/README.md for BSG (Behavioral Spec Graph) usage and commands."

func addClaudeMDReference() error {
	claudeMD := "CLAUDE.md"
	data, err := os.ReadFile(claudeMD)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read CLAUDE.md: %w", err)
	}

	if bytes.Contains(data, []byte(".bsg/README.md")) {
		return nil
	}

	f, err := os.OpenFile(claudeMD, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("open CLAUDE.md: %w", err)
	}
	defer f.Close()

	line := "\n" + bsgClaudeMDLine + "\n"
	if len(data) == 0 {
		line = bsgClaudeMDLine + "\n"
	}
	if _, err := f.WriteString(line); err != nil {
		return fmt.Errorf("write CLAUDE.md: %w", err)
	}
	fmt.Println("added BSG reference to CLAUDE.md")
	return nil
}

func removeBSGHooks(path string) error {
	settings, err := readSettings(path)
	if err != nil {
		return err
	}

	existingHooks, _ := settings["hooks"].(map[string]any)
	if existingHooks == nil {
		fmt.Println("no BSG hooks found")
		return nil
	}

	for hookType, entries := range existingHooks {
		entryList, _ := entries.([]any)
		var filtered []any
		for _, entry := range entryList {
			if !isBSGHookEntry(entry) {
				filtered = append(filtered, entry)
			}
		}
		if len(filtered) == 0 {
			delete(existingHooks, hookType)
		} else {
			existingHooks[hookType] = filtered
		}
	}

	if len(existingHooks) == 0 {
		delete(settings, "hooks")
	}

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
