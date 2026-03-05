package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/specfile"
	"github.com/spf13/cobra"
)

var summarizeCmd = &cobra.Command{
	Use:   "summarize",
	Short: "Print a summary of all specs",
	RunE: func(cmd *cobra.Command, args []string) error {
		summary, err := specfile.Summarize(BsgDir())
		if err != nil {
			return err
		}
		fmt.Print(summary)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(summarizeCmd)
}
