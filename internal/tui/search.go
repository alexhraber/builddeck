package tui

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func (m Model) filteredPipelineIndices() []int {
	query := normalizedQueryForPane(m, leftPane)
	indices := make([]int, 0, len(m.pipelines))
	for i, pipe := range m.pipelines {
		if query == "" || pipelineMatches(pipe, query) {
			indices = append(indices, i)
		}
	}
	return indices
}

func (m Model) filteredBuildIndices() []int {
	query := normalizedQueryForPane(m, centerPane)
	indices := make([]int, 0, len(m.builds))
	for i, build := range m.builds {
		if query == "" || buildMatches(build, query) {
			indices = append(indices, i)
		}
	}
	if m.sortAsc {
		for i, j := 0, len(indices)-1; i < j; i, j = i+1, j-1 {
			indices[i], indices[j] = indices[j], indices[i]
		}
	}
	return indices
}

func buildsByIndex(builds []buildkite.Build, indices []int) []buildkite.Build {
	filtered := make([]buildkite.Build, 0, len(indices))
	for _, i := range indices {
		if i >= 0 && i < len(builds) {
			filtered = append(filtered, builds[i])
		}
	}
	return filtered
}

func (m Model) filteredSteps() []buildkite.Step {
	if m.selectedBuild == nil {
		return nil
	}
	query := normalizedQueryForPane(m, rightPane)
	if query == "" {
		return m.selectedBuild.Steps
	}

	steps := make([]buildkite.Step, 0, len(m.selectedBuild.Steps))
	for _, step := range m.selectedBuild.Steps {
		if stepMatches(step, query) {
			steps = append(steps, step)
		}
	}
	return steps
}

func normalizedQueryForPane(m Model, p pane) string {
	if m.filterPane != p {
		return ""
	}
	return normalizeSearch(m.filterQuery)
}

func pipelineMatches(pipe buildkite.Pipeline, query string) bool {
	return containsQuery(query, pipe.Name, pipe.Slug, pipe.Repository, pipe.WebURL)
}

func buildMatches(build buildkite.Build, query string) bool {
	creator := ""
	if build.Creator != nil {
		creator = strings.Join([]string{build.Creator.Name, build.Creator.Email}, " ")
	}
	return containsQuery(query,
		strconv.Itoa(build.Number),
		fmt.Sprintf("#%d", build.Number),
		build.State,
		build.Branch,
		build.Commit,
		shortSHA(build.Commit),
		build.Message,
		creator,
		build.WebURL,
	)
}

func stepMatches(step buildkite.Step, query string) bool {
	agent := ""
	if step.Agent != nil {
		agent = step.Agent.Name
	}
	return containsQuery(query,
		step.State,
		step.Name,
		step.Label,
		step.Command,
		strings.Join(step.AgentQueryRules, " "),
		agent,
	)
}

func containsQuery(query string, fields ...string) bool {
	for _, field := range fields {
		if strings.Contains(normalizeSearch(field), query) {
			return true
		}
	}
	return false
}

func normalizeSearch(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
