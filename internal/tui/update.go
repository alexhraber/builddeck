package tui

import (
	"time"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Init() tea.Cmd {
	return tea.Batch(loadOrgsCmd(m.client), tickCmd())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		return m, nil

	case orgsLoadedMsg:
		m.loadingOrgs = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to load organizations"
			return m, nil
		}
		m.orgs = msg.orgs
		m.err = nil
		m.errMsg = ""
		if len(m.orgs) > 0 {
			m.orgIndex = 0
			m.loadingPipes = true
			return m, loadPipelinesCmd(m.client, m.orgs[0].Slug)
		}
		return m, nil

	case pipelinesLoadedMsg:
		m.loadingPipes = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to load pipelines"
			return m, nil
		}
		m.pipelines = msg.pipelines
		m.err = nil
		m.errMsg = ""
		m.pipeIndex = 0
		if len(m.pipelines) > 0 {
			m.loadingBuilds = true
			m.buildsInFlight = true
			org := m.selectedOrg()
			pipe := m.selectedPipeline()
			return m, loadBuildsCmd(m.client, org.Slug, pipe.Slug)
		}
		m.resetBuildState()
		return m, nil

	case buildsLoadedMsg:
		m.loadingBuilds = false
		m.buildsInFlight = false
		m.lastRefresh = time.Now()
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to load builds"
			return m, nil
		}
		prevBuildNumber := 0
		if m.selectedBuild != nil {
			prevBuildNumber = m.selectedBuild.Number
		}
		m.builds = msg.builds
		m.err = nil
		m.errMsg = ""
		if len(m.builds) > 0 {
			m.buildIndex = preserveSelection(m.builds, prevBuildNumber, m.buildIndex)
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
		m.selectedBuild = nil
		m.annotations = nil
		m.artifacts = nil
		return m, nil

	case buildDetailMsg:
		m.loadingDetail = false
		m.detailInFlight = false
		if msg.err == nil && msg.build != nil {
			m.selectedBuild = msg.build
		}
		return m, nil

	case annotationsLoadedMsg:
		m.loadingAnnotations = false
		m.annotsInFlight = false
		if msg.err == nil {
			m.annotations = msg.annotations
		}
		return m, nil

	case artifactsLoadedMsg:
		m.loadingArtifacts = false
		m.artifactsInFlight = false
		if msg.err == nil {
			m.artifacts = msg.artifacts
		}
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmd()}
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		if org != nil && pipe != nil && !m.buildsInFlight {
			m.buildsInFlight = true
			m.loadingBuilds = false
			cmds = append(cmds, loadBuildsCmd(m.client, org.Slug, pipe.Slug))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		if key.Matches(msg, keys.Help) || key.Matches(msg, keys.Quit) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}

	switch {
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit

	case key.Matches(msg, keys.Help):
		m.showHelp = true
		return m, nil

	case key.Matches(msg, keys.Refresh):
		return m, m.refresh()

	case key.Matches(msg, keys.Tab), key.Matches(msg, keys.Right):
		m.activePane = m.activePane.next()
		return m, nil

	case key.Matches(msg, keys.ShiftTab), key.Matches(msg, keys.Left):
		m.activePane = m.activePane.prev()
		return m, nil

	case key.Matches(msg, keys.Up):
		return m.moveUp()

	case key.Matches(msg, keys.Down):
		return m.moveDown()

	case key.Matches(msg, keys.Top):
		return m.jumpTop()

	case key.Matches(msg, keys.Bottom):
		return m.jumpBottom()

	case key.Matches(msg, keys.Enter):
		cmd := m.onEnter()
		return m, cmd

	case key.Matches(msg, keys.Search):
		m.searchMsg = "search not implemented yet"
		return m, nil
	}

	m.searchMsg = ""
	return m, nil
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case leftPane:
		if m.pipeIndex > 0 {
			m.pipeIndex--
			return m, m.onPipelineChange()
		}
		if m.orgIndex > 0 {
			m.orgIndex--
			return m, m.onOrgChange()
		}
	case centerPane:
		if m.buildIndex > 0 {
			m.buildIndex--
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
	case rightPane:
		if m.rightScroll > 0 {
			m.rightScroll--
		}
	}
	return m, nil
}

