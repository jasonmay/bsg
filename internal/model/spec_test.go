package model

import "testing"

func TestParseSpecType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		input string
		want  SpecType
		err   bool
	}{
		{"behavior", SpecTypeBehavior, false},
		{"constraint", SpecTypeConstraint, false},
		{"interface", SpecTypeInterface, false},
		{"data-shape", SpecTypeDataShape, false},
		{"invariant", SpecTypeInvariant, false},
		{"bogus", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			got, err := ParseSpecType(tt.input)
			if (err != nil) != tt.err {
				t.Fatalf("err=%v, want err=%v", err, tt.err)
			}
			if got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateTransition(t *testing.T) {
	t.Parallel()
	tests := []struct {
		from, to SpecStatus
		err      bool
	}{
		{StatusDraft, StatusAccepted, false},
		{StatusDraft, StatusDeprecated, false},
		{StatusDraft, StatusVerified, true},
		{StatusAccepted, StatusImplemented, false},
		{StatusAccepted, StatusDraft, true},
		{StatusImplemented, StatusVerified, false},
		{StatusVerified, StatusDeprecated, false},
		{StatusVerified, StatusDraft, true},
		{StatusDraft, StatusPaused, false},
		{StatusAccepted, StatusPaused, false},
		{StatusImplemented, StatusPaused, false},
		{StatusVerified, StatusPaused, true},
		{StatusPaused, StatusDraft, false},
		{StatusPaused, StatusAccepted, false},
		{StatusPaused, StatusImplemented, false},
		{StatusPaused, StatusVerified, true},
		{StatusVerified, StatusVerified, false},
	}
	for _, tt := range tests {
		t.Run(string(tt.from)+"->"+string(tt.to), func(t *testing.T) {
			t.Parallel()
			err := ValidateTransition(tt.from, tt.to)
			if (err != nil) != tt.err {
				t.Fatalf("err=%v, want err=%v", err, tt.err)
			}
		})
	}
}
