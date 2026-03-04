package cmd

import (
	"fmt"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/editor"
	"github.com/jasonmay/bsg/internal/id"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var (
	addType string
	addTags string
	addBody string
)

var addCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create a new spec",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]

		specType, err := model.ParseSpecType(addType)
		if err != nil {
			return err
		}

		body := addBody
		if body == "" {
			body, err = editor.Open("# " + name + "\n\n")
			if err != nil {
				return err
			}
		}

		var tags []string
		if addTags != "" {
			tags = strings.Split(addTags, ",")
			for i := range tags {
				tags[i] = strings.TrimSpace(tags[i])
			}
		}

		specID := id.Generate(name, func(candidate string) bool {
			return db.IDExists(DB, candidate)
		})

		err = db.CreateSpec(DB, db.CreateSpecInput{
			ID:   specID,
			Name: name,
			Type: specType,
			Body: body,
			Tags: tags,
		})
		if err != nil {
			return err
		}

		fmt.Println(specID)
		return nil
	},
}

func init() {
	addCmd.Flags().StringVar(&addType, "type", "", "spec type (behavior, constraint, interface, data-shape, invariant)")
	addCmd.Flags().StringVar(&addTags, "tag", "", "comma-separated tags")
	addCmd.Flags().StringVar(&addBody, "body", "", "spec body (opens $EDITOR if omitted)")
	addCmd.MarkFlagRequired("type")
	rootCmd.AddCommand(addCmd)
}
