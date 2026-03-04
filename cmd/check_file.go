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

		links, err := db.GetLinksByFile(DB, filePath)
		if err != nil {
			return err
		}

		if len(links) == 0 {
			return nil
		}

		fmt.Println("# Linked specs:")
		for _, l := range links {
			spec, err := db.GetSpec(DB, l.SpecID)
			if err != nil {
				continue
			}
			suffix := ""
			if l.Symbol != "" {
				suffix = ":" + l.Symbol
			}
			if r := l.RangeString(); r != "" {
				if suffix != "" {
					suffix += " " + r
				} else {
					suffix = " " + r
				}
			}
			fmt.Printf("#   %s %q [%s] — %s%s\n",
				spec.ID, spec.Name, spec.Status, l.LinkType, suffix)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(checkFileCmd)
}
