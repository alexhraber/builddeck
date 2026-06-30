package tui

import (
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func TestFilteredBuildIndices(t *testing.T) {
	m := Model{
		filterPane:  centerPane,
		filterQuery: "main",
		builds: []buildkite.Build{
			{Number: 1, Branch: "main", State: "passed"},
			{Number: 2, Branch: "release", State: "failed"},
			{Number: 3, Branch: "feature", Message: "Update main dashboard"},
		},
	}

	got := m.filteredBuildIndices()
	want := []int{0, 2}
	if len(got) != len(want) {
		t.Fatalf("filteredBuildIndices() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("filteredBuildIndices() = %v, want %v", got, want)
		}
	}
}

func TestBuildMatchesCreatorAndNumber(t *testing.T) {
	build := buildkite.Build{
		Number: 42,
		State:  "running",
		Creator: &buildkite.Creator{
			Name:  "Release Captain",
			Email: "release@example.com",
		},
	}

	for _, query := range []string{"42", "#42", "captain", "release@example.com", "RUN"} {
		if !buildMatches(build, normalizeSearch(query)) {
			t.Fatalf("buildMatches() did not match %q", query)
		}
	}
}

func TestFilteredPipelineIndices(t *testing.T) {
	m := Model{
		filterPane:  leftPane,
		filterQuery: "deploy",
		pipelines: []buildkite.Pipeline{
			{Name: "API", Slug: "api"},
			{Name: "Deploy Production", Slug: "deploy-prod"},
		},
	}

	got := m.filteredPipelineIndices()
	if len(got) != 1 || got[0] != 1 {
		t.Fatalf("filteredPipelineIndices() = %v, want [1]", got)
	}
}

func TestFilteredJobs(t *testing.T) {
	m := Model{
		filterPane:  rightPane,
		filterQuery: "linux",
		selectedBuild: &buildkite.Build{
			Jobs: []buildkite.Job{
				{Label: "Unit tests", AgentQueryRules: []string{"queue=linux"}},
				{Label: "Lint", AgentQueryRules: []string{"queue=mac"}},
			},
		},
	}

	got := m.filteredJobs()
	if len(got) != 1 || got[0].Label != "Unit tests" {
		t.Fatalf("filteredJobs() = %+v, want Unit tests only", got)
	}
}
