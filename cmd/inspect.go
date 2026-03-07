package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var inspectCmd = &cobra.Command{
	Use:   "inspect [path[:line[:col]]]",
	Short: "Show specs linked to a file, directory, or cwd",
	Long: `Show all specs applicable to a code location.

With no arguments, shows specs for files under the current directory.
With a file path, shows specs at every scope level for that file.
With a directory path, shows specs for all files under it.
Supports file:line:col syntax for position-level inspection.

Use --recursive to also show upstream specs via edges.

Examples:
  bsg inspect                        # specs for cwd
  bsg inspect cmd/                   # specs under cmd/
  bsg inspect cmd/init.go            # specs for a file
  bsg inspect cmd/init.go:15         # includes range matches
  bsg inspect --recursive            # also show upstream specs`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := filepath.Dir(BsgDir())

		if len(args) == 0 || isDirectory(args[0]) {
			dir := "."
			if len(args) > 0 {
				dir = args[0]
			}
			return inspectDirectory(cmd, projectRoot, dir)
		}

		return inspectFile(cmd, projectRoot, args[0])
	},
}

func resolveToProjectRelative(projectRoot, path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("abs path: %w", err)
	}
	rel, err := filepath.Rel(projectRoot, abs)
	if err != nil {
		return "", fmt.Errorf("rel path: %w", err)
	}
	if strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path %s is outside the project root", path)
	}
	return rel, nil
}

func inspectDirectory(cmd *cobra.Command, projectRoot, dir string) error {
	rel, err := resolveToProjectRelative(projectRoot, dir)
	if err != nil {
		return err
	}
	if rel == "." {
		rel = ""
	}

	results, err := db.GetSpecsForDirectory(DB, rel)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("no specs found")
		return nil
	}

	// Group by file
	type fileGroup struct {
		file    string
		results []db.ScopedResult
	}
	var groups []fileGroup
	groupMap := make(map[string]int)
	for _, r := range results {
		idx, ok := groupMap[r.Link.FilePath]
		if !ok {
			idx = len(groups)
			groupMap[r.Link.FilePath] = idx
			groups = append(groups, fileGroup{file: r.Link.FilePath})
		}
		groups[idx].results = append(groups[idx].results, r)
	}

	for _, g := range groups {
		fmt.Printf("%s:\n", g.file)
		for _, r := range g.results {
			detail := formatLinkDetail(r)
			fmt.Printf("  %s %q [%s] (%s)%s\n",
				r.Spec.ID, r.Spec.Name, r.Spec.Status, r.Link.LinkType, detail)
		}
		fmt.Println()
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	if recursive {
		return printUpstream(results)
	}
	return nil
}

func inspectFile(cmd *cobra.Command, projectRoot, arg string) error {
	parsed, err := parseFileArg(arg)
	if err != nil {
		return fmt.Errorf("parse arg: %w", err)
	}

	rel, err := resolveToProjectRelative(projectRoot, parsed.FilePath)
	if err != nil {
		return err
	}

	line := 0
	col := 0
	if parsed.StartLine != nil {
		line = *parsed.StartLine
	}
	if parsed.StartCol != nil {
		col = *parsed.StartCol
	}

	results, err := db.GetSpecsForLocation(DB, rel, line, col)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		fmt.Println("no specs found")
		return nil
	}

	fmt.Println(arg)
	fmt.Println()

	scopeOrder := []model.LinkScope{
		model.ScopeCodebase,
		model.ScopeDirectory,
		model.ScopeFile,
		model.ScopeRange,
		model.ScopeSymbol,
	}
	scopeNames := map[model.LinkScope]string{
		model.ScopeCodebase:  "Codebase",
		model.ScopeDirectory: "Directory",
		model.ScopeFile:      "File",
		model.ScopeRange:     "Range",
		model.ScopeSymbol:    "Symbol",
	}

	grouped := make(map[model.LinkScope][]db.ScopedResult)
	for _, r := range results {
		grouped[r.Scope] = append(grouped[r.Scope], r)
	}

	for _, scope := range scopeOrder {
		items := grouped[scope]
		if len(items) == 0 {
			continue
		}
		label := scopeNames[scope]
		if scope == model.ScopeRange {
			for _, item := range items {
				r := item.Link.RangeString()
				if r != "" {
					label = fmt.Sprintf("Range (%s)", r)
					break
				}
			}
		}
		fmt.Printf("%s:\n", label)
		for _, item := range items {
			detail := formatLinkDetail(item)
			fmt.Printf("  %s %q [%s] (%s)%s\n",
				item.Spec.ID, item.Spec.Name, item.Spec.Status, item.Link.LinkType, detail)
		}
		fmt.Println()
	}

	recursive, _ := cmd.Flags().GetBool("recursive")
	if recursive {
		return printUpstream(results)
	}
	return nil
}

func formatLinkDetail(r db.ScopedResult) string {
	detail := ""
	if r.Link.Symbol != "" {
		detail = fmt.Sprintf(" :%s", r.Link.Symbol)
	}
	return detail
}

func printUpstream(results []db.ScopedResult) error {
	var specIDs []string
	for _, r := range results {
		specIDs = append(specIDs, r.Spec.ID)
	}
	upstream, err := db.GetUpstreamSpecs(DB, specIDs)
	if err != nil {
		return fmt.Errorf("get upstream: %w", err)
	}
	if len(upstream) > 0 {
		fmt.Println("Upstream (via edges):")
		for _, u := range upstream {
			fmt.Printf("  %s %q [%s] (%s --%s-->)\n",
				u.Spec.ID, u.Spec.Name, u.Spec.Type, u.ViaSpec, u.Relation)
		}
		fmt.Println()
	}
	return nil
}

func isDirectory(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func init() {
	inspectCmd.Flags().Bool("recursive", false, "traverse edges to show upstream specs")
	rootCmd.AddCommand(inspectCmd)
}
