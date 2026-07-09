package tui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/alexhraber/builddeck/internal/config"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

var errNoRetryableJob = errors.New("no retryable job found")

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

	case agentsLoadedMsg:
		m.loadingAgents = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to load agents"
			return m, nil
		}
		m.agents = msg.agents
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

	case buildActionMsg:
		m.actionInFlight = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to " + string(msg.action)
			return m, nil
		}
		m.err = nil
		m.errMsg = ""
		switch msg.action {
		case actionRetryJob:
			m.searchMsg = "Retry queued"
		case actionRebuild:
			m.searchMsg = "Rebuild queued"
		case actionCancel:
			m.searchMsg = "Cancel requested"
		case actionUnblock:
			m.searchMsg = "Job unblocked"
		}
		return m, m.refreshBuilds()

	case artifactDownloadMsg:
		m.actionInFlight = false
		if msg.err != nil {
			m.err = msg.err
			m.errMsg = "failed to download artifact"
			return m, nil
		}
		m.searchMsg = fmt.Sprintf("Downloaded: %s", msg.filename)
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmdWithInterval(m.currentPollInterval())}
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
	if m.globalSearching {
		return m.handleGlobalSearchKey(msg)
	}

	if m.searching {
		return m.handleSearchKey(msg)
	}

	if m.showPresetPicker {
		return m.handlePresetPickerKey(msg)
	}

	if m.showHelp {
		if key.Matches(msg, keys.Help) || key.Matches(msg, keys.Quit) || msg.String() == "esc" {
			m.showHelp = false
			return m, nil
		}
		return m, nil
	}

	if m.showAgents {
		return m.handleAgentViewKey(msg)
	}

	if m.showOptions {
		return m.handleOptionsKey(msg)
	}

	if m.showStatsOverlay != "" {
		return m.handleStatsOverlayKey(msg)
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
		case msg.String() == "esc":
			m.showLogs = false
			return m, nil
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

	case key.Matches(msg, keys.RetryJob):
		switch m.activePane {
		case rightPane:
			cmd := m.retrySelectedJob()
			return m, cmd
		default:
			cmd := m.rebuildSelectedBuild()
			return m, cmd
		}

	case key.Matches(msg, keys.Cancel):
		cmd := m.cancelSelectedBuild()
		return m, cmd

	case key.Matches(msg, keys.Unblock):
		cmd := m.unblockSelectedJob()
		return m, cmd

	case key.Matches(msg, keys.OpenRepo):
		m.openRepoInBrowser()
		return m, nil

	case key.Matches(msg, keys.OpenBrowser):
		m.openInBrowser()
		return m, nil

	case key.Matches(msg, keys.OpenCommit):
		m.openCommitInBrowser()
		return m, nil

	case key.Matches(msg, keys.Download):
		cmd := m.downloadSelectedArtifact()
		return m, cmd

	case key.Matches(msg, keys.Agents):
		return m.toggleAgentView()

	case key.Matches(msg, keys.GlobalSearch):
		m.globalSearching = true
		m.globalSearchQuery = ""
		m.globalSearchResult = nil
		return m, nil

	case key.Matches(msg, keys.AgentStats):
		switch m.activePane {
		case centerPane:
			if m.selectedBuild == nil {
				m.searchMsg = "No build selected"
				return m, nil
			}
			m.showStatsOverlay = "build"
			return m, nil
		case rightPane:
			if m.selectedRightPaneJob() == nil {
				m.searchMsg = "No job selected"
				return m, nil
			}
			m.showStatsOverlay = "agent"
			return m, nil
		default:
			m.searchMsg = "Select a build or job first"
			return m, nil
		}

	case key.Matches(msg, keys.LiveMode):
		m.liveMode = !m.liveMode
		if m.liveMode {
			m.searchMsg = "Live mode ON"
		} else {
			m.searchMsg = "Live mode OFF"
		}
		return m, tickCmdWithInterval(m.currentPollInterval())

	case key.Matches(msg, keys.SavePreset):
		return m.saveCurrentFilterAsPreset()

	case key.Matches(msg, keys.LoadPreset):
		return m.showPresetPickerView()

	case key.Matches(msg, keys.Options):
		m.showOptions = true
		m.optionsCursor = 0
		return m, nil

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

func (m Model) handleGlobalSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.globalSearching = false
		m.globalSearchQuery = ""
		m.globalSearchResult = nil
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		m.globalSearching = false
		m.globalSearchResult = m.performGlobalSearch(m.globalSearchQuery)
		return m, nil
	case "backspace", "ctrl+h":
		if m.globalSearchQuery != "" {
			_, size := utf8.DecodeLastRuneInString(m.globalSearchQuery)
			m.globalSearchQuery = m.globalSearchQuery[:len(m.globalSearchQuery)-size]
			m.globalSearchResult = m.performGlobalSearch(m.globalSearchQuery)
		}
		return m, nil
	case "ctrl+u":
		m.globalSearchQuery = ""
		m.globalSearchResult = nil
		return m, nil
	}

	if msg.Type == tea.KeyRunes {
		m.globalSearchQuery += string(msg.Runes)
		m.globalSearchQuery = strings.TrimLeft(m.globalSearchQuery, " ")
		m.globalSearchResult = m.performGlobalSearch(m.globalSearchQuery)
	}

	return m, nil
}

