package tui

import (
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func TestIsTerminalState(t *testing.T) {
	tests := []struct {
		state string
		want  bool
	}{
		{"passed", true},
		{"failed", true},
		{"canceled", true},
		{"cancelled", true},
		{"skipped", true},
		{"timed_out", true},
		{"broken", true},
		{"not_run", true},
		{"running", false},
		{"scheduled", false},
		{"waiting", false},
		{"failing", false},
		{"active", false},
		{"blocked", false},
	}

	for _, tt := range tests {
		got := isTerminalState(tt.state)
		if got != tt.want {
			t.Errorf("isTerminalState(%q) = %v, want %v", tt.state, got, tt.want)
		}
	}
}

func TestModelCachingAndDebouncing(t *testing.T) {
	client := buildkite.NewClient("dummy-token")
	m := NewModel(client)

	// Create some builds
	build1 := buildkite.Build{ID: "build-1", Number: 1, State: "passed"}
	build2 := buildkite.Build{ID: "build-2", Number: 2, State: "running"}
	m.builds = []buildkite.Build{build1, build2}

	// 1. Initial selection (not cached)
	m.buildIndex = 0 // build-1 (passed)
	cmds := m.onBuildIndexChanged()
	if len(cmds) != 1 {
		t.Fatalf("Expected 1 debounce command when not cached, got %d", len(cmds))
	}
	if m.buildSelectionSeq != 1 {
		t.Errorf("Expected buildSelectionSeq to be 1, got %d", m.buildSelectionSeq)
	}

	// 2. Populate cache for build-1
	m.ensureCachesInitialized()
	m.buildDetails[build1.ID] = &build1
	m.buildAnnotations[build1.ID] = []buildkite.Annotation{}
	m.buildArtifacts[build1.ID] = []buildkite.Artifact{}

	// 3. Selection change to build-1 (cached & terminal)
	// Reset selection sequence
	m.buildSelectionSeq = 0
	cmds = m.onBuildIndexChanged()
	if len(cmds) != 0 {
		t.Errorf("Expected 0 commands for cached terminal build, got %d", len(cmds))
	}
	if m.buildSelectionSeq != 0 {
		t.Errorf("Expected buildSelectionSeq to remain 0, got %d", m.buildSelectionSeq)
	}

	// 4. Selection change to build-2 (uncached)
	m.buildIndex = 1 // build-2 (running)
	cmds = m.onBuildIndexChanged()
	if len(cmds) != 1 {
		t.Fatalf("Expected 1 debounce command, got %d", len(cmds))
	}
	if m.buildSelectionSeq != 1 {
		t.Errorf("Expected buildSelectionSeq to be 1, got %d", m.buildSelectionSeq)
	}

	// 5. Populate cache for build-2 (non-terminal)
	m.buildDetails[build2.ID] = &build2
	m.buildAnnotations[build2.ID] = []buildkite.Annotation{}
	m.buildArtifacts[build2.ID] = []buildkite.Artifact{}

	// 6. Selection change to build-2 (cached but NOT terminal)
	m.buildSelectionSeq = 0
	cmds = m.onBuildIndexChanged()
	if len(cmds) != 1 {
		t.Fatalf("Expected 1 debounce command for cached non-terminal build, got %d", len(cmds))
	}
	if m.buildSelectionSeq != 1 {
		t.Errorf("Expected buildSelectionSeq to increment to 1, got %d", m.buildSelectionSeq)
	}
}
