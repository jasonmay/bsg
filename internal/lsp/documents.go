package lsp

import (
	"net/url"
	"os"
	"strings"
	"sync"
)

type DocumentStore struct {
	mu   sync.RWMutex
	docs map[string]string // URI -> text
}

func NewDocumentStore() *DocumentStore {
	return &DocumentStore{docs: make(map[string]string)}
}

func (s *DocumentStore) Set(uri, text string) {
	s.mu.Lock()
	s.docs[uri] = text
	s.mu.Unlock()
}

func (s *DocumentStore) Close(uri string) {
	s.mu.Lock()
	delete(s.docs, uri)
	s.mu.Unlock()
}

func (s *DocumentStore) Get(uri string) (string, bool) {
	s.mu.RLock()
	text, ok := s.docs[uri]
	s.mu.RUnlock()
	if ok {
		return text, true
	}
	path := URIToPath(uri)
	data, err := os.ReadFile(path)
	if err != nil {
		return "", false
	}
	return string(data), true
}

func URIToPath(uri string) string {
	if strings.HasPrefix(uri, "file://") {
		u, err := url.Parse(uri)
		if err == nil {
			return u.Path
		}
	}
	return uri
}

func PathToURI(path string) string {
	return "file://" + path
}