func (m Model) handlePresetPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	presets := m.activeFilterPresets()

	switch {
	case msg.String() == "esc", key.Matches(msg, keys.Quit):
		m.showPresetPicker = false
		return m, nil
	case key.Matches(msg, keys.Up):
		if m.presetPickerIndex > 0 {
			m.presetPickerIndex--
		}
		return m, nil
	case key.Matches(msg, keys.Down):
		if m.presetPickerIndex < len(presets)-1 {
			m.presetPickerIndex++
		}
		return m, nil
	case key.Matches(msg, keys.Enter):
		if m.presetPickerIndex >= 0 && m.presetPickerIndex < len(presets) {
			p := presets[m.presetPickerIndex]
			m.filterQuery = p.Query
			switch p.Pane {
			case "pipelines":
				m.filterPane = leftPane
			case "builds":
				m.filterPane = centerPane
			case "jobs":
				m.filterPane = rightPane
			}
			m.showPresetPicker = false
			m.searchMsg = fmt.Sprintf("Preset loaded: %s", p.Name)
			return m.applyFilterSelection()
		}
		m.showPresetPicker = false
		return m, nil
	}

	return m, nil
}

func (m Model) handleAgentViewKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc", key.Matches(msg, keys.Agents):
		m.showAgents = false
		return m, nil
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Refresh):
		org := m.selectedOrg()
		if org != nil {
			m.loadingAgents = true
			return m, loadAgentsCmd(m.client, org.Slug)
		}
		return m, nil
	case key.Matches(msg, keys.Up):
		if m.rightScroll > 0 {
			m.rightScroll--
		}
		return m, nil
	case key.Matches(msg, keys.Down):
		if m.rightScroll < len(m.agents)-1 {
			m.rightScroll++
		}
		return m, nil
	case key.Matches(msg, keys.Top):
		m.rightScroll = 0
		return m, nil
	case key.Matches(msg, keys.Bottom):
		if len(m.agents) > 0 {
			m.rightScroll = len(m.agents) - 1
		}
		return m, nil
	case key.Matches(msg, keys.OpenBrowser):
		if m.rightScroll >= 0 && m.rightScroll < len(m.agents) {
			agent := m.agents[m.rightScroll]
			if agent.WebURL != "" {
				openURL(agent.WebURL)
				m.searchMsg = "Opened agent in browser"
			}
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleStatsOverlayKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case msg.String() == "esc", key.Matches(msg, keys.AgentStats), key.Matches(msg, keys.Quit):
		m.showStatsOverlay = ""
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleOptionsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	oldRefreshIndex := m.refreshRateIndex

	switch msg.String() {
	case "esc", "O":
		m.showOptions = false
		return m, nil
	case "ctrl+c":
		return m, tea.Quit
	case "up", "k":
		if m.optionsCursor > 0 {
			m.optionsCursor--
		}
		return m, nil
	case "down", "j":
		if m.optionsCursor < 3 {
			m.optionsCursor++
		}
		return m, nil
	case "left", "h":
		switch m.optionsCursor {
		case 0:
			m.themeIndex = (m.themeIndex - 1 + len(Themes)) % len(Themes)
			initStyles(Themes[m.themeIndex])
		case 1:
			m.refreshRateIndex = (m.refreshRateIndex - 1 + 6) % 6
		case 2:
			m.denseMode = !m.denseMode
		case 3:
			m.sortAsc = !m.sortAsc
		}

		var cmd tea.Cmd
		if oldRefreshIndex == 5 && m.refreshRateIndex != 5 {
			cmd = tickCmdWithInterval(m.currentPollInterval())
		}
		return m, cmd

	case "right", "l":
		switch m.optionsCursor {
		case 0:
			m.themeIndex = (m.themeIndex + 1) % len(Themes)
			initStyles(Themes[m.themeIndex])
		case 1:
			m.refreshRateIndex = (m.refreshRateIndex + 1) % 6
		case 2:
			m.denseMode = !m.denseMode
		case 3:
			m.sortAsc = !m.sortAsc
		}

		var cmd tea.Cmd
		if oldRefreshIndex == 5 && m.refreshRateIndex != 5 {
			cmd = tickCmdWithInterval(m.currentPollInterval())
		}
		return m, cmd
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
		maxScroll := len(m.visibleRunnableJobs()) - 1
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.rightScroll < maxScroll {
			m.rightScroll++
		}
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

func (m *Model) refreshBuilds() tea.Cmd {
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		return nil
	}
	m.loadingBuilds = true
	m.buildsInFlight = true
	return loadBuildsCmd(m.client, org.Slug, pipe.Slug)
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

func (m *Model) retrySelectedJob() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	build := m.actionBuild()
	if org == nil || pipe == nil || build == nil {
		m.searchMsg = "No build selected to retry"
		return nil
	}

	jobID := ""
	if m.activePane == rightPane {
		job := m.selectedRightPaneJob()
		if job == nil {
			m.searchMsg = "No job selected to retry"
			return nil
		}
		jobID = job.ID
	} else if job := firstRunnableJob(build.Jobs); job != nil {
		jobID = job.ID
	}

	m.actionInFlight = true
	m.searchMsg = "Retrying job..."
	return retryJobCmd(m.client, org.Slug, pipe.Slug, build.Number, jobID)
}

func (m *Model) rebuildSelectedBuild() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	build := m.actionBuild()
	if org == nil || pipe == nil || build == nil {
		m.searchMsg = "No build selected to rebuild"
		return nil
	}

	m.actionInFlight = true
	m.searchMsg = "Rebuilding build..."
	return rebuildBuildCmd(m.client, org.Slug, pipe.Slug, build.Number)
}

