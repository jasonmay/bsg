package model

import "time"

type HistoryEntry struct {
	ID        int64
	SpecID    string
	ChangedAt time.Time
	Field     string
	OldValue  string
	NewValue  string
}
