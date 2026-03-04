package cmd

import (
	"os"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/display"
	"github.com/spf13/cobra"
)

var showJSON bool

var showCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show spec details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		spec, err := db.GetSpec(DB, args[0])
		if err != nil {
			return err
		}

		if showJSON {
			return display.ShowSpecJSON(os.Stdout, spec)
		}

		history, err := db.GetHistory(DB, args[0])
		if err != nil {
			return err
		}

		display.ShowSpec(os.Stdout, spec, history)
		return nil
	},
}

func init() {
	showCmd.Flags().BoolVar(&showJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(showCmd)
}
