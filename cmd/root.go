package cmd

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var DB *sql.DB
var DBPath string

func BsgDir() string {
	return filepath.Dir(DBPath)
}

var skipDBCommands = map[string]bool{
	"init":  true,
	"setup": true,
}

var rootCmd = &cobra.Command{
	Use:   "bsg",
	Short: "A spec graph CLI",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if skipDBCommands[cmd.Name()] {
			return nil
		}
		var err error
		DBPath, err = db.FindDB()
		if err != nil {
			return fmt.Errorf("no .bsg/bsg.db found (run 'bsg init' first): %w", err)
		}
		DB, err = db.Open(DBPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}
		return nil
	},
	PersistentPostRunE: func(cmd *cobra.Command, args []string) error {
		if DB != nil {
			return DB.Close()
		}
		return nil
	},
}

func Execute() error {
	return rootCmd.Execute()
}
