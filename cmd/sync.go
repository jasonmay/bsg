package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Rebuild database from spec files",
	Long:  "Force resync: deletes the database and rebuilds it from .bsg/specs/*.json files.",
	RunE: func(cmd *cobra.Command, args []string) error {
		DB.Close()
		DB = nil

		os.Remove(DBPath)
		os.Remove(DBPath + "-wal")
		os.Remove(DBPath + "-shm")

		if err := db.Initialize(DBPath); err != nil {
			return fmt.Errorf("initialize db: %w", err)
		}

		// Remove sync marker after Initialize so Open() triggers SyncFromFiles
		os.Remove(filepath.Join(BsgDir(), "specs", ".synced"))

		var err error
		DB, err = db.Open(DBPath)
		if err != nil {
			return fmt.Errorf("open db: %w", err)
		}

		var count int
		DB.QueryRow(`SELECT COUNT(*) FROM specs`).Scan(&count)
		fmt.Printf("synced %d specs from files\n", count)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(syncCmd)
}
