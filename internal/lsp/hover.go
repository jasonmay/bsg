package lsp

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/jasonmay/bsg/internal/db"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) handleHover(database *sql.DB, uri string, pos protocol.Position) (*protocol.Hover, error) {
	filePath := URIToPath(uri)
	line := int(pos.Line) + 1 // LSP is 0-based, bsg is 1-based
	col := int(pos.Character)

	links, err := db.GetLinksByFileAndPosition(database, filePath, line, col)
	if err != nil {
		return nil, err
	}

	if len(links) == 0 {
		return nil, nil
	}

	var parts []string
	for _, l := range links {
		spec, err := db.GetSpec(database, l.SpecID)
		if err != nil {
			continue
		}
		md := fmt.Sprintf("### %s — %s\n**Status:** %s | **Type:** %s\n\n%s",
			spec.ID, spec.Name, spec.Status, spec.Type, spec.Body)
		if len(spec.Tags) > 0 {
			md += fmt.Sprintf("\n\n**Tags:** %s", strings.Join(spec.Tags, ", "))
		}
		md += fmt.Sprintf("\n\n*Link: %s*", l.LinkType)
		if r := l.RangeString(); r != "" {
			md += fmt.Sprintf(" *(%s)*", r)
		}
		parts = append(parts, md)
	}

	return &protocol.Hover{
		Contents: protocol.MarkupContent{
			Kind:  protocol.MarkupKindMarkdown,
			Value: strings.Join(parts, "\n\n---\n\n"),
		},
	}, nil
}
