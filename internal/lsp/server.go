package lsp

import (
	"database/sql"

	"github.com/tliron/commonlog"
	_ "github.com/tliron/commonlog/simple"
	"github.com/tliron/glsp"
	protocol "github.com/tliron/glsp/protocol_3_16"
	glspserver "github.com/tliron/glsp/server"
)

const serverName = "bsg"

var version = "0.1.0"

type Server struct {
	db      *sql.DB
	docs    *DocumentStore
	handler protocol.Handler
}

func NewServer(database *sql.DB) *Server {
	s := &Server{
		db:   database,
		docs: NewDocumentStore(),
	}

	s.handler = protocol.Handler{
		Initialize:  s.initialize,
		Initialized: s.initialized,
		Shutdown:    s.shutdown,

		TextDocumentDidOpen:   s.textDocumentDidOpen,
		TextDocumentDidSave:   s.textDocumentDidSave,
		TextDocumentDidClose:  s.textDocumentDidClose,
		TextDocumentDidChange: s.textDocumentDidChange,

		TextDocumentHover:      s.textDocumentHover,
		TextDocumentDefinition: s.textDocumentDefinition,
	}

	return s
}

func (s *Server) RunStdio() error {
	commonlog.Configure(0, nil) // suppress logging to avoid polluting stdio
	srv := glspserver.NewServer(&s.handler, serverName, false)
	return srv.RunStdio()
}

func (s *Server) initialize(ctx *glsp.Context, params *protocol.InitializeParams) (any, error) {
	capabilities := s.handler.CreateServerCapabilities()

	syncKind := protocol.TextDocumentSyncKindFull
	capabilities.TextDocumentSync = protocol.TextDocumentSyncOptions{
		OpenClose: boolPtr(true),
		Change:    &syncKind,
		Save: &protocol.SaveOptions{
			IncludeText: boolPtr(true),
		},
	}

	return protocol.InitializeResult{
		Capabilities: capabilities,
		ServerInfo: &protocol.InitializeResultServerInfo{
			Name:    serverName,
			Version: &version,
		},
	}, nil
}

func (s *Server) initialized(ctx *glsp.Context, params *protocol.InitializedParams) error {
	return nil
}

func (s *Server) shutdown(ctx *glsp.Context) error {
	return nil
}

func (s *Server) textDocumentDidOpen(ctx *glsp.Context, params *protocol.DidOpenTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	s.docs.Open(uri, params.TextDocument.Text)
	s.publishDiagnostics(ctx, s.db, uri)
	return nil
}

func (s *Server) textDocumentDidChange(ctx *glsp.Context, params *protocol.DidChangeTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	for _, change := range params.ContentChanges {
		if c, ok := change.(protocol.TextDocumentContentChangeEventWhole); ok {
			s.docs.Change(uri, c.Text)
		}
	}
	return nil
}

func (s *Server) textDocumentDidSave(ctx *glsp.Context, params *protocol.DidSaveTextDocumentParams) error {
	uri := string(params.TextDocument.URI)
	if params.Text != nil {
		s.docs.Change(uri, *params.Text)
	}
	s.publishDiagnostics(ctx, s.db, uri)
	return nil
}

func (s *Server) textDocumentDidClose(ctx *glsp.Context, params *protocol.DidCloseTextDocumentParams) error {
	s.docs.Close(string(params.TextDocument.URI))
	return nil
}

func (s *Server) textDocumentHover(ctx *glsp.Context, params *protocol.HoverParams) (*protocol.Hover, error) {
	return s.handleHover(s.db, string(params.TextDocument.URI), params.Position)
}

func (s *Server) textDocumentDefinition(ctx *glsp.Context, params *protocol.DefinitionParams) (any, error) {
	return s.handleDefinition(s.db, string(params.TextDocument.URI), params.Position, s.docs)
}

func boolPtr(b bool) *bool {
	return &b
}
