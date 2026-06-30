package tui

import (
	"reflect"
	"testing"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func TestMatchesQuery(t *testing.T) {
	if !matchesQuery("main", "release", "Main branch") {
		t.Fatal("expected case-insensitive match")
	}
	if matchesQuery("deploy", "test", "build") {
		t.Fatal("expected query not to match unrelated fields")
	}
	if !matchesQuery("  ", "anything") {
		t.Fatal("expected blank query to match")
	}
}

func TestFilteredPipelineIndexes(t *testing.T) {
	m := Model{
		searchQuery: "api",
		pipelines: []buildkite.Pipeline{
			{Name: "Web", Slug: "web"},
			{Name: "API", Slug: "api"},
			{Name: "Deploy", Repository: "git@example.com:service-api.git"},
		},
	}

	got := m.filteredPipelineIndexes()
	want := []int{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filteredPipelineIndexes() = %v, want %v", got, want)
	}
}

func TestFilteredBuildIndexes(t *testing.T) {
	m := Model{
		searchQuery: "alice",
		builds: []buildkite.Build{
			{Number: 1, Branch: "main", Creator: &buildkite.Creator{Name: "Bob"}},
			{Number: 2, Branch: "release", Creator: &buildkite.Creator{Name: "Alice"}},
			{Number: 3, Message: "Fix Alice deploy"},
		},
	}

	got := m.filteredBuildIndexes()
	want := []int{1, 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("filteredBuildIndexes() = %v, want %v", got, want)
	}
}

func TestFilteredJobsSkipsWaiters(t *testing.T) {
	m := Model{
		searchQuery: "test",
		selectedBuild: &buildkite.Build{Jobs: []buildkite.Job{
			{Type: "script", Label: "Build"},
			{Type: "waiter", Label: "Wait for test"},
			{Type: "script", Label: "Test"},
		}},
	}

	got := m.filteredJobs()
	if len(got) != 1 || got[0].Label != "Test" {
		t.Fatalf("filteredJobs() = %+v, want only Test script job", got)
	}
}

func TestPositionInIndexes(t *testing.T) {
	indexes := []int{2, 4, 8}
	if got := positionInIndexes(indexes, 4); got != 1 {
		t.Fatalf("positionInIndexes() = %d, want 1", got)
	}
	if got := positionInIndexes(indexes, 5); got != -1 {
		t.Fatalf("positionInIndexes() = %d, want -1", got)
	}
}
