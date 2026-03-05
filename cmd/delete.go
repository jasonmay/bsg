package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete a spec",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specID := args[0]
		if err := db.DeleteSpec(DB, BsgDir(), specID); err != nil {
			return err
		}
		fmt.Printf("deleted %s\n", specID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}
