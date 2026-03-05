package cmd

import (
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var (
	linkDependsOn    string
	linkRefines      string
	linkConflicts    string
	linkImplements   string
	linkSupersedes   string
)

var linkCmd = &cobra.Command{
	Use:   "link <from-id>",
	Short: "Create a spec-to-spec relationship",
	Long: `Create a directed edge between two specs.

Exactly one relation flag is required:
  --depends-on, --refines, --conflicts-with, --implements, --supersedes`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fromID := args[0]
		toID, relation, err := parseLinkFlags()
		if err != nil {
			return err
		}
		if err := db.CreateEdge(DB, BsgDir(), fromID, toID, relation); err != nil {
			return err
		}
		fmt.Printf("%s --%s--> %s\n", fromID, relation, toID)
		return nil
	},
}

func parseLinkFlags() (string, model.Relation, error) {
	flags := []struct {
		val      string
		relation model.Relation
	}{
		{linkDependsOn, model.RelDependsOn},
		{linkRefines, model.RelRefines},
		{linkConflicts, model.RelConflictsWith},
		{linkImplements, model.RelImplements},
		{linkSupersedes, model.RelSupersedes},
	}

	var toID string
	var rel model.Relation
	count := 0
	for _, f := range flags {
		if f.val != "" {
			toID = f.val
			rel = f.relation
			count++
		}
	}
	if count == 0 {
		return "", "", fmt.Errorf("exactly one relation flag required (--depends-on, --refines, --conflicts-with, --implements, --supersedes)")
	}
	if count > 1 {
		return "", "", fmt.Errorf("exactly one relation flag required, got %d", count)
	}
	return toID, rel, nil
}

func init() {
	linkCmd.Flags().StringVar(&linkDependsOn, "depends-on", "", "target spec ID")
	linkCmd.Flags().StringVar(&linkRefines, "refines", "", "target spec ID")
	linkCmd.Flags().StringVar(&linkConflicts, "conflicts-with", "", "target spec ID")
	linkCmd.Flags().StringVar(&linkImplements, "implements", "", "target spec ID")
	linkCmd.Flags().StringVar(&linkSupersedes, "supersedes", "", "target spec ID")
	rootCmd.AddCommand(linkCmd)
}
