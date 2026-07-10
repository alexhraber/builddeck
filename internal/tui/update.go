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

var errNoRetryableStep = errors.New("no retryable step found")

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

				targetStep := firstRunnableStep(m.selectedBuild.Steps)

				var cmds []tea.Cmd
				cmds = append(cmds, m.onBuildIndexChanged()...)

				if targetStep != nil {
					m.logStepID = targetStep.ID
					m.ensureCachesInitialized()
					if cachedLog, has := m.stepLogs[targetStep.ID]; has {
						m.currentLog = cachedLog
						m.loadingLog = false
					} else {
						m.currentLog = ""
						m.loadingLog = true
						org := m.selectedOrg()
						pipe := m.selectedPipeline()
						cmds = append(cmds, loadLogCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, targetStep.ID))
					}
				} else {
					m.logStepID = ""
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
				if m.showLogs && m.logStepID == "" {
					targetStep := firstRunnableStep(msg.build.Steps)
					if targetStep != nil {
						m.logStepID = targetStep.ID
						m.logScroll = 0
						if cachedLog, has := m.stepLogs[targetStep.ID]; has {
							m.currentLog = cachedLog
							m.loadingLog = false
						} else {
							m.currentLog = ""
							m.loadingLog = true
							org := m.selectedOrg()
							pipe := m.selectedPipeline()
							return m, loadLogCmd(m.client, org.Slug, pipe.Slug, msg.build.Number, targetStep.ID)
						}
					} else {
						m.currentLog = "No steps found for this build"
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
				cmd := m.loadArtifactMetadata()
				return m, cmd
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

	case emojisLoadedMsg:
		if msg.err == nil {
			initEmojiMap(msg.emojis)
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
		m.stepLogs[msg.stepID] = msg.log
		if m.showLogs && m.logStepID == msg.stepID {
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
		case actionRetryStep:
			m.searchMsg = "Retry queued"
		case actionRebuild:
			m.searchMsg = "Rebuild queued"
		case actionCancel:
			m.searchMsg = "Cancel requested"
		case actionUnblock:
			m.searchMsg = "Step unblocked"
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

	case artifactChecksumMsg:
		m.loadingArtifacts = false
		if msg.err == nil && msg.checksum != "" {
			for i := range m.artifacts {
				if m.artifacts[i].ID == msg.artifactID {
					m.artifacts[i].Checksum = msg.checksum
					break
				}
			}
		}
	case artifactTagMsg:
		m.loadingArtifacts = false
		if msg.err == nil {
			for i := range m.artifacts {
				if m.artifacts[i].ID == msg.artifactID {
					m.artifacts[i].Tag = msg.tag
					break
				}
			}
		}
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

	if m.showArtifactPicker {
		return m.handleArtifactPickerKey(msg)
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

				targetStep := firstRunnableStep(m.selectedBuild.Steps)

				if targetStep == nil {
					m.logStepID = ""
					m.currentLog = ""
					m.loadingLog = true
					cmd := m.loadSelectedBuildDetailsForce()
					return m, cmd
				}

				m.logStepID = targetStep.ID
				m.ensureCachesInitialized()
				if cachedLog, has := m.stepLogs[targetStep.ID]; has {
					m.currentLog = cachedLog
					m.loadingLog = false
					return m, nil
				}

				m.currentLog = ""
				m.loadingLog = true
				return m, loadLogCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, targetStep.ID)
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

		var targetStep *buildkite.Step
		if m.activePane == rightPane {
			targetStep = m.selectedRightPaneStep()
		}
		if targetStep == nil {
			targetStep = firstRunnableStep(b.Steps)
		}
		if targetStep == nil {
			if m.loadingDetail || len(b.Steps) == 0 {
				m.logStepID = ""
				m.currentLog = ""
				m.loadingLog = true
				var cmd tea.Cmd
				if !m.detailInFlight && !m.loadingDetail {
					cmd = m.loadSelectedBuildDetailsForce()
				}
				return m, cmd
			}
			m.searchMsg = "No steps found for this build"
			m.showLogs = false
			return m, nil
		}
		m.logStepID = targetStep.ID
		m.ensureCachesInitialized()
		if cachedLog, has := m.stepLogs[targetStep.ID]; has {
			m.currentLog = cachedLog
			m.loadingLog = false
			return m, nil
		}
		m.currentLog = ""
		m.loadingLog = true
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		return m, loadLogCmd(m.client, org.Slug, pipe.Slug, b.Number, targetStep.ID)
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

	case key.Matches(msg, keys.RetryStep):
		switch m.activePane {
		case rightPane:
			cmd := m.retrySelectedStep()
			return m, cmd
		default:
			cmd := m.rebuildSelectedBuild()
			return m, cmd
		}

	case key.Matches(msg, keys.Cancel):
		cmd := m.cancelSelectedBuild()
		return m, cmd

	case key.Matches(msg, keys.Unblock):
		cmd := m.unblockSelectedStep()
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
		if m.selectedBuild == nil || len(m.artifacts) == 0 {
			m.searchMsg = "No artifacts available"
			return m, nil
		}
		m.showArtifactPicker = true
		m.artifactCursor = 0
		return m, nil

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
			if m.selectedRightPaneStep() == nil {
				m.searchMsg = "No step selected"
				return m, nil
			}
			m.showStatsOverlay = "agent"
			return m, nil
		default:
			m.searchMsg = "Select a build or step first"
			return m, nil
		}

	case key.Matches(msg, keys.LiveMode):
		m.liveMode = !m.liveMode
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
			case "steps":
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

func (m Model) handleArtifactPickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	n := len(m.artifacts)
	switch {
	case msg.String() == "esc", key.Matches(msg, keys.Download):
		m.showArtifactPicker = false
		return m, nil
	case key.Matches(msg, keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, keys.Up):
		if m.artifactCursor > 0 {
			m.artifactCursor--
		}
		return m, nil
	case key.Matches(msg, keys.Down):
		if m.artifactCursor < n-1 {
			m.artifactCursor++
		}
		return m, nil
	case key.Matches(msg, keys.Top):
		m.artifactCursor = 0
		return m, nil
	case key.Matches(msg, keys.Bottom):
		m.artifactCursor = n - 1
		return m, nil
	case msg.String() == "a":
		m.showArtifactPicker = false
		cmd := m.downloadAllArtifacts()
		return m, cmd
	case msg.String() == "enter":
		m.showArtifactPicker = false
		cmd := m.downloadArtifactAtIndex(m.artifactCursor)
		return m, cmd
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
		maxScroll := len(m.visibleRunnableSteps()) - 1
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
		return tea.Batch(
			loadPipelinesCmd(m.client, org.Slug),
			loadEmojisCmd(m.client, org.Slug),
		)
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

	if hasNoSteps(b) && !m.detailInFlight {
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

func (m *Model) retrySelectedStep() tea.Cmd {
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

	stepID := ""
	if m.activePane == rightPane {
		step := m.selectedRightPaneStep()
		if step == nil {
			m.searchMsg = "No step selected to retry"
			return nil
		}
		stepID = step.ID
	} else if step := firstRunnableStep(build.Steps); step != nil {
		stepID = step.ID
	}

	m.actionInFlight = true
	m.searchMsg = "Retrying step..."
	return retryStepCmd(m.client, org.Slug, pipe.Slug, build.Number, stepID)
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

func (m *Model) unblockSelectedStep() tea.Cmd {
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

	// Find blocked step
	var blockedStep *buildkite.Step
	if m.activePane == rightPane {
		step := m.selectedRightPaneStep()
		if step != nil && step.State == "blocked" {
			blockedStep = step
		}
	}

	if blockedStep == nil {
		// Try to find first blocked step in build
		for i := range build.Steps {
			if build.Steps[i].State == "blocked" {
				blockedStep = &build.Steps[i]
				break
			}
		}
	}

	if blockedStep == nil {
		m.searchMsg = "No blocked step found"
		return nil
	}

	m.actionInFlight = true
	m.searchMsg = "Unblocking step..."
	return unblockStepCmd(m.client, org.Slug, pipe.Slug, build.Number, blockedStep.ID)
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

func (m *Model) downloadArtifactAtIndex(idx int) tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	if idx < 0 || idx >= len(m.artifacts) {
		return nil
	}

	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		m.searchMsg = "No pipeline selected"
		return nil
	}

	art := m.artifacts[idx]
	downloadDir := "."
	if m.config != nil && m.config.DownloadDir != "" {
		downloadDir = m.config.DownloadDir
	}

	m.actionInFlight = true
	m.searchMsg = fmt.Sprintf("Downloading %s...", art.Filename)
	return downloadArtifactCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, art.StepID, art.ID, art.Filename, downloadDir)
}

func (m *Model) downloadAllArtifacts() tea.Cmd {
	if m.actionInFlight {
		m.searchMsg = "Action already in flight"
		return nil
	}
	if len(m.artifacts) == 0 {
		return nil
	}

	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		m.searchMsg = "No pipeline selected"
		return nil
	}

	downloadDir := "."
	if m.config != nil && m.config.DownloadDir != "" {
		downloadDir = m.config.DownloadDir
	}

	m.actionInFlight = true
	m.searchMsg = fmt.Sprintf("Downloading %d artifacts...", len(m.artifacts))

	cmds := make([]tea.Cmd, 0, len(m.artifacts))
	for i := range m.artifacts {
		art := m.artifacts[i]
		cmds = append(cmds, downloadArtifactCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, art.StepID, art.ID, art.Filename, downloadDir))
	}
	return tea.Batch(cmds...)
}

func (m *Model) loadArtifactMetadata() tea.Cmd {
	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	if org == nil || pipe == nil {
		return nil
	}

	var cmds []tea.Cmd
	for _, art := range m.artifacts {
		// Handle checksums
		if strings.HasSuffix(art.Filename, ".sha256") {
			targetName := strings.TrimSuffix(art.Filename, ".sha256")
			for i := range m.artifacts {
				if m.artifacts[i].Filename == targetName {
					cmds = append(cmds,
						loadChecksumCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, art.StepID, art.ID, m.artifacts[i].ID))
					break
				}
			}
		}
		// Handle tags
		if strings.HasSuffix(art.Filename, ".tag") {
			targetName := strings.TrimSuffix(art.Filename, ".tag")
			for i := range m.artifacts {
				if m.artifacts[i].Filename == targetName {
					cmds = append(cmds,
						loadTagCmd(m.client, org.Slug, pipe.Slug, m.selectedBuild.Number, art.StepID, art.ID, m.artifacts[i].ID))
					break
				}
			}
		}
	}
	return tea.Batch(cmds...)
}

func loadTagCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID, tagArtifactID, targetArtifactID string) tea.Cmd {
	return func() tea.Msg {
		url, err := client.DownloadArtifactURL(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID, tagArtifactID)
		if err != nil {
			return artifactTagMsg{artifactID: targetArtifactID, err: err}
		}
		if url == "" {
			return artifactTagMsg{artifactID: targetArtifactID, err: fmt.Errorf("empty download URL")}
		}

		resp, err := http.Get(url) // #nosec G107
		if err != nil {
			return artifactTagMsg{artifactID: targetArtifactID, err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return artifactTagMsg{artifactID: targetArtifactID, err: err}
		}

		return artifactTagMsg{artifactID: targetArtifactID, tag: strings.TrimSpace(string(body))}
	}
}

type artifactTagMsg struct {
	artifactID string
	tag        string
	err        error
}

func loadChecksumCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID, shaArtifactID, targetArtifactID string) tea.Cmd {
	return func() tea.Msg {
		url, err := client.DownloadArtifactURL(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID, shaArtifactID)
		if err != nil {
			return artifactChecksumMsg{artifactID: targetArtifactID, err: err}
		}
		if url == "" {
			return artifactChecksumMsg{artifactID: targetArtifactID, err: fmt.Errorf("empty download URL")}
		}

		resp, err := http.Get(url) //#nosec G107
		if err != nil {
			return artifactChecksumMsg{artifactID: targetArtifactID, err: err}
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return artifactChecksumMsg{artifactID: targetArtifactID, err: err}
		}

		parts := strings.Fields(string(body))
		if len(parts) > 0 {
			return artifactChecksumMsg{artifactID: targetArtifactID, checksum: parts[0]}
		}
		return artifactChecksumMsg{artifactID: targetArtifactID, err: fmt.Errorf("unable to parse checksum")}
	}
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

func (m Model) selectedRightPaneStep() *buildkite.Step {
	steps := m.visibleRunnableSteps()
	if m.rightScroll < 0 || m.rightScroll >= len(steps) {
		return nil
	}
	return &steps[m.rightScroll]
}

func (m Model) visibleRunnableSteps() []buildkite.Step {
	filtered := m.filteredSteps()
	steps := make([]buildkite.Step, 0, len(filtered))
	for _, step := range filtered {
		if step.Type == "waiter" {
			continue
		}
		steps = append(steps, step)
	}
	return steps
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

func hasNoSteps(b *buildkite.Build) bool {
	return b == nil || len(b.Steps) == 0
}

func firstRunnableStep(steps []buildkite.Step) *buildkite.Step {
	for i := range steps {
		if steps[i].Type == "script" {
			return &steps[i]
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
	stepID string
	log    string
	err    error
}

type buildActionMsg struct {
	action      buildAction
	buildNumber int
	stepID      string
	err         error
}

func loadLogCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID string) tea.Cmd {
	return func() tea.Msg {
		log, err := client.GetStepLog(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID)
		if err != nil {
			return logLoadedMsg{stepID: stepID, err: err}
		}
		return logLoadedMsg{stepID: stepID, log: log.Content, err: nil}
	}
}

func retryStepCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID string) tea.Cmd {
	return func() tea.Msg {
		if stepID == "" {
			build, err := client.GetBuild(context.Background(), orgSlug, pipelineSlug, buildNumber)
			if err != nil {
				return buildActionMsg{action: actionRetryStep, buildNumber: buildNumber, err: err}
			}
			step := firstRunnableStep(build.Steps)
			if step == nil {
				return buildActionMsg{action: actionRetryStep, buildNumber: buildNumber, err: errNoRetryableStep}
			}
			stepID = step.ID
		}
		err := client.RetryStep(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID)
		return buildActionMsg{action: actionRetryStep, buildNumber: buildNumber, stepID: stepID, err: err}
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

func unblockStepCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID string) tea.Cmd {
	return func() tea.Msg {
		err := client.UnblockStep(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID)
		return buildActionMsg{action: actionUnblock, buildNumber: buildNumber, stepID: stepID, err: err}
	}
}

func downloadArtifactCmd(client *buildkite.Client, orgSlug, pipelineSlug string, buildNumber int, stepID, artifactID, filename, downloadDir string) tea.Cmd {
	return func() tea.Msg {
		downloadURL, err := client.DownloadArtifactURL(context.Background(), orgSlug, pipelineSlug, buildNumber, stepID, artifactID)
		if err != nil {
			return artifactDownloadMsg{filename: filename, err: err}
		}

		if downloadURL == "" {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("no download URL returned")}
		}

		// Download the file
		resp, err := http.Get(downloadURL) //#nosec G107
		if err != nil {
			return artifactDownloadMsg{filename: filename, err: fmt.Errorf("downloading: %w", err)}
		}
		defer resp.Body.Close()

		outPath := filepath.Join(downloadDir, filename)
		// Create parent dirs if needed
		if dir := filepath.Dir(outPath); dir != "." {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				return artifactDownloadMsg{filename: filename, err: fmt.Errorf("creating download dir: %w", err)}
			}
		}

		f, err := os.Create(outPath) //#nosec G304
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
		cmd = exec.Command("open", url) //#nosec G204
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", url) //#nosec G204
	default:
		cmd = exec.Command("xdg-open", url) //#nosec G204
	}
	_ = cmd.Start()
}

// performGlobalSearch searches across all loaded orgs, pipelines, builds, and steps.
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
		for _, step := range m.selectedBuild.Steps {
			if step.Type == "waiter" {
				continue
			}
			if stepMatches(step, q) {
				org := m.selectedOrg()
				pipe := m.selectedPipeline()
				orgSlug, pipeSlug := "", ""
				if org != nil {
					orgSlug = org.Slug
				}
				if pipe != nil {
					pipeSlug = pipe.Slug
				}
				label := step.Label
				if label == "" {
					label = step.Command
				}
				results = append(results, GlobalSearchResult{
					Type:     "step",
					Label:    fmt.Sprintf("%s [%s]", label, step.State),
					OrgSlug:  orgSlug,
					PipeSlug: pipeSlug,
					BuildNum: m.selectedBuild.Number,
					StepID:   step.ID,
					WebURL:   step.WebURL,
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