func (m *Model) cancelSelectedBuild() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	build := m.actionBuild()
	if org == nil || pipe == nil || build == nil {
		m.searchMsg = "No build selected to cancel"
		return nil
	}
	if build.State != "running" {
		m.searchMsg = "Only running builds can be canceled"
		return nil
	}

	m.actionInFlight = true
	m.searchMsg = "Canceling build..."
	return cancelBuildCmd(m.client, org.Slug, pipe.Slug, build.Number)
}

func (m *Model) unblockSelectedJob() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	build := m.actionBuild()
	if org == nil || pipe == nil || build == nil {
		m.searchMsg = "No build selected"
		return nil
	}

	// Find blocked job
	var blockedJob *buildkite.Job
	if m.activePane == rightPane {
		job := m.selectedRightPaneJob()
		if job != nil && job.State == "blocked" {
			blockedJob = job
		}
	}

	if blockedJob == nil {
		// Try to find first blocked job in build
		for i := range build.Jobs {
			if build.Jobs[i].State == "blocked" {
				blockedJob = &build.Jobs[i]
				break
			}
		}
	}

	if blockedJob == nil {
		m.searchMsg = "No blocked job found"
		return nil
	}

	m.actionInFlight = true
	m.searchMsg = "Unblocking job..."
	return unblockJobCmd(m.client, org.Slug, pipe.Slug, build.Number, blockedJob.ID)
}

