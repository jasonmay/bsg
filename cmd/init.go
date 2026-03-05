package cmd

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var initForce bool

//go:embed bsg_readme.md
var bsgReadme string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new BSG spec database",
	RunE: func(cmd *cobra.Command, args []string) error {
		dir := filepath.Join(".", ".bsg")
		dbPath := filepath.Join(dir, "bsg.db")
		dbExists := false

		if _, err := os.Stat(dbPath); err == nil {
			dbExists = true
			if initForce {
				os.Remove(dbPath)
				os.Remove(dbPath + "-wal")
				os.Remove(dbPath + "-shm")
				dbExists = false
			}
		}

		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create .bsg dir: %w", err)
		}

		if err := os.MkdirAll(filepath.Join(dir, "specs"), 0755); err != nil {
			return fmt.Errorf("create .bsg/specs dir: %w", err)
		}

		gitignore := filepath.Join(dir, ".gitignore")
		if _, err := os.Stat(gitignore); os.IsNotExist(err) {
			content := "bsg.db\nbsg.db-wal\nbsg.db-shm\nspecs/.synced\n"
			if err := os.WriteFile(gitignore, []byte(content), 0644); err != nil {
				return fmt.Errorf("write .gitignore: %w", err)
			}
		}

		readmePath := filepath.Join(dir, "README.md")
		if err := os.WriteFile(readmePath, []byte(bsgReadme), 0644); err != nil {
			return fmt.Errorf("write README.md: %w", err)
		}

		if !dbExists {
			if err := db.Initialize(dbPath); err != nil {
				return fmt.Errorf("initialize db: %w", err)
			}
		}

		fmt.Println("initialized .bsg/")
		return nil
	},
}

func init() {
	initCmd.Flags().BoolVar(&initForce, "force", false, "reinitialize if exists")
	rootCmd.AddCommand(initCmd)
}
