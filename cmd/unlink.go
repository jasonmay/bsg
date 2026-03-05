package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var unlinkRelation string

var unlinkCmd = &cobra.Command{
	Use:   "unlink <from-id> <to-id>",
	Short: "Remove a spec-to-spec relationship",
	Long: `Remove edges between two specs.

Without --relation, removes all edges from <from-id> to <to-id>.
With --relation, removes only the specified relation.`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromID := args[0]
		toID := args[1]

		var rel *model.Relation
		if unlinkRelation != "" {
			r, err := model.ParseRelation(unlinkRelation)
			if err != nil {
				return err
			}
			rel = &r
		}

		if err := db.DeleteEdge(DB, BsgDir(), fromID, toID, rel); err != nil {
			return err
		}
		if rel != nil {
			fmt.Printf("removed %s edge: %s -> %s\n", *rel, fromID, toID)
		} else {
			fmt.Printf("removed all edges: %s -> %s\n", fromID, toID)
		}
		return nil
	},
}

func init() {
	unlinkCmd.Flags().StringVar(&unlinkRelation, "relation", "", "specific relation to remove")
	rootCmd.AddCommand(unlinkCmd)
}
