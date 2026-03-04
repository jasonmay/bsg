package lsp

import (
	"database/sql"
	"regexp"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

var bsgIDPattern = regexp.MustCompile(`bsg-[0-9a-f]{4,8}`)

func (s *Server) handleDefinition(database *sql.DB, uri string, pos protocol.Position, docs *DocumentStore) (any, error) {
	text, ok := docs.Get(uri)
	if !ok {
		return nil, nil
	}

	word := extractBSGID(text, pos)
	if word == "" {
		return nil, nil
	}

	links, err := db.GetLinksBySpec(database, word)
	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return nil, nil
	}

	var locations []protocol.LocationLink
	for _, l := range links {
		targetURI := protocol.DocumentUri(PathToURI(l.FilePath))
		targetRange := linkRange(l)
		locations = append(locations, protocol.LocationLink{
			TargetURI:            targetURI,
			TargetRange:          targetRange,
			TargetSelectionRange: targetRange,
		})
	}

	return locations, nil
}

func extractBSGID(text string, pos protocol.Position) string {
	lines := strings.Split(text, "\n")
	line := int(pos.Line)
	if line >= len(lines) {
		return ""
	}
	lineText := lines[line]
	col := int(pos.Character)

	matches := bsgIDPattern.FindAllStringIndex(lineText, -1)
	for _, m := range matches {
		if col >= m[0] && col <= m[1] {
			return lineText[m[0]:m[1]]
		}
	}
	return ""
}
