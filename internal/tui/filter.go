package tui

import (
	"fmt"
	"strings"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

func matchesQuery(query string, fields ...string) bool {
	query = strings.TrimSpace(strings.ToLower(query))
	if query == "" {
		return true
	}
	for _, field := range fields {
		if strings.Contains(strings.ToLower(field), query) {
			return true
		}
	}
	return false
}

func (m Model) hasSearch() bool {
	return strings.TrimSpace(m.searchQuery) != ""
}

func (m Model) filteredPipelineIndexes() []int {
	indexes := make([]int, 0, len(m.pipelines))
	for i, pipe := range m.pipelines {
		if matchesQuery(m.searchQuery, pipe.Name, pipe.Slug, pipe.Repository) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (m Model) filteredBuildIndexes() []int {
	indexes := make([]int, 0, len(m.builds))
	for i, build := range m.builds {
		creator := ""
		if build.Creator != nil {
			creator = build.Creator.Name + " " + build.Creator.Email
		}
		if matchesQuery(
			m.searchQuery,
			fmt.Sprintf("%d", build.Number),
			build.State,
			build.Branch,
			build.Commit,
			build.Message,
			creator,
		) {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (m Model) filteredJobs() []buildkite.Job {
	if m.selectedBuild == nil {
		return nil
	}
	jobs := make([]buildkite.Job, 0, len(m.selectedBuild.Jobs))
	for _, job := range m.selectedBuild.Jobs {
		if job.Type == "waiter" {
			continue
		}
		agent := ""
		if job.Agent != nil {
			agent = job.Agent.Name + " " + job.Agent.Hostname
		}
		if matchesQuery(m.searchQuery, job.State, job.Name, job.Label, job.Command, agent) {
			jobs = append(jobs, job)
		}
	}
	return jobs
}

func (m Model) filteredAnnotations() []buildkite.Annotation {
	annotations := make([]buildkite.Annotation, 0, len(m.annotations))
	for _, ann := range m.annotations {
		if matchesQuery(m.searchQuery, ann.Style, ann.Context, stripHTMLTags(ann.BodyHTML)) {
			annotations = append(annotations, ann)
		}
	}
	return annotations
}

func (m Model) filteredArtifacts() []buildkite.Artifact {
	artifacts := make([]buildkite.Artifact, 0, len(m.artifacts))
	for _, art := range m.artifacts {
		if matchesQuery(m.searchQuery, art.Filename, art.Dirname, art.ContentType, art.State) {
			artifacts = append(artifacts, art)
		}
	}
	return artifacts
}

func positionInIndexes(indexes []int, current int) int {
	for i, idx := range indexes {
		if idx == current {
			return i
		}
	}
	return -1
}
