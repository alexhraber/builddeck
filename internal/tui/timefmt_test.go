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
		{"passed", "PASS"},
		{"failed", "FAIL"},
		{"running", "RUN"},
		{"blocked", "BLCK"},
		{"scheduled", "SCHD"},
		{"canceled", "CNCL"},
		{"skipped", "SKIP"},
		{"waiting", "WAIT"},
		{"timed_out", "TMOU"},
		{"unknown", "unkn"},
		{"x", "x"},
	}
	for _, tt := range tests {
		got := stateLabel(tt.input)
		if got != tt.want {
			t.Errorf("stateLabel(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestStripHTMLTags(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"<p>Hello</p>", "Hello"},
		{"plain text", "plain text"},
		{"<b>bold</b> and <i>italic</i>", "bold and italic"},
		{"", ""},
	}
	for _, tt := range tests {
		got := stripHTMLTags(tt.input)
		if got != tt.want {
			t.Errorf("stripHTMLTags(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestFormatFileSize(t *testing.T) {
	tests := []struct {
		bytes int
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
	}
	for _, tt := range tests {
		got := formatFileSize(tt.bytes)
		if got != tt.want {
			t.Errorf("formatFileSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
