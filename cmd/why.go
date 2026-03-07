package cmd

import (
	"github.com/spf13/cobra"
)

var whyCmd = &cobra.Command{
	Use:    "why <file[:line[:col]]>",
	Short:  "Show all specs applicable to a code location (use 'inspect' instead)",
	Hidden: true,
	Args:   cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return inspectCmd.RunE(cmd, args)
	},
}

func init() {
	whyCmd.Flags().Bool("recursive", false, "traverse edges to show upstream specs")
	rootCmd.AddCommand(whyCmd)
}
