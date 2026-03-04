package model

import (
	"fmt"
	"time"
)

type LinkType string

const (
	LinkImplements LinkType = "implements"
	LinkTests      LinkType = "tests"
	LinkDocuments  LinkType = "documents"
)

var ValidLinkTypes = []LinkType{
	LinkImplements,
	LinkTests,
	LinkDocuments,
}

func ParseLinkType(s string) (LinkType, error) {
	for _, lt := range ValidLinkTypes {
		if string(lt) == s {
			return lt, nil
		}
	}
	return "", fmt.Errorf("invalid link type %q", s)
}

type CodeLink struct {
	SpecID    string
	FilePath  string
	Symbol    string
	LinkType  LinkType
	StartLine *int
	StartCol  *int
	EndLine   *int
	EndCol    *int
	CreatedAt time.Time
}

func (l *CodeLink) HasRange() bool {
	return l.StartLine != nil
}

func (l *CodeLink) ContainsPosition(line, col int) bool {
	if !l.HasRange() {
		return true
	}
	startLine := *l.StartLine
	endLine := startLine
	if l.EndLine != nil {
		endLine = *l.EndLine
	}

	if line < startLine || line > endLine {
		return false
	}

	startCol := 0
	if l.StartCol != nil {
		startCol = *l.StartCol
	}
	endCol := 999999
	if l.EndCol != nil {
		endCol = *l.EndCol
	}

	if line == startLine && col < startCol {
		return false
	}
	if line == endLine && col > endCol {
		return false
	}
	return true
}

func (l *CodeLink) RangeString() string {
	if !l.HasRange() {
		return ""
	}
	start := *l.StartLine
	if l.EndLine != nil && *l.EndLine != start {
		return fmt.Sprintf("L%d-%d", start, *l.EndLine)
	}
	return fmt.Sprintf("L%d", start)
}
