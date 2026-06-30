package tui

import (
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
	tea "github.com/charmbracelet/bubbletea"
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

func TestLogsToggle(t *testing.T) {
	client := buildkite.NewClient("dummy-token")
	m := NewModel(client)

	// 1. Pressing L with no build selected
	m1, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	model1 := m1.(Model)
	if model1.showLogs {
		t.Error("Expected showLogs to remain false when no build is selected")
	}
	if model1.searchMsg == "" {
		t.Error("Expected error search message when logs requested with no build")
	}

	// 2. Select a build and toggle logs
	build := buildkite.Build{ID: "build-1", Number: 10, State: "running"}
	m.builds = []buildkite.Build{build}
	m.buildIndex = 0
	m.selectedBuild = &build
	m.orgs = []buildkite.Organization{{Slug: "org"}}
	m.pipelines = []buildkite.Pipeline{{Slug: "pipe"}}
	m.loadingDetail = true
	m.activePane = centerPane

	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	model2 := m2.(Model)
	if !model2.showLogs {
		t.Error("Expected showLogs to be true after pressing L")
	}
	if !model2.loadingLog {
		t.Error("Expected loadingLog to be true since build has no jobs initially")
	}

	// 3. Simulate build detail loaded with jobs
	buildWithJobs := buildkite.Build{
		ID:     "build-1",
		Number: 10,
		State:  "running",
		Jobs: []buildkite.Job{
			{ID: "job-wait", Type: "waiter"},
			{ID: "job-run", Type: "script", Label: "Run tests"},
		},
	}
	m2Detail, cmd := model2.Update(buildDetailMsg{
		buildID: "build-1",
		build:   &buildWithJobs,
	})
	modelDetail := m2Detail.(Model)
	if modelDetail.logJobID != "job-run" {
		t.Errorf("Expected logJobID to be job-run, got %s", modelDetail.logJobID)
	}
	if cmd == nil {
		t.Error("Expected command to load logs after jobs are loaded")
	}

	// 4. Simulate log loaded
	mLoaded, _ := modelDetail.Update(logLoadedMsg{
		jobID: "job-run",
		log:   "test output",
	})
	modelLoaded := mLoaded.(Model)
	if modelLoaded.loadingLog {
		t.Error("Expected loadingLog to be false after log is loaded")
	}
	if modelLoaded.currentLog != "test output" {
		t.Errorf("Expected currentLog to be 'test output', got %s", modelLoaded.currentLog)
	}

	// 5. Toggle logs off
	m3, _ := modelLoaded.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	model3 := m3.(Model)
	if model3.showLogs {
		t.Error("Expected showLogs to be false after pressing L again")
	}
}

func TestLogsToggleLeftPane(t *testing.T) {
	client := buildkite.NewClient("dummy-token")
	m := NewModel(client)

	m.orgs = []buildkite.Organization{{Slug: "org"}}
	m.pipelines = []buildkite.Pipeline{{Slug: "pipe"}}
	m.activePane = leftPane

	// Press L on left pane -> builds not loaded
	m1, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("L")})
	model1 := m1.(Model)
	if model1.showLogs {
		t.Error("Expected showLogs to remain false until builds load")
	}
	if !model1.pendingLogsForLatestBuild {
		t.Error("Expected pendingLogsForLatestBuild to be true")
	}
	if cmd == nil {
		t.Error("Expected loadBuildsCmd to be triggered")
	}

	// Simulate builds loaded
	builds := []buildkite.Build{
		{ID: "build-latest", Number: 20, State: "passed", Jobs: []buildkite.Job{{ID: "job-1", Type: "script"}}},
		{ID: "build-old", Number: 19, State: "passed"},
	}
	m2, _ := model1.Update(buildsLoadedMsg{builds: builds})
	model2 := m2.(Model)
	if !model2.showLogs {
		t.Error("Expected showLogs to be true after builds load")
	}
	if model2.buildIndex != 0 || model2.selectedBuild.ID != "build-latest" {
		t.Errorf("Expected latest build to be selected, got index %d ID %s", model2.buildIndex, model2.selectedBuild.ID)
	}
	if model2.logJobID != "job-1" {
		t.Errorf("Expected logJobID to be job-1, got %s", model2.logJobID)
	}
}

