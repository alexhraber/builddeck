package tui

import (
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func TestSummarizeBuilds(t *testing.T) {
	tests := []struct {
		name   string
		builds []buildkite.Build
		want   BuildSummary
	}{
		{"empty", nil, BuildSummary{}},
		{
			"mixed",
			[]buildkite.Build{
				{State: "passed"},
				{State: "passed"},
				{State: "failed"},
				{State: "running"},
				{State: "blocked"},
			},
			BuildSummary{Total: 5, Running: 1, Failed: 1, Passed: 2, Blocked: 1},
		},
		{
			"all passed",
			[]buildkite.Build{{State: "passed"}, {State: "passed"}},
			BuildSummary{Total: 2, Passed: 2},
		},
		{
			"with failing",
			[]buildkite.Build{{State: "failing"}, {State: "passed"}},
			BuildSummary{Total: 2, Failed: 1, Passed: 1},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SummarizeBuilds(tt.builds)
			if got != tt.want {
				t.Errorf("SummarizeBuilds() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestBuildSummaryFailureRate(t *testing.T) {
	tests := []struct {
		name    string
		summary BuildSummary
		want    float64
	}{
		{"empty", BuildSummary{}, 0},
		{"no completed", BuildSummary{Total: 2, Running: 2}, 0},
		{"half failed", BuildSummary{Total: 4, Failed: 1, Passed: 1, Running: 2}, 50},
		{"all passed", BuildSummary{Total: 3, Passed: 3}, 0},
		{"all failed", BuildSummary{Total: 2, Failed: 2}, 100},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.summary.FailureRate()
			if got != tt.want {
				t.Errorf("FailureRate() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestClampIndex(t *testing.T) {
	tests := []struct {
		idx    int
		length int
		want   int
	}{
		{-1, 5, 0},
		{0, 5, 0},
		{3, 5, 3},
		{10, 5, 4},
		{0, 0, 0},
		{5, 0, 0},
	}
	for _, tt := range tests {
		got := clampIndex(tt.idx, tt.length)
		if got != tt.want {
			t.Errorf("clampIndex(%d, %d) = %d, want %d", tt.idx, tt.length, got, tt.want)
		}
	}
}

func TestPreserveSelection(t *testing.T) {
	builds := []buildkite.Build{
		{Number: 10},
		{Number: 20},
		{Number: 30},
	}
	tests := []struct {
		name       string
		prevNumber int
		prevIndex  int
		want       int
	}{
		{"found", 20, 0, 1},
		{"not found fallback", 99, 1, 1},
		{"not found out of range", 99, 5, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preserveSelection(builds, tt.prevNumber, tt.prevIndex)
			if got != tt.want {
				t.Errorf("preserveSelection() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestPreserveSelectionEmpty(t *testing.T) {
	got := preserveSelection(nil, 1, 0)
	if got != 0 {
		t.Errorf("preserveSelection(nil) = %d, want 0", got)
	}
}
