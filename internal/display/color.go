package display

import "github.com/jasonmay/bsg/internal/model"

const (
	Reset   = "\033[0m"
	Dim     = "\033[2m"
	Bold    = "\033[1m"
	Green   = "\033[32m"
	Yellow  = "\033[33m"
	Blue    = "\033[34m"
	Magenta = "\033[35m"
	Cyan    = "\033[36m"
	Red     = "\033[31m"
	Gray    = "\033[90m"
	Brown   = "\033[38;5;130m"
)

func StatusColor(s model.SpecStatus) string {
	switch s {
	case model.StatusDraft:
		return Gray
	case model.StatusAccepted:
		return Blue
	case model.StatusImplemented:
		return Yellow
	case model.StatusVerified:
		return Green
	case model.StatusPaused:
		return Magenta
	case model.StatusDeprecated:
		return Red
	case model.StatusArchived:
		return Dim
	default:
		return ""
	}
}

func ColorStatus(s model.SpecStatus) string {
	return StatusColor(s) + string(s) + Reset
}
