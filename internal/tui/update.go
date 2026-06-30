package tui

import (
	"strings"
	"time"
	"unicode/utf8"

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
			if m.filterPane == centerPane && m.filterQuery != "" {
				indices := m.filteredBuildIndices()
				if len(indices) == 0 {
					m.selectedBuild = nil
					m.annotations = nil
					m.artifacts = nil
					return m, nil
				}
				if !containsIndex(indices, m.buildIndex) {
					m.buildIndex = indices[0]
				}
			}
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
	if m.searching {
		return m.handleSearchKey(msg)
	}

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
		cmd := m.refresh()
		return m, cmd

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
		m.searching = true
		m.filterPane = m.activePane
		m.searchMsg = ""
		return m, nil
	}

	m.searchMsg = ""
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "enter", "esc":
		m.searching = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "backspace", "ctrl+h":
		if m.filterQuery != "" {
			_, size := utf8.DecodeLastRuneInString(m.filterQuery)
			m.filterQuery = m.filterQuery[:len(m.filterQuery)-size]
			return m.applyFilterSelection()
		}
		return m, nil
	case "ctrl+u":
		m.filterQuery = ""
		return m.applyFilterSelection()
	}

	if msg.Type == tea.KeyRunes {
		m.filterQuery += string(msg.Runes)
		m.filterQuery = strings.TrimLeft(m.filterQuery, " ")
		return m.applyFilterSelection()
	}

	return m, nil
}

func (m Model) moveUp() (tea.Model, tea.Cmd) {
	switch m.activePane {
	case leftPane:
		if m.filterQuery != "" && m.filterPane == leftPane {
			indices := m.filteredPipelineIndices()
			if len(indices) == 0 {
				return m, nil
			}
			pos := indexPosition(indices, m.pipeIndex)
			if pos > 0 {
				m.pipeIndex = indices[pos-1]
				cmd := m.onPipelineChange()
				return m, cmd
			}
			return m, nil
		}
		if m.pipeIndex > 0 {
			m.pipeIndex--
			cmd := m.onPipelineChange()
			return m, cmd
		}
		if m.orgIndex > 0 {
			m.orgIndex--
			cmd := m.onOrgChange()
			return m, cmd
		}
	case centerPane:
		indices := m.filteredBuildIndices()
		pos := indexPosition(indices, m.buildIndex)
		if pos > 0 {
			m.buildIndex = indices[pos-1]
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
		if m.filterQuery != "" && m.filterPane == leftPane {
			indices := m.filteredPipelineIndices()
			pos := indexPosition(indices, m.pipeIndex)
			if pos >= 0 && pos < len(indices)-1 {
				m.pipeIndex = indices[pos+1]
				cmd := m.onPipelineChange()
				return m, cmd
			}
			return m, nil
		}
		if m.pipeIndex < len(m.pipelines)-1 {
			m.pipeIndex++
			cmd := m.onPipelineChange()
			return m, cmd
		}
		if m.orgIndex < len(m.orgs)-1 {
			m.orgIndex++
			cmd := m.onOrgChange()
			return m, cmd
		}
	case centerPane:
		indices := m.filteredBuildIndices()
		pos := indexPosition(indices, m.buildIndex)
		if pos >= 0 && pos < len(indices)-1 {
			m.buildIndex = indices[pos+1]
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
		if m.filterQuery != "" && m.filterPane == leftPane {
			indices := m.filteredPipelineIndices()
			if len(indices) > 0 {
				m.pipeIndex = indices[0]
				cmd := m.onPipelineChange()
				return m, cmd
			}
			return m, nil
		}
		m.orgIndex = 0
		m.pipeIndex = 0
		cmd := m.onPipelineChange()
		return m, cmd
	case centerPane:
		indices := m.filteredBuildIndices()
		if len(indices) > 0 {
			m.buildIndex = indices[0]
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
		if m.filterQuery != "" && m.filterPane == leftPane {
			indices := m.filteredPipelineIndices()
			if len(indices) > 0 {
				m.pipeIndex = indices[len(indices)-1]
				cmd := m.onPipelineChange()
				return m, cmd
			}
			return m, nil
		}
		if len(m.orgs) > 0 {
			m.orgIndex = len(m.orgs) - 1
		}
		m.pipeIndex = 0
		cmd := m.onOrgChange()
		return m, cmd
	case centerPane:
		indices := m.filteredBuildIndices()
		if len(indices) > 0 {
			m.buildIndex = indices[len(indices)-1]
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

func (m Model) applyFilterSelection() (tea.Model, tea.Cmd) {
	switch m.filterPane {
	case leftPane:
		indices := m.filteredPipelineIndices()
		if len(indices) > 0 && !containsIndex(indices, m.pipeIndex) {
			m.pipeIndex = indices[0]
			cmd := m.onPipelineChange()
			return m, cmd
		}
	case centerPane:
		indices := m.filteredBuildIndices()
		if len(indices) == 0 {
			m.selectedBuild = nil
			m.annotations = nil
			m.artifacts = nil
			return m, nil
		}
		if !containsIndex(indices, m.buildIndex) {
			m.buildIndex = indices[0]
			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
	}
	return m, nil
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

func indexPosition(indices []int, idx int) int {
	for i, candidate := range indices {
		if candidate == idx {
			return i
		}
	}
	return -1
}

func containsIndex(indices []int, idx int) bool {
	return indexPosition(indices, idx) >= 0
}
