package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/jasonmay/bsg/internal/specfile"
	"github.com/spf13/cobra"
)

var tagsJSON bool

type tagSpec struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

var tagsCmd = &cobra.Command{
	Use:   "tags [prefix]",
	Short: "List tags with counts, or filter by prefix",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specs, err := specfile.ReadAll(BsgDir())
		if err != nil {
			return err
		}

		tagMap := make(map[string][]tagSpec)
		for _, s := range specs {
			for _, t := range s.Tags {
				tagMap[t] = append(tagMap[t], tagSpec{ID: s.ID, Name: s.Name})
			}
		}

		var prefix string
		if len(args) > 0 {
			prefix = args[0]
		}

		var tags []string
		for t := range tagMap {
			if prefix == "" || strings.HasPrefix(t, prefix) {
				tags = append(tags, t)
			}
		}
		sort.Strings(tags)

		if len(tags) == 0 {
			if prefix != "" {
				fmt.Fprintf(os.Stderr, "no tags matching prefix %q\n", prefix)
			} else {
				fmt.Fprintln(os.Stderr, "no tags")
			}
			return nil
		}

		if prefix != "" {
			return printTagsDetail(tags, tagMap)
		}
		return printTagsCounts(tags, tagMap)
	},
}

func printTagsCounts(tags []string, tagMap map[string][]tagSpec) error {
	if tagsJSON {
		enc := json.NewEncoder(os.Stdout)
		for _, t := range tags {
			ids := make([]string, len(tagMap[t]))
			for i, s := range tagMap[t] {
				ids[i] = s.ID
			}
			enc.Encode(struct {
				Tag   string   `json:"tag"`
				Count int      `json:"count"`
				Specs []string `json:"specs"`
			}{Tag: t, Count: len(tagMap[t]), Specs: ids})
		}
		return nil
	}

	maxLen := 0
	for _, t := range tags {
		if len(t) > maxLen {
			maxLen = len(t)
		}
	}
	for _, t := range tags {
		fmt.Printf("%-*s  %d\n", maxLen, t, len(tagMap[t]))
	}
	return nil
}

func printTagsDetail(tags []string, tagMap map[string][]tagSpec) error {
	if tagsJSON {
		enc := json.NewEncoder(os.Stdout)
		for _, t := range tags {
			enc.Encode(struct {
				Tag   string    `json:"tag"`
				Specs []tagSpec `json:"specs"`
			}{Tag: t, Specs: tagMap[t]})
		}
		return nil
	}

	maxTagLen := 0
	for _, t := range tags {
		if len(t) > maxTagLen {
			maxTagLen = len(t)
		}
	}
	for _, t := range tags {
		for _, s := range tagMap[t] {
			fmt.Printf("%-*s  %s  %s\n", maxTagLen, t, s.ID, s.Name)
		}
	}
	return nil
}

func init() {
	tagsCmd.Flags().BoolVar(&tagsJSON, "json", false, "output as JSON")
	rootCmd.AddCommand(tagsCmd)
}
