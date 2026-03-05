package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/spf13/cobra"
)

var initForce bool

const bsgReadme = `# BSG — Behavioral Spec Graph

BSG is an LLM-first spec tracking tool for requirements, expectations, and intentions.
Specs live as JSON files in .bsg/specs/ and are version-controlled with git. The SQLite
database is a local cache rebuilt automatically from these files.

## IDs

Every spec gets an auto-generated ID like ` + "`bsg-a3f2`" + `. The ` + "`bsg add`" + ` command prints the
generated ID to stdout. All other commands use this ID, not the spec name.

## Commands

| Command | Description |
|---------|-------------|
| bsg add <name> --type <type> [--body <text>] [--tag <csv>] | Create a spec, prints generated ID |
| bsg show <id> | Display a spec and its history |
| bsg status <id> <new-status> | Transition spec status (e.g. draft -> accepted) |
| bsg delete <id> | Delete a spec and its links |
| bsg trace <id> --file <path> [--as type] | Link a spec to code (--as: implements, tests, documents) |
| bsg untrace <id> <file> | Remove a code link |
| bsg prime | Show spec coverage and status |
| bsg sync | Rebuild database from spec files |
| bsg check-file <path> | Show specs linked to a file |

## Spec Types

| Type | When to use |
|------|-------------|
| behavior | What the system should do — user-visible actions, responses, side effects |
| constraint | Limits and rules — validation, rate limits, size bounds, permissions |
| interface | API contracts — endpoints, function signatures, protocols, data formats |
| data-shape | Data structures — schemas, models, field definitions, relationships |
| invariant | Things that must always be true — consistency rules, ordering guarantees |

## Spec Lifecycle

draft -> accepted -> implemented -> verified -> deprecated -> archived

Any status can also transition directly to archived.

## Trace syntax

` + "`--file`" + ` accepts: ` + "`file`" + `, ` + "`file:Symbol`" + `, ` + "`file:10-25`" + ` (line range), ` + "`file:10:5-25:0`" + ` (line:col range)

## Worked Example

` + "```" + `
$ bsg add "Weight entries must be positive" --type constraint --body "Reject zero or negative weight values at input" --tag validation,weight
bsg-7f1a

$ bsg trace bsg-7f1a --file src/weight.go:ValidateWeight
traced bsg-7f1a -> src/weight.go:ValidateWeight (implements)

$ bsg trace bsg-7f1a --file src/weight_test.go --as tests
traced bsg-7f1a -> src/weight_test.go (tests)

$ bsg status bsg-7f1a accepted
bsg-7f1a -> accepted

$ bsg status bsg-7f1a implemented
bsg-7f1a -> implemented

$ bsg show bsg-7f1a
ID:         bsg-7f1a
Name:       Weight entries must be positive
Type:       constraint
Status:     implemented
...

$ bsg check-file src/weight.go
src/weight.go:
  bsg-7f1a "Weight entries must be positive" [constraint/implemented] :ValidateWeight (implements)

$ bsg delete bsg-7f1a
deleted bsg-7f1a
` + "```" + `

## File Format

Spec files in .bsg/specs/<id>.json contain: id, name, type, status, body, tags, and links.
These files are the source of truth — commit them to git.
`

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
