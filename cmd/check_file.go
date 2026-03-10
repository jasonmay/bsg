package cmd

import (
	"fmt"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var checkFileCmd = &cobra.Command{
	Use:    "check-file <file_path>",
	Short:  "Show specs linked to a file (use 'inspect' instead)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		projectRoot := filepath.Dir(BsgDir())
		filePath := args[0]

		rel, err := resolveToProjectRelative(projectRoot, filePath)
		if err != nil {
			return err
		}

		results, err := db.GetSpecsForLocation(DB, rel, 0, 0)
		if err != nil {
			return err
		}

		if len(results) == 0 {
			return nil
		}

		fmt.Println("# Linked specs:")
		for _, r := range results {
			suffix := ""
			if r.Link.Symbol != "" {
				suffix = ":" + r.Link.Symbol
			}
			if rs := r.Link.RangeString(); rs != "" {
				if suffix != "" {
					suffix += " " + rs
				} else {
					suffix = " " + rs
				}
			}
			scopeTag := ""
			if r.Scope != "file" {
				scopeTag = fmt.Sprintf(" [%s]", r.Scope)
			}
			fmt.Printf("#   %s %q [%s] — %s%s%s\n",
				r.Spec.ID, r.Spec.Name, r.Spec.Status, r.Link.LinkType, suffix, scopeTag)
			if r.Spec.Body != "" {
				fmt.Printf("#     %s\n", r.Spec.Body)
			}
			if warning := checkLinkHealth(projectRoot, r); warning != "" {
				fmt.Printf("#     !! %s\n", warning)
			}
		}

		// Edge traversal — surface upstream specs
		var specIDs []string
		for _, r := range results {
			specIDs = append(specIDs, r.Spec.ID)
		}
		upstream, err := db.GetUpstreamSpecs(DB, specIDs)
		if err != nil {
			return err
		}
		if len(upstream) > 0 {
			fmt.Println("# Upstream specs (via edges):")
			for _, u := range upstream {
				fmt.Printf("#   %s %q [%s] (%s --%s-->)\n",
					u.Spec.ID, u.Spec.Name, u.Spec.Status, u.ViaSpec, u.Relation)
				if u.Spec.Body != "" {
					fmt.Printf("#     %s\n", u.Spec.Body)
				}
			}
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkFileCmd)
}
