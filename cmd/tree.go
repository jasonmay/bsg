package cmd

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/display"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "Show file tree of spec-linked files",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := filepath.Dir(BsgDir())

		prefix := ""
		if len(args) > 0 {
			rel, err := resolveToProjectRelative(projectRoot, args[0])
			if err != nil {
				return err
			}
			if rel != "." {
				prefix = rel
			}
		}

		results, err := db.GetSpecsForDirectory(DB, prefix)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("no spec-linked files found")
			return nil
		}

		driftedIDs := db.GetDriftedIDs(DB)

		// Group specs by file path
		type fileEntry struct {
			specs []db.ScopedResult
		}
		fileMap := make(map[string]*fileEntry)
		for _, r := range results {
			e, ok := fileMap[r.Link.FilePath]
			if !ok {
				e = &fileEntry{}
				fileMap[r.Link.FilePath] = e
			}
			e.specs = append(e.specs, r)
		}

		// Build tree — strip prefix from paths
		trimPrefix := ""
		if prefix != "" {
			trimPrefix = prefix + "/"
		}
		root := &treeNode{name: ".", children: make(map[string]*treeNode)}
		for path, entry := range fileMap {
			rel := strings.TrimPrefix(path, trimPrefix)
			rel = strings.TrimSuffix(rel, "/")
			parts := strings.Split(rel, "/")
			node := root
			for _, part := range parts {
				if part == "" {
					continue
				}
				child, ok := node.children[part]
				if !ok {
					child = &treeNode{name: part, children: make(map[string]*treeNode)}
					node.children[part] = child
				}
				node = child
			}
			node.specs = append(node.specs, entry.specs...)
		}

		// Render
		label := "."
		if prefix != "" {
			label = prefix
		}
		fmt.Println(label)
		printTree(root, "", driftedIDs)

		return nil
	},
}

type treeNode struct {
	name     string
	children map[string]*treeNode
	specs    []db.ScopedResult
}

func printTree(node *treeNode, prefix string, driftedIDs map[string]bool) {
	names := make([]string, 0, len(node.children))
	for name := range node.children {
		names = append(names, name)
	}
	sort.Strings(names)

	for i, name := range names {
		child := node.children[name]
		isLast := i == len(names)-1

		connector := "├── "
		childPrefix := prefix + "│   "
		if isLast {
			connector = "└── "
			childPrefix = prefix + "    "
		}

		if len(child.children) > 0 {
			annotation := formatSpecAnnotation(child.specs, driftedIDs)
			if annotation != "" {
				fmt.Printf("%s%s%-24s%s\n", prefix, connector, name+"/", annotation)
			} else {
				fmt.Printf("%s%s%s/\n", prefix, connector, name)
			}
			printTree(child, childPrefix, driftedIDs)
		} else {
			annotation := formatSpecAnnotation(child.specs, driftedIDs)
			fmt.Printf("%s%s%-24s%s\n", prefix, connector, name, annotation)
		}
	}
}

func formatSpecAnnotation(specs []db.ScopedResult, driftedIDs map[string]bool) string {
	if len(specs) == 0 {
		return ""
	}

	seen := make(map[string]bool)
	var parts []string
	for _, r := range specs {
		if seen[r.Spec.ID] {
			continue
		}
		seen[r.Spec.ID] = true

		drift := ""
		if driftedIDs[r.Spec.ID] {
			drift = display.Gray + "*" + display.Reset
		}

		color := display.StatusColor(r.Spec.Status)
		part := fmt.Sprintf("%s%s%s%s", color, r.Spec.ID, drift, display.Reset)
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
