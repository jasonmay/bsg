package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/spf13/cobra"
)

type parsedFileArg struct {
	FilePath  string
	Symbol    string
	StartLine *int
	StartCol  *int
	EndLine   *int
	EndCol    *int
}

// parseFileArg parses file path with optional range/symbol suffix:
//
//	src/main.go             -> whole file
//	src/main.go:Validate    -> symbol
//	src/main.go:10-25       -> line range
//	src/main.go:10:5-25:0   -> full range (line:col-line:col)
func parseFileArg(s string) (parsedFileArg, error) {
	result := parsedFileArg{}

	colonIdx := strings.Index(s, ":")
	if colonIdx == -1 {
		result.FilePath = s
		return result, nil
	}

	result.FilePath = s[:colonIdx]
	suffix := s[colonIdx+1:]

	if suffix == "" {
		return result, fmt.Errorf("empty suffix after ':'")
	}

	// check if first char is a digit -> range syntax
	if suffix[0] >= '0' && suffix[0] <= '9' {
		return parseRangeSuffix(result, suffix)
	}

	// non-numeric -> symbol name
	result.Symbol = suffix
	return result, nil
}

func parseRangeSuffix(result parsedFileArg, suffix string) (parsedFileArg, error) {
	// formats:
	//   10-25      -> line range
	//   10:5-25:0  -> full range
	//   10         -> single line

	dashIdx := strings.Index(suffix, "-")
	if dashIdx == -1 {
		// single line
		line, err := strconv.Atoi(suffix)
		if err != nil {
			return result, fmt.Errorf("invalid line number %q: %w", suffix, err)
		}
		result.StartLine = &line
		result.EndLine = &line
		return result, nil
	}

	startPart := suffix[:dashIdx]
	endPart := suffix[dashIdx+1:]

	startLine, startCol, err := parseLineCol(startPart)
	if err != nil {
		return result, fmt.Errorf("invalid start %q: %w", startPart, err)
	}
	endLine, endCol, err := parseLineCol(endPart)
	if err != nil {
		return result, fmt.Errorf("invalid end %q: %w", endPart, err)
	}

	result.StartLine = &startLine
	result.EndLine = &endLine
	if startCol >= 0 {
		result.StartCol = &startCol
	}
	if endCol >= 0 {
		result.EndCol = &endCol
	}
	return result, nil
}

func parseLineCol(s string) (line, col int, err error) {
	parts := strings.SplitN(s, ":", 2)
	line, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, -1, err
	}
	if len(parts) == 2 {
		col, err = strconv.Atoi(parts[1])
		if err != nil {
			return 0, -1, err
		}
		return line, col, nil
	}
	return line, -1, nil
}

var traceAs string

var traceCmd = &cobra.Command{
	Use:   "trace <spec-id> --file <path[:range|symbol]>",
	Short: "Link a spec to a code location",
	Long: `Link a spec to a file, line range, or symbol.

Examples:
  bsg trace bsg-a3f2 --file src/main.go                  # whole file
  bsg trace bsg-a3f2 --file src/main.go:Validate          # symbol
  bsg trace bsg-a3f2 --file src/main.go:10-25             # line range
  bsg trace bsg-a3f2 --file src/main.go:10:5-25:0         # full range`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		specID := args[0]
		fileArg, _ := cmd.Flags().GetString("file")

		linkType, err := model.ParseLinkType(traceAs)
		if err != nil {
			return err
		}

		parsed, err := parseFileArg(fileArg)
		if err != nil {
			return fmt.Errorf("parse file arg: %w", err)
		}

		// verify spec exists
		spec, err := db.GetSpec(DB, specID)
		if err != nil {
			return fmt.Errorf("spec %s: %w", specID, err)
		}

		err = db.CreateLink(DB, BsgDir(), db.CreateLinkInput{
			SpecID:    spec.ID,
			FilePath:  parsed.FilePath,
			Symbol:    parsed.Symbol,
			LinkType:  linkType,
			StartLine: parsed.StartLine,
			StartCol:  parsed.StartCol,
			EndLine:   parsed.EndLine,
			EndCol:    parsed.EndCol,
		})
		if err != nil {
			return err
		}

		rangeInfo := ""
		if parsed.StartLine != nil {
			if parsed.EndLine != nil && *parsed.EndLine != *parsed.StartLine {
				rangeInfo = fmt.Sprintf(" L%d-%d", *parsed.StartLine, *parsed.EndLine)
			} else {
				rangeInfo = fmt.Sprintf(" L%d", *parsed.StartLine)
			}
		}
		if parsed.Symbol != "" {
			rangeInfo = ":" + parsed.Symbol
		}

		fmt.Printf("traced %s -> %s%s (%s)\n", spec.ID, parsed.FilePath, rangeInfo, linkType)
		return nil
	},
}

var untraceCmd = &cobra.Command{
	Use:   "untrace <spec-id> <file_path>",
	Short: "Remove a code link",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		specID := args[0]
		filePath := args[1]

		if err := db.DeleteLink(DB, BsgDir(), specID, filePath); err != nil {
			return err
		}
		fmt.Printf("untraced %s -> %s\n", specID, filePath)
		return nil
	},
}

func init() {
	traceCmd.Flags().String("file", "", "file path with optional range/symbol")
	traceCmd.MarkFlagRequired("file")
	traceCmd.Flags().StringVar(&traceAs, "as", "implements", "link type: implements, tests, documents")
	rootCmd.AddCommand(traceCmd)
	rootCmd.AddCommand(untraceCmd)
}
