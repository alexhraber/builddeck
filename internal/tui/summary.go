package tui

import "github.com/alexhraber/builddeck/internal/buildkite"

type BuildSummary struct {
	Total   int
	Running int
	Failed  int
	Passed  int
	Blocked int
}

func SummarizeBuilds(builds []buildkite.Build) BuildSummary {
	var s BuildSummary
	s.Total = len(builds)
	for _, b := range builds {
		switch b.State {
		case "running", "active":
			s.Running++
		case "failed", "failing":
			s.Failed++
		case "passed":
			s.Passed++
		case "blocked":
			s.Blocked++
		}
	}
	return s
}

func (s BuildSummary) FailureRate() float64 {
	if s.Total == 0 {
		return 0
	}
	completed := s.Passed + s.Failed
	if completed == 0 {
		return 0
	}
	return float64(s.Failed) / float64(completed) * 100
}
