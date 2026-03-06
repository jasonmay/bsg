package model

import (
	"fmt"
	"time"
)

type SpecType string

const (
	SpecTypeBehavior   SpecType = "behavior"
	SpecTypeConstraint SpecType = "constraint"
	SpecTypeInterface  SpecType = "interface"
	SpecTypeDataShape  SpecType = "data-shape"
	SpecTypeInvariant  SpecType = "invariant"
)

var ValidSpecTypes = []SpecType{
	SpecTypeBehavior,
	SpecTypeConstraint,
	SpecTypeInterface,
	SpecTypeDataShape,
	SpecTypeInvariant,
}

func ParseSpecType(s string) (SpecType, error) {
	for _, t := range ValidSpecTypes {
		if string(t) == s {
			return t, nil
		}
	}
	return "", fmt.Errorf("invalid spec type %q (valid: behavior, constraint, interface, data-shape, invariant)", s)
}

type SpecStatus string

const (
	StatusDraft       SpecStatus = "draft"
	StatusAccepted    SpecStatus = "accepted"
	StatusImplemented SpecStatus = "implemented"
	StatusVerified    SpecStatus = "verified"
	StatusDeprecated  SpecStatus = "deprecated"
	StatusArchived    SpecStatus = "archived"
	StatusPaused      SpecStatus = "paused"
)

var validTransitions = map[SpecStatus][]SpecStatus{
	StatusDraft:       {StatusAccepted, StatusPaused, StatusDeprecated, StatusArchived},
	StatusAccepted:    {StatusImplemented, StatusPaused, StatusDeprecated, StatusArchived},
	StatusImplemented: {StatusVerified, StatusPaused, StatusDeprecated, StatusArchived},
	StatusVerified:    {StatusVerified, StatusDeprecated, StatusArchived},
	StatusPaused:      {StatusDraft, StatusAccepted, StatusImplemented, StatusDeprecated, StatusArchived},
	StatusDeprecated:  {StatusArchived},
}

func ValidateTransition(from, to SpecStatus) error {
	allowed, ok := validTransitions[from]
	if !ok {
		return fmt.Errorf("unknown status %q", from)
	}
	for _, a := range allowed {
		if a == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition %s -> %s", from, to)
}

type Spec struct {
	ID        string
	Name      string
	Type      SpecType
	Status    SpecStatus
	Body      string
	Tags      []string
	CreatedAt time.Time
	UpdatedAt time.Time
}
