package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status <id> <new-status>",
	Short: "Transition a spec's status",
	Long: `Move a spec through its lifecycle.

Valid statuses: draft, accepted, implemented, verified, paused, deprecated, archived

Transitions:
  draft       -> accepted, paused, deprecated, archived
  accepted    -> implemented, paused, deprecated, archived
  implemented -> verified, paused, deprecated, archived
  verified    -> verified (re-verify to clear drift), deprecated, archived
  paused      -> draft, accepted, implemented, deprecated, archived
  deprecated  -> archived`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		specID := args[0]
		newStatus := model.SpecStatus(args[1])

		err := db.UpdateSpec(DB, BsgDir(), db.UpdateSpecInput{
			ID:     specID,
			Status: &newStatus,
		})
		if err != nil {
			return err
		}

		fmt.Printf("%s -> %s\n", specID, newStatus)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statusCmd)
}