func (m *Model) openInBrowser() {
	var url string

	switch m.activePane {
	case leftPane:
		pipe := m.selectedPipeline()
		if pipe != nil && pipe.WebURL != "" {
			url = pipe.WebURL
		} else if org := m.selectedOrg(); org != nil && org.WebURL != "" {
			url = org.WebURL
		}
	case centerPane:
		if b := m.selectedBuildEntry(); b != nil && b.WebURL != "" {
			url = b.WebURL
		}
	case rightPane:
		if m.selectedBuild != nil && m.selectedBuild.WebURL != "" {
			url = m.selectedBuild.WebURL
		}
	}

	if url == "" {
		m.searchMsg = "No URL available"
		return
	}

	openURL(url)
	m.searchMsg = "Opened in browser"
}

func (m *Model) openRepoInBrowser() {
	pipe := m.selectedPipeline()
	if pipe == nil || pipe.Repository == "" {
		m.searchMsg = "No repo URL available"
		return
	}
	url := gitToHTTPS(pipe.Repository)
	if url == "" {
		m.searchMsg = "No repo URL available"
		return
	}
	openURL(url)
	m.searchMsg = "Opened repo in browser"
}

func (m *Model) openCommitInBrowser() {
	pipe := m.selectedPipeline()
	if pipe == nil || pipe.Repository == "" {
		m.searchMsg = "No repo URL available"
		return
	}
	bd := m.selectedBuildEntry()
	if bd == nil || bd.Commit == "" {
		m.searchMsg = "No commit SHA available"
		return
	}
	base := gitToHTTPS(pipe.Repository)
	if base == "" {
		m.searchMsg = "Could not parse repo URL"
		return
	}
	url := base + "/commit/" + shortSHA(bd.Commit)
	openURL(url)
	m.searchMsg = "Opened commit in browser"
}

func (m *Model) downloadSelectedArtifact() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	if m.selectedBuild == nil || len(m.artifacts) == 0 {
		m.searchMsg = "No artifacts available"
		return nil
	}

	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		m.searchMsg = "No pipeline selected"
		return nil
	}

	// Pick first artifact (or could enhance with artifact selection)
	art := m.artifacts[0]
	downloadDir := "."
	if m.config != nil && m.config.DownloadDir != "" {
		downloadDir = m.config.DownloadDir
	}

	m.actionInFlight = true
	m.searchMsg = fmt.Sprintf("Downloading %s...", art.Filename)
	return downloadArtifactCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, art.JobID, art.ID, art.Filename, downloadDir)
}

func (m Model) toggleAgentView() (tea.Model, tea.Cmd) {
	if m.showAgents {
		m.showAgents = false
		return m, nil
	}

	org := m.selectedOrg()
	if org == nil {
		m.searchMsg = "No organization selected"
		return m, nil
	}

	m.showAgents = true
	m.rightScroll = 0
	if len(m.agents) == 0 {
		m.loadingAgents = true
		return m, loadAgentsCmd(m.client, org.Slug)
	}
	return m, nil
}

func (m Model) saveCurrentFilterAsPreset() (tea.Model, tea.Cmd) {
	if m.filterQuery == "" {
		m.searchMsg = "No active filter to save"
		return m, nil
	}
	if m.config == nil {
		m.searchMsg = "No config loaded"
		return m, nil
	}

	preset := config.FilterPreset{
		Name:  m.filterQuery,
		Query: m.filterQuery,
		Pane:  paneName(m.filterPane),
	}
	m.config.FilterPresets = append(m.config.FilterPresets, preset)
	if err := m.config.Save(); err != nil {
		m.searchMsg = fmt.Sprintf("Save failed: %v", err)
	} else {
		m.searchMsg = fmt.Sprintf("Preset saved: %s", preset.Name)
	}
	return m, nil
}

