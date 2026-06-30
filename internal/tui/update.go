package tui

import (
	"context"
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
			m.pendingLogsForLatestBuild = false
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
					m.pendingLogsForLatestBuild = false
					return m, nil
				}
				if !containsIndex(indices, m.buildIndex) {
					m.buildIndex = indices[0]
				}
			}

			if m.pendingLogsForLatestBuild {
				m.pendingLogsForLatestBuild = false
				m.buildIndex = 0
				m.selectedBuild = &m.builds[0]
				m.showLogs = true
				m.logScroll = 0
				
				var targetJob *buildkite.Job
				for i := range m.selectedBuild.Jobs {
					if m.selectedBuild.Jobs[i].Type != "waiter" {
						targetJob = &m.selectedBuild.Jobs[i]
						break
					}
				}
				
				var cmds []tea.Cmd
				cmds = append(cmds, m.onBuildIndexChanged()...)
				
				if targetJob != nil {
					m.logJobID = targetJob.ID
					m.ensureCachesInitialized()
					if cachedLog, has := m.jobLogs[targetJob.ID]; has {
						m.currentLog = cachedLog
						m.loadingLog = false
					} else {
						m.currentLog = ""
						m.loadingLog = true
						org := m.selectedOrg()
						pipe := m.selectedPipeline()
						cmds = append(cmds, loadLogCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, targetJob.ID))
					}
				} else {
					m.logJobID = ""
					m.currentLog = ""
					m.loadingLog = true
				}
				return m, tea.Batch(cmds...)
			}

			cmds := m.onBuildIndexChanged()
			return m, tea.Batch(cmds...)
		}
		m.pendingLogsForLatestBuild = false
		m.selectedBuild = nil
		m.annotations = nil
		m.artifacts = nil
		return m, nil

	case buildDetailMsg:
		m.loadingDetail = false
		m.detailInFlight = false
		if msg.err == nil && msg.build != nil {
			m.ensureCachesInitialized()
			m.buildDetails[msg.buildID] = msg.build
			if m.selectedBuild != nil && m.selectedBuild.ID == msg.buildID {
				m.selectedBuild = msg.build
				if m.showLogs && m.logJobID == "" {
					var targetJob *buildkite.Job
					for i := range msg.build.Jobs {
						if msg.build.Jobs[i].Type != "waiter" {
							targetJob = &msg.build.Jobs[i]
							break
						}
					}
					if targetJob != nil {
						m.logJobID = targetJob.ID
						m.logScroll = 0
						if cachedLog, has := m.jobLogs[targetJob.ID]; has {
							m.currentLog = cachedLog
							m.loadingLog = false
						} else {
							m.currentLog = ""
							m.loadingLog = true
							org := m.selectedOrg()
							pipe := m.selectedPipeline()
							return m, loadLogCmd(m.client, org.Slug, pipe.Slug, msg.build.Number, targetJob.ID)
						}
					} else {
						m.currentLog = "No jobs found for this build"
						m.loadingLog = false
					}
				}
			}
		}
		return m, nil

	case annotationsLoadedMsg:
		m.loadingAnnotations = false
		m.annotsInFlight = false
		if msg.err == nil {
			m.ensureCachesInitialized()
			m.buildAnnotations[msg.buildID] = msg.annotations
			if m.selectedBuild != nil && m.selectedBuild.ID == msg.buildID {
				m.annotations = msg.annotations
			}
		}
		return m, nil

	case artifactsLoadedMsg:
		m.loadingArtifacts = false
		m.artifactsInFlight = false
		if msg.err == nil {
			m.ensureCachesInitialized()
			m.buildArtifacts[msg.buildID] = msg.artifacts
			if m.selectedBuild != nil && m.selectedBuild.ID == msg.buildID {
				m.artifacts = msg.artifacts
			}
		}
		return m, nil

	case buildSelectionDebounceMsg:
		if msg.seq == m.buildSelectionSeq {
			cmd := m.loadSelectedBuildDetailsForce()
			return m, cmd
		}
		return m, nil

	case logLoadedMsg:
		m.loadingLog = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to load logs"
			m.currentLog = "Error loading logs: " + msg.err.Error()
			return m, nil
		}
		m.ensureCachesInitialized()
		m.jobLogs[msg.jobID] = msg.log
		if m.showLogs && m.logJobID == msg.jobID {
			m.currentLog = msg.log
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

	if key.Matches(msg, keys.Logs) {
		if m.showLogs {
			m.showLogs = false
			return m, nil
		}
		
		if m.activePane == leftPane {
			org := m.selectedOrg()
			pipe := m.selectedPipeline()
			if org == nil || pipe == nil {
				m.searchMsg = "No pipeline selected"
				return m, nil
			}

			if len(m.builds) > 0 {
				m.showLogs = true
				m.logScroll = 0
				m.buildIndex = 0
				m.selectedBuild = &m.builds[0]
				
				var targetJob *buildkite.Job
				for i := range m.selectedBuild.Jobs {
					if m.selectedBuild.Jobs[i].Type != "waiter" {
						targetJob = &m.selectedBuild.Jobs[i]
						break
					}
				}
				
				if targetJob == nil {
					m.logJobID = ""
					m.currentLog = ""
					m.loadingLog = true
					cmd := m.loadSelectedBuildDetailsForce()
					return m, cmd
				}
				
				m.logJobID = targetJob.ID
				m.ensureCachesInitialized()
				if cachedLog, has := m.jobLogs[targetJob.ID]; has {
					m.currentLog = cachedLog
					m.loadingLog = false
					return m, nil
				}
				
				m.currentLog = ""
				m.loadingLog = true
				return m, loadLogCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, targetJob.ID)
			}

			m.pendingLogsForLatestBuild = true
			if !m.buildsInFlight && !m.loadingBuilds {
				m.loadingBuilds = true
				m.buildsInFlight = true
				return m, loadBuildsCmd(m.client, org.Slug, pipe.Slug)
			}
			return m, nil
		}

		b := m.selectedBuildEntry()
		if b == nil {
			m.searchMsg = "No build selected to show logs"
			return m, nil
		}
		m.showLogs = true
		m.logScroll = 0
		m.selectedBuild = b
		var targetJob *buildkite.Job
		for i := range b.Jobs {
			if b.Jobs[i].Type != "waiter" {
				targetJob = &b.Jobs[i]
				break
			}
		}
		if targetJob == nil {
			if m.loadingDetail || len(b.Jobs) == 0 {
				m.logJobID = ""
				m.currentLog = ""
				m.loadingLog = true
				var cmd tea.Cmd
				if !m.detailInFlight && !m.loadingDetail {
					cmd = m.loadSelectedBuildDetailsForce()
				}
				return m, cmd
			}
			m.searchMsg = "No jobs found for this build"
			m.showLogs = false
			return m, nil
		}
		m.logJobID = targetJob.ID
		m.ensureCachesInitialized()
		if cachedLog, has := m.jobLogs[targetJob.ID]; has {
			m.currentLog = cachedLog
			m.loadingLog = false
			return m, nil
		}
		m.currentLog = ""
		m.loadingLog = true
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		return m, loadLogCmd(m.client, org.Slug, pipe.Slug, b.Number, targetJob.ID)
	}

	if m.showLogs {
		switch {
		case key.Matches(msg, keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, keys.Up):
			if m.logScroll > 0 {
				m.logScroll--
			}
			return m, nil
		case key.Matches(msg, keys.Down):
			lines := strings.Split(m.currentLog, "\n")
			maxScroll := len(lines) - (m.height - 2)
			if maxScroll < 0 {
				maxScroll = 0
			}
			if m.logScroll < maxScroll {
				m.logScroll++
			}
			return m, nil
		case key.Matches(msg, keys.Top):
			m.logScroll = 0
			return m, nil
		case key.Matches(msg, keys.Bottom):
			lines := strings.Split(m.currentLog, "\n")
			maxScroll := len(lines) - (m.height - 2)
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.logScroll = maxScroll
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
	m.ensureCachesInitialized()

	b := m.selectedBuildEntry()
	if b == nil {
		m.selectedBuild = nil
		m.annotations = nil
		m.artifacts = nil
		return nil
	}

	m.rightScroll = 0

	// Check if we have fully cached detail, annotations, and artifacts
	cachedDetail, hasDetail := m.buildDetails[b.ID]
	cachedAnnots, hasAnnots := m.buildAnnotations[b.ID]
	cachedArts, hasArts := m.buildArtifacts[b.ID]

	if hasDetail && hasAnnots && hasArts {
		m.selectedBuild = cachedDetail
		m.annotations = cachedAnnots
		m.artifacts = cachedArts

		// If the build is in a terminal state, we do not need to fetch anything from the API!
		if isTerminalState(cachedDetail.State) {
			return nil
		}
	} else {
		// Not fully cached, display the list build entry and clear details
		m.selectedBuild = b
		m.annotations = nil
		m.artifacts = nil
	}

	// Increment selection sequence for debouncing selection change API calls
	m.buildSelectionSeq++
	seq := m.buildSelectionSeq

	return []tea.Cmd{
		debounceSelectionCmd(seq),
	}
}

func (m *Model) loadSelectedBuildDetailsForce() tea.Cmd {
	b := m.selectedBuild
	if b == nil {
		return nil
	}
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		return nil
	}

	var cmds []tea.Cmd

	if hasNoJobs(b) && !m.detailInFlight {
		m.loadingDetail = true
		m.detailInFlight = true
		cmds = append(cmds, loadBuildDetailCmd(m.client, org.Slug, pipe.Slug, b.ID, b.Number))
	}
	if !m.annotsInFlight {
		m.loadingAnnotations = true
		m.annotsInFlight = true
		cmds = append(cmds, loadAnnotationsCmd(m.client, org.Slug, pipe.Slug, b.ID, b.Number))
	}
	if !m.artifactsInFlight {
		m.loadingArtifacts = true
		m.artifactsInFlight = true
		cmds = append(cmds, loadArtifactsCmd(m.client, org.Slug, pipe.Slug, b.ID, b.Number))
	}

	return tea.Batch(cmds...)
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
	m.clearCaches() // Clear API caches on manual full refresh
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

type logLoadedMsg struct {
	jobID string
	log   string
	err   error
}

func loadLogCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, jobID string) tea.Cmd {
	return func() tea.Msg {
		log, err := client.GetJobLog(context.Background(), orgSlug, pipelineSlug, buildNumber, jobID)
		if err != nil {
			return logLoadedMsg{jobID: jobID, err: err}
		}
		return logLoadedMsg{jobID: jobID, log: log.Content, err: nil}
	}
}
