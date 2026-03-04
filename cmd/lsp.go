package cmd

import (
	"github.com/jasonmay/bsg/internal/lsp"
	"github.com/spf13/cobra"
)

var lspCmd = &cobra.Command{
	Use:   "lsp",
	Short: "Start the BSG LSP server (stdio)",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		srv := lsp.NewServer(DB)
		return srv.RunStdio()
	},
}

func init() {
	rootCmd.AddCommand(lspCmd)
}