func (m Model) showPresetPickerView() (tea.Model, tea.Cmd) {
	presets := m.activeFilterPresets()
	if len(presets) == 0 {
		m.searchMsg = "No saved presets (use S to save current filter)"
		return m, nil
	}
	m.showPresetPicker = true
	m.presetPickerIndex = 0
	return m, nil
}

func (m Model) actionBuild() *buildkite.Build {
	if m.activePane == leftPane {
		if len(m.builds) == 0 {
			return nil
		}
		return &m.builds[0]
	}
	if m.activePane == rightPane && m.selectedBuild != nil {
		return m.selectedBuild
	}
	return m.selectedBuildEntry()
}

func (m Model) selectedRightPaneJob() *buildkite.Job {
	jobs := m.visibleRunnableJobs()
	if m.rightScroll < 0 || m.rightScroll >= len(jobs) {
		return nil
	}
	return &jobs[m.rightScroll]
}

func (m Model) visibleRunnableJobs() []buildkite.Job {
	filtered := m.filteredJobs()
	jobs := make([]buildkite.Job, 0, len(filtered))
	for _, job := range filtered {
		if job.Type == "waiter" {
			continue
		}
		jobs = append(jobs, job)
	}
	return jobs
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

func firstRunnableJob(jobs []buildkite.Job) *buildkite.Job {
	for i := range jobs {
		if jobs[i].Type != "waiter" {
			return &jobs[i]
		}
	}
	return nil
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

type buildActionMsg struct {
	action      buildAction
	buildNumber int
	jobID       string
	err         error
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

func retryJobCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, jobID string) tea.Cmd {
	return func() tea.Msg {
		if jobID == "" {
			build, err := client.GetBuild(context.Background(), orgSlug, pipelineSlug, buildNumber)
			if err != nil {
				return buildActionMsg{action: actionRetryJob, buildNumber: buildNumber, err: err}
			}
			job := firstRunnableJob(build.Jobs)
			if job == nil {
				return buildActionMsg{action: actionRetryJob, buildNumber: buildNumber, err: errNoRetryableJob}
			}
			jobID = job.ID
		}
		err := client.RetryJob(context.Background(), orgSlug, pipelineSlug, buildNumber, jobID)
		return buildActionMsg{action: actionRetryJob, buildNumber: buildNumber, jobID: jobID, err: err}
	}
}

func rebuildBuildCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int) tea.Cmd {
	return func() tea.Msg {
		_, err := client.RebuildBuild(context.Background(), orgSlug, pipelineSlug, buildNumber)
		return buildActionMsg{action: actionRebuild, buildNumber: buildNumber, err: err}
	}
}

func cancelBuildCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int) tea.Cmd {
	return func() tea.Msg {
		_, err := client.CancelBuild(context.Background(), orgSlug, pipelineSlug, buildNumber)
		return buildActionMsg{action: actionCancel, buildNumber: buildNumber, err: err}
	}
}

func unblockJobCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, jobID string) tea.Cmd {
	return func() tea.Msg {
		err := client.UnblockJob(context.Background(), orgSlug, pipelineSlug, buildNumber, jobID)
		return buildActionMsg{action: actionUnblock, buildNumber: buildNumber, jobID: jobID, err: err}
	}
}

func downloadArtifactCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, jobID, artifactID, filename, downloadDir string) tea.Cmd {
	return func() tea.Msg {
		downloadURL, err := client.DownloadArtifactURL(context.Background(), orgSlug, pipelineSlug, buildNumber, jobID, artifactID)
		if err != nil {
			return artifactDownloadMsg{filename: filename, err: err}
		}

		if downloadURL == "" {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("no download URL returned")}
		}

		// Download the file
		resp, err := http.Get(downloadURL) //nolint:gosec
		if err != nil {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("downloading: %w", err)}
		}
		defer resp.Body.Close()

		outPath := filepath.Join(downloadDir, filename)
		// Create parent dirs if needed
		if dir := filepath.Dir(outPath); dir != "." {
			os.MkdirAll(dir, 0o755)
		}

		f, err := os.Create(outPath)
		if err != nil {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("creating file: %w", err)}
		}
		defer f.Close()

		if _, err := io.Copy(f, resp.Body); err != nil {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("writing file: %w", err)}
		}

		return artifactDownloadMsg{filename: filename, err: nil}
	}
}

