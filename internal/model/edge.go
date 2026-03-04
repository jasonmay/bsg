package model

import (
	"fmt"
	"time"
)

type Relation string

const (
	RelDependsOn    Relation = "depends_on"
	RelRefines      Relation = "refines"
	RelConflictsWith Relation = "conflicts_with"
	RelImplements   Relation = "implements"
	RelSupersedes   Relation = "supersedes"
)

var ValidRelations = []Relation{
	RelDependsOn,
	RelRefines,
	RelConflictsWith,
	RelImplements,
	RelSupersedes,
}

func ParseRelation(s string) (Relation, error) {
	for _, r := range ValidRelations {
		if string(r) == s {
			return r, nil
		}
	}
	return "", fmt.Errorf("invalid relation %q", s)
}

type Edge struct {
	FromID    string
	ToID      string
	Relation  Relation
	CreatedAt time.Time
}
