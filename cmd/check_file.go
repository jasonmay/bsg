package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var checkFileCmd = &cobra.Command{
	Use:   "check-file <file_path>",
	Short: "Show specs linked to a file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		filePath := args[0]

		results, err := db.GetSpecsForLocation(DB, filePath, 0, 0)
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
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkFileCmd)
}
