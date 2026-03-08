package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

var (
	primeJSON    bool
	primeCompact bool
)

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output spec context for Claude Code hooks",
	RunE: func(cmd *cobra.Command, args []string) error {
		stats, err := db.GetCoverage(DB)
		if err != nil {
			return err
		}

		if primeJSON {
			return printPrimeJSON(stats)
		}
		if primeCompact {
			return printPrimeCompact(stats)
		}
		return printPrimeFull(stats)
	},
}

func printPrimeFull(stats *db.CoverageStats) error {
	fmt.Print(bsgReadme)
	fmt.Println()
	fmt.Printf("## Coverage: %d specs total, %d with code links, %d verified\n",
		stats.Total, stats.WithLinks, stats.Verified)

	if len(stats.Drifted) > 0 {
		fmt.Printf("## Drifted (%d):\n", len(stats.Drifted))
		for _, d := range stats.Drifted {
			for _, f := range d.DriftedFiles {
				fmt.Printf("  %s %q — %s modified +%dd since verify\n",
					d.Spec.ID, d.Spec.Name, f.FilePath, f.DriftDays)
			}
		}
	}

	if len(stats.ReadyToImpl) > 0 {
		fmt.Printf("## Ready to implement (%d):\n", len(stats.ReadyToImpl))
		for _, s := range stats.ReadyToImpl {
			fmt.Printf("  %s %q [accepted, no code links]\n", s.ID, s.Name)
		}
	}

	if len(stats.ReadyToVerify) > 0 {
		fmt.Printf("## Ready to verify (%d):\n", len(stats.ReadyToVerify))
		for _, s := range stats.ReadyToVerify {
			fmt.Printf("  %s %q [implemented, has code links]\n", s.ID, s.Name)
		}
	}

	for _, specType := range []model.SpecType{model.SpecTypeConstraint, model.SpecTypeInvariant} {
		specs, err := db.ListSpecs(DB, db.ListSpecsInput{Type: &specType})
		if err != nil {
			return err
		}
		if len(specs) == 0 {
			continue
		}
		fmt.Printf("## %ss (%d):\n", specType, len(specs))
		for _, s := range specs {
			fmt.Printf("  %s %q [%s]\n", s.ID, s.Name, s.Status)
			if s.Body != "" {
				fmt.Printf("    %s\n", s.Body)
			}
		}
	}

	return nil
}

func printPrimeCompact(stats *db.CoverageStats) error {
	fmt.Printf("# BSG: %d specs, %d linked, %d verified", stats.Total, stats.WithLinks, stats.Verified)
	if len(stats.Drifted) > 0 {
		fmt.Printf(", %d drifted", len(stats.Drifted))
	}
	if len(stats.ReadyToImpl) > 0 {
		fmt.Printf(", %d to implement", len(stats.ReadyToImpl))
	}
	if len(stats.ReadyToVerify) > 0 {
		fmt.Printf(", %d to verify", len(stats.ReadyToVerify))
	}
	fmt.Println()
	return nil
}

type primeJSONOutput struct {
	Total         int              `json:"total"`
	WithLinks     int              `json:"with_links"`
	Verified      int              `json:"verified"`
	Drifted       []driftedJSON    `json:"drifted,omitempty"`
	ReadyToImpl   []readyJSON      `json:"ready_to_implement,omitempty"`
	ReadyToVerify []readyJSON      `json:"ready_to_verify,omitempty"`
}

type driftedJSON struct {
	ID    string        `json:"id"`
	Name  string        `json:"name"`
	Files []driftFileJSON `json:"files"`
}

type driftFileJSON struct {
	Path      string `json:"path"`
	DriftDays int    `json:"drift_days"`
}

type readyJSON struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func printPrimeJSON(stats *db.CoverageStats) error {
	out := primeJSONOutput{
		Total:     stats.Total,
		WithLinks: stats.WithLinks,
		Verified:  stats.Verified,
	}
	for _, d := range stats.Drifted {
		dj := driftedJSON{ID: d.Spec.ID, Name: d.Spec.Name}
		for _, f := range d.DriftedFiles {
			dj.Files = append(dj.Files, driftFileJSON{Path: f.FilePath, DriftDays: f.DriftDays})
		}
		out.Drifted = append(out.Drifted, dj)
	}
	for _, s := range stats.ReadyToImpl {
		out.ReadyToImpl = append(out.ReadyToImpl, readyJSON{ID: s.ID, Name: s.Name})
	}
	for _, s := range stats.ReadyToVerify {
		out.ReadyToVerify = append(out.ReadyToVerify, readyJSON{ID: s.ID, Name: s.Name})
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func init() {
	primeCmd.Flags().BoolVar(&primeJSON, "json", false, "machine-readable JSON output")
	primeCmd.Flags().BoolVar(&primeCompact, "compact", false, "minimal output for low-context situations")
	rootCmd.AddCommand(primeCmd)
}
