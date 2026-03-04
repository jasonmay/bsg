package lsp

import (
	"database/sql"
	"fmt"

	"github.com/jasonmay/bsg/internal/db"
	"github.com/jasonmay/bsg/internal/model"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
)

func (s *Server) publishDiagnostics(ctx *glsp.Context, database *sql.DB, uri string) {
	filePath := URIToPath(uri)
	links, err := db.GetLinksByFile(database, filePath)
	if err != nil || len(links) == 0 {
		ctx.Notify(string(protocol.ServerTextDocumentPublishDiagnostics), protocol.PublishDiagnosticsParams{
			URI:         protocol.DocumentUri(uri),
			Diagnostics: []protocol.Diagnostic{},
		})
		return
	}

	var diags []protocol.Diagnostic
	for _, l := range links {
		spec, err := db.GetSpec(database, l.SpecID)
		if err != nil {
			continue
		}

		r := linkRange(l)
		severity := protocol.DiagnosticSeverityInformation
		if spec.Status == model.StatusVerified {
			if isDrifted(database, spec) {
				severity = protocol.DiagnosticSeverityWarning
			}
		}

		source := "bsg"
		msg := fmt.Sprintf("%s %q [%s] — %s", spec.ID, spec.Name, spec.Status, l.LinkType)
		if l.Symbol != "" {
			msg += ":" + l.Symbol
		}

		diags = append(diags, protocol.Diagnostic{
			Range:    r,
			Severity: &severity,
			Source:   &source,
			Message:  msg,
		})
	}

	ctx.Notify(string(protocol.ServerTextDocumentPublishDiagnostics), protocol.PublishDiagnosticsParams{
		URI:         protocol.DocumentUri(uri),
		Diagnostics: diags,
	})
}

func linkRange(l model.CodeLink) protocol.Range {
	if !l.HasRange() {
		return protocol.Range{
			Start: protocol.Position{Line: 0, Character: 0},
			End:   protocol.Position{Line: 0, Character: 0},
		}
	}
	startLine := protocol.UInteger(*l.StartLine - 1) // 1-based to 0-based
	endLine := startLine
	if l.EndLine != nil {
		endLine = protocol.UInteger(*l.EndLine - 1)
	}
	var startCol, endCol protocol.UInteger
	if l.StartCol != nil {
		startCol = protocol.UInteger(*l.StartCol)
	}
	if l.EndCol != nil {
		endCol = protocol.UInteger(*l.EndCol)
	}
	return protocol.Range{
		Start: protocol.Position{Line: startLine, Character: startCol},
		End:   protocol.Position{Line: endLine, Character: endCol},
	}
}

func isDrifted(database *sql.DB, spec *model.Spec) bool {
	coverage, err := db.GetCoverage(database)
	if err != nil {
		return false
	}
	for _, d := range coverage.Drifted {
		if d.Spec.ID == spec.ID {
			return true
		}
	}
	return false
}