// gitToHTTPS converts common git remote URLs to HTTPS browser URLs.
func gitToHTTPS(repo string) string {
	switch {
	case strings.HasPrefix(repo, "https://"):
		return strings.TrimSuffix(repo, ".git")
	case strings.HasPrefix(repo, "git@"):
		repo = strings.TrimPrefix(repo, "git@")
		repo = strings.Replace(repo, ":", "/", 1)
		repo = "https://" + repo
		return strings.TrimSuffix(repo, ".git")
	case strings.HasPrefix(repo, "ssh://"):
		repo = strings.TrimPrefix(repo, "ssh://")
		if i := strings.IndexByte(repo, '/'); i >= 0 {
			repo = repo[i+1:]
		}
		repo = "https://" + repo
		return strings.TrimSuffix(repo, ".git")
	}
	return ""
}

// openURL opens a URL in the user's default browser.
func openURL(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	cmd.Start() //nolint:errcheck
}

// performGlobalSearch searches across all loaded orgs, pipelines, builds, and jobs.
func (m Model) performGlobalSearch(query string) []GlobalSearchResult {
	if query == "" {
		return nil
	}
	q := normalizeSearch(query)
	var results []GlobalSearchResult

	for _, org := range m.orgs {
		if containsQuery(q, org.Name, org.Slug) {
			results = append(results, GlobalSearchResult{
				Type:    "org",
				Label:   org.Name,
				OrgSlug: org.Slug,
				WebURL:  org.WebURL,
			})
		}
	}

	for _, pipe := range m.pipelines {
		if containsQuery(q, pipe.Name, pipe.Slug, pipe.Repository) {
			org := m.selectedOrg()
			orgSlug := ""
			if org != nil {
				orgSlug = org.Slug
			}
			results = append(results, GlobalSearchResult{
				Type:     "pipeline",
				Label:    pipe.Name,
				OrgSlug:  orgSlug,
				PipeSlug: pipe.Slug,
				WebURL:   pipe.WebURL,
			})
		}
	}

	for _, build := range m.builds {
		if buildMatches(build, q) {
			org := m.selectedOrg()
			pipe := m.selectedPipeline()
			orgSlug, pipeSlug := "", ""
			if org != nil {
				orgSlug = org.Slug
			}
			if pipe != nil {
				pipeSlug = pipe.Slug
			}
			results = append(results, GlobalSearchResult{
				Type:     "build",
				Label:    fmt.Sprintf("#%d %s %s", build.Number, build.Branch, build.State),
				OrgSlug:  orgSlug,
				PipeSlug: pipeSlug,
				BuildNum: build.Number,
				WebURL:   build.WebURL,
			})
		}
	}

	if m.selectedBuild != nil {
		for _, job := range m.selectedBuild.Jobs {
			if job.Type == "waiter" {
				continue
			}
			if jobMatches(job, q) {
				org := m.selectedOrg()
				pipe := m.selectedPipeline()
				orgSlug, pipeSlug := "", ""
				if org != nil {
					orgSlug = org.Slug
				}
				if pipe != nil {
					pipeSlug = pipe.Slug
				}
				label := job.Label
				if label == "" {
					label = job.Command
				}
				results = append(results, GlobalSearchResult{
					Type:     "job",
					Label:    fmt.Sprintf("%s [%s]", label, job.State),
					OrgSlug:  orgSlug,
					PipeSlug: pipeSlug,
					BuildNum: m.selectedBuild.Number,
					JobID:    job.ID,
					WebURL:   job.WebURL,
				})
			}
		}
	}

	// Cap results
	if len(results) > 50 {
		results = results[:50]
	}

	return results
}
