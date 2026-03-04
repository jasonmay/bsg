package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var initForce bool

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new BSG spec database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := filepath.Join(".", ".bsg")
		dbPath := filepath.Join(dir, "bsg.db")

		if _, err := os.Stat(dbPath); err == nil && !initForce {
			return fmt.Errorf(".bsg/bsg.db already exists (use --force to reinitialize)")
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create .bsg dir: %w", err)
		}

		if initForce {
			os.Remove(dbPath)
		}

		if err := db.Initialize(dbPath); err != nil {
			return fmt.Errorf("initialize db: %w", err)
		}

		fmt.Println("initialized .bsg/bsg.db")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "reinitialize if exists")
	rootCmd.AddCommand(initCmd)
}
