package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:   "why <file[:line[:col]]>",
	Short: "Show all specs applicable to a code location",
	Long: `Show specs at every scope level for a given file and position.

Examples:
  bsg why src/weight.go              # file-level lookup
  bsg why src/weight.go:15           # includes range matches
  bsg why src/weight.go:15 --recursive  # also show upstream specs via edges`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		parsed, err := parseFileArg(args[0])
		if err != nil {
			return fmt.Errorf("parse arg: %w", err)
		}

		line := 0
		col := 0
		if parsed.StartLine != nil {
			line = *parsed.StartLine
		}
		if parsed.StartCol != nil {
			col = *parsed.StartCol
		}

		results, err := db.GetSpecsForLocation(DB, parsed.FilePath, line, col)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			fmt.Println("no specs found")
			return nil
		}

		fmt.Println(args[0])
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
		var specIDs []string
		for _, r := range results {
			grouped[r.Scope] = append(grouped[r.Scope], r)
			specIDs = append(specIDs, r.Spec.ID)
		}

		for _, scope := range scopeOrder {
			items := grouped[scope]
			if len(items) == 0 {
				continue
			}
			label := scopeNames[scope]
			if scope == model.ScopeRange {
				// annotate with line range info
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
				detail := ""
				if item.Link.Symbol != "" {
					detail = fmt.Sprintf(" :%s", item.Link.Symbol)
				}
				fmt.Printf("  %s %q [%s] (%s)%s\n",
					item.Spec.ID, item.Spec.Name, item.Spec.Type, item.Link.LinkType, detail)
			}
			fmt.Println()
		}

		recursive, _ := cmd.Flags().GetBool("recursive")
		if recursive && len(specIDs) > 0 {
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
		}

		return nil
	},
}

func init() {
	whyCmd.Flags().Bool("recursive", false, "traverse edges to show upstream specs")
	rootCmd.AddCommand(whyCmd)
}
