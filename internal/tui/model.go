package tui

import (
	"context"
	"time"

	"github.com/alexhraber/builddeck/internal/buildkite"
	tea "github.com/charmbracelet/bubbletea"
)

type pane int

const (
	leftPane pane = iota
	centerPane
	rightPane
)

func (p pane) next() pane {
	return pane((int(p) + 1) % 3)
}

func (p pane) prev() pane {
	return pane((int(p) + 2) % 3)
}

type Model struct {
	client *buildkite.Client

	activePane pane

	orgs          []buildkite.Organization
	orgIndex      int
	pipelines     []buildkite.Pipeline
	pipeIndex     int
	builds        []buildkite.Build
	buildIndex    int
	selectedBuild *buildkite.Build

	annotations []buildkite.Annotation
	artifacts   []buildkite.Artifact

	loadingOrgs        bool
	loadingPipes       bool
	loadingBuilds      bool
	loadingDetail      bool
	loadingAnnotations bool
	loadingArtifacts   bool

	buildsInFlight    bool
	detailInFlight    bool
	annotsInFlight    bool
	artifactsInFlight bool

	err    error
	errMsg string

	lastRefresh time.Time
	showHelp    bool
	searching   bool
	filterPane  pane
	filterQuery string
	searchMsg   string
	ready       bool
	width       int
	height      int

	leftScroll   int
	centerScroll int
	rightScroll  int
	logScroll    int

	showLogs                  bool
	loadingLog                bool
	currentLog                string
	logJobID                  string
	pendingLogsForLatestBuild bool


	// Cache maps to prevent duplicate, rate-limiting API requests
	buildDetails      map[string]*buildkite.Build
	buildAnnotations  map[string][]buildkite.Annotation
	buildArtifacts    map[string][]buildkite.Artifact
	jobLogs           map[string]string
	buildSelectionSeq int
}

func NewModel(client *buildkite.Client) Model {
	return Model{
		client:           client,
		activePane:       leftPane,
		loadingOrgs:      true,
		buildDetails:     make(map[string]*buildkite.Build),
		buildAnnotations: make(map[string][]buildkite.Annotation),
		buildArtifacts:   make(map[string][]buildkite.Artifact),
		jobLogs:          make(map[string]string),
	}
}

type orgsLoadedMsg struct {
	orgs []buildkite.Organization
	err  error
}

type pipelinesLoadedMsg struct {
	pipelines []buildkite.Pipeline
	err       error
}

type buildsLoadedMsg struct {
	builds []buildkite.Build
	err    error
}

type buildDetailMsg struct {
	buildID string
	build   *buildkite.Build
	err     error
}

type annotationsLoadedMsg struct {
	buildID     string
	annotations []buildkite.Annotation
	err         error
}

type artifactsLoadedMsg struct {
	buildID   string
	artifacts []buildkite.Artifact
	err         error
}

type buildSelectionDebounceMsg struct {
	seq int
}

type tickMsg time.Time

func isTerminalState(state string) bool {
	switch state {
	case "passed", "failed", "canceled", "cancelled", "skipped", "timed_out", "broken", "not_run":
		return true
	}
	return false
}

func debounceSelectionCmd(seq int) tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg {
		return buildSelectionDebounceMsg{seq: seq}
	})
}

func (m *Model) ensureCachesInitialized() {
	if m.buildDetails == nil {
		m.buildDetails = make(map[string]*buildkite.Build)
	}
	if m.buildAnnotations == nil {
		m.buildAnnotations = make(map[string][]buildkite.Annotation)
	}
	if m.buildArtifacts == nil {
		m.buildArtifacts = make(map[string][]buildkite.Artifact)
	}
	if m.jobLogs == nil {
		m.jobLogs = make(map[string]string)
	}
}

func (m *Model) clearCaches() {
	m.buildDetails = make(map[string]*buildkite.Build)
	m.buildAnnotations = make(map[string][]buildkite.Annotation)
	m.buildArtifacts = make(map[string][]buildkite.Artifact)
	m.jobLogs = make(map[string]string)
}

func loadOrgsCmd(client *buildkite.Client) tea.Cmd {
	return func() tea.Msg {
		orgs, err := client.ListOrganizations(context.Background())
		return orgsLoadedMsg{orgs: orgs, err: err}
	}
}

func loadPipelinesCmd(client *buildkite.Client, orgSlug string) tea.Cmd {
	return func() tea.Msg {
		pipelines, err := client.ListPipelines(context.Background(), orgSlug)
		return pipelinesLoadedMsg{pipelines: pipelines, err: err}
	}
}

func loadBuildsCmd(client *buildkite.Client, orgSlug, pipelineSlug string) tea.Cmd {
	return func() tea.Msg {
		builds, err := client.ListBuilds(context.Background(), orgSlug, pipelineSlug)
		return buildsLoadedMsg{builds: builds, err: err}
	}
}

func loadBuildDetailCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildID string, buildNumber int) tea.Cmd {
	return func() tea.Msg {
		build, err := client.GetBuild(context.Background(), orgSlug, pipelineSlug, buildNumber)
		return buildDetailMsg{buildID: buildID, build: build, err: err}
	}
}

func loadAnnotationsCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildID string, buildNumber int) tea.Cmd {
	return func() tea.Msg {
		anns, err := client.ListAnnotations(context.Background(), orgSlug, pipelineSlug, buildNumber)
		return annotationsLoadedMsg{buildID: buildID, annotations: anns, err: err}
	}
}

func loadArtifactsCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildID string, buildNumber int) tea.Cmd {
	return func() tea.Msg {
		arts, err := client.ListArtifacts(context.Background(), orgSlug, pipelineSlug, buildNumber)
		return artifactsLoadedMsg{buildID: buildID, artifacts: arts, err: err}
	}
}

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m Model) selectedOrg() *buildkite.Organization {
	if len(m.orgs) == 0 || m.orgIndex < 0 || m.orgIndex >= len(m.orgs) {
		return nil
	}
	return &m.orgs[m.orgIndex]
}

func (m Model) selectedPipeline() *buildkite.Pipeline {
	if len(m.pipelines) == 0 || m.pipeIndex < 0 || m.pipeIndex >= len(m.pipelines) {
		return nil
	}
	return &m.pipelines[m.pipeIndex]
}

func (m Model) selectedBuildEntry() *buildkite.Build {
	if len(m.builds) == 0 || m.buildIndex < 0 || m.buildIndex >= len(m.builds) {
		return nil
	}
	return &m.builds[m.buildIndex]
}

func clampIndex(idx, length int) int {
	if length <= 0 {
		return 0
	}
	if idx < 0 {
		return 0
	}
	if idx >= length {
		return length - 1
	}
	return idx
}
