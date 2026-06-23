package tui

import (
	"testing"
)

func TestFormatTime(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", "—"},
		{"2024-01-15T10:30:00Z", "Jan 15 10:30"},
		{"invalid", "invalid"},
	}
	for _, tt := range tests {
		got := FormatTime(tt.input)
		if got != tt.want {
			t.Errorf("FormatTime(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		name   string
		start  string
		finish string
		want   string
	}{
		{"empty start", "", "2024-01-15T10:31:00Z", "—"},
		{"invalid start", "invalid", "2024-01-15T10:31:00Z", "—"},
		{"invalid finish", "2024-01-15T10:30:00Z", "invalid", "—"},
		{"seconds", "2024-01-15T10:30:00Z", "2024-01-15T10:30:45Z", "45s"},
		{"minutes and seconds", "2024-01-15T10:30:00Z", "2024-01-15T10:32:30Z", "2m30s"},
		{"hours and minutes", "2024-01-15T10:30:00Z", "2024-01-15T12:45:00Z", "2h15m"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatDuration(tt.start, tt.finish)
			if got != tt.want {
				t.Errorf("FormatDuration(%q, %q) = %q, want %q", tt.start, tt.finish, got, tt.want)
			}
		})
	}
}

func TestShortSHA(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"abc1234def5678", "abc1234"},
		{"short", "short"},
		{"", ""},
	}
	for _, tt := range tests {
		got := shortSHA(tt.input)
		if got != tt.want {
			t.Errorf("shortSHA(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStateLabel(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"passed", "PASSED"},
		{"failed", "FAILED"},
		{"running", "RUNNING"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		got := stateLabel(tt.input)
		if got != tt.want {
			t.Errorf("stateLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