func (m Model) moveDown() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case leftPane:
		if m.pipeIndex < len(m.pipelines)-1 {
			m.pipeIndex++
			return m, m.onPipelineChange()
		}
		if m.orgIndex < len(m.orgs)-1 {
			m.orgIndex++
			return m, m.onOrgChange()
		}
	case centerPane:
		if m.buildIndex < len(m.builds)-1 {
			m.buildIndex++
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
	case rightPane:
		m.rightScroll++
	}
	return m, nil
}

func (m Model) jumpTop() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case leftPane:
		m.orgIndex = 0
		m.pipeIndex = 0
		return m, m.onPipelineChange()
	case centerPane:
		if len(m.builds) > 0 {
			m.buildIndex = 0
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
	case rightPane:
		m.rightScroll = 0
	}
	return m, nil
}

func (m Model) jumpBottom() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case leftPane:
		if len(m.orgs) > 0 {
			m.orgIndex = len(m.orgs) - 1
		}
		m.pipeIndex = 0
		return m, m.onOrgChange()
	case centerPane:
		if len(m.builds) > 0 {
			m.buildIndex = len(m.builds) - 1
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
	case rightPane:
	}
	return m, nil
}

func (m *Model) onOrgChange() tea.Cmd {
	m.pipelines = nil
	m.pipeIndex = 0
	m.resetBuildState()
	m.loadingPipes = true
	m.loadingBuilds = false
	m.loadingDetail = false
	m.loadingAnnotations = false
	m.loadingArtifacts = false
	org := m.selectedOrg()
	if org != nil {
		return loadPipelinesCmd(m.client, org.Slug)
	}
	return nil
}

func (m *Model) onPipelineChange() tea.Cmd {
	m.resetBuildState()
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org != nil && pipe != nil {
		m.loadingBuilds = true
		m.buildsInFlight = true
		return loadBuildsCmd(m.client, org.Slug, pipe.Slug)
	}
	return nil
}

func (m *Model) onBuildIndexChanged() []tea.Cmd {
	var cmds []tea.Cmd
	if b := m.selectedBuildEntry(); b != nil {
		m.selectedBuild = b
		m.rightScroll = 0
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		if org != nil && pipe != nil {
			if hasNoJobs(b) && !m.detailInFlight {
				m.loadingDetail = true
				m.detailInFlight = true
				cmds = append(cmds, loadBuildDetailCmd(m.client, org.Slug, pipe.Slug, b.Number))
			}
			if !m.annotsInFlight {
				m.loadingAnnotations = true
				m.annotsInFlight = true
				m.annotations = nil
				cmds = append(cmds, loadAnnotationsCmd(m.client, org.Slug, pipe.Slug, b.Number))
			}
			if !m.artifactsInFlight {
				m.loadingArtifacts = true
				m.artifactsInFlight = true
				m.artifacts = nil
				cmds = append(cmds, loadArtifactsCmd(m.client, org.Slug, pipe.Slug, b.Number))
			}
		}
	} else {
		m.selectedBuild = nil
		m.annotations = nil
		m.artifacts = nil
	}
	return cmds
}

func (m *Model) onEnter() tea.Cmd {
	if m.activePane == leftPane {
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		if org != nil && pipe != nil {
			return m.onPipelineChange()
		}
	}
	return nil
}

func (m *Model) refresh() tea.Cmd {
	m.loadingOrgs = true
	m.resetBuildState()
	m.orgs = nil
	m.pipelines = nil
	m.err = nil
	m.errMsg = ""
	m.searchMsg = ""
	m.buildsInFlight = false
	m.detailInFlight = false
	m.annotsInFlight = false
	m.artifactsInFlight = false
	return loadOrgsCmd(m.client)
}

func (m *Model) resetBuildState() {
	m.builds = nil
	m.buildIndex = 0
	m.selectedBuild = nil
	m.annotations = nil
	m.artifacts = nil
	m.loadingBuilds = false
	m.loadingDetail = false
	m.loadingAnnotations = false
	m.loadingArtifacts = false
	m.buildsInFlight = false
	m.detailInFlight = false
	m.annotsInFlight = false
	m.artifactsInFlight = false
}

func preserveSelection(builds []buildkite.Build, prevNumber, prevIndex int) int {
	if prevNumber > 0 {
		for i, b := range builds {
			if b.Number == prevNumber {
				return i
			}
		}
	}
	return clampIndex(prevIndex, len(builds))
}

func hasNoJobs(b *buildkite.Build) bool {
	return b == nil || len(b.Jobs) == 0
}
