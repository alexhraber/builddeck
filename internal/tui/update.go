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
			org := m.selectedOrg()
			pipe := m.selectedPipeline()
			return m, loadBuildsCmd(m.client, org.Slug, pipe.Slug)
		}
		m.builds = nil
		m.buildIndex = 0
		m.selectedBuild = nil
		return m, nil

	case buildsLoadedMsg:
		m.loadingBuilds = false
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
			foundIdx := -1
			if prevBuildNumber > 0 {
				for i, b := range m.builds {
					if b.Number == prevBuildNumber {
						foundIdx = i
						break
					}
				}
			}
			if foundIdx >= 0 {
				m.buildIndex = foundIdx
			} else if m.buildIndex >= len(m.builds) {
				m.buildIndex = 0
			}
			b := m.builds[m.buildIndex]
			if hasNoJobs(&b) {
				m.loadingDetail = true
				m.selectedBuild = &b
				org := m.selectedOrg()
				pipe := m.selectedPipeline()
				return m, loadBuildDetailCmd(m.client, org.Slug, pipe.Slug, b.Number)
			}
			m.selectedBuild = &b
		} else {
			m.selectedBuild = nil
		}
		return m, nil

	case buildDetailMsg:
		m.loadingDetail = false
		if msg.err == nil && msg.build != nil {
			m.selectedBuild = msg.build
		}
		return m, nil

	case tickMsg:
		cmds := []tea.Cmd{tickCmd()}
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		if org != nil && pipe != nil && !m.loadingBuilds {
			cmds = append(cmds, loadBuildsCmd(m.client, org.Slug, pipe.Slug))
		}
		return m, tea.Batch(cmds...)

	case tea.KeyMsg:
		if m.showHelp {
			if msg.String() == "?" || msg.String() == "esc" || msg.String() == "q" {
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

		case key.Matches(msg, keys.Tab):
			m.activePane = pane((int(m.activePane) + 1) % 3)
			return m, nil

		case key.Matches(msg, keys.ShiftTab):
			m.activePane = pane((int(m.activePane) + 2) % 3)
			return m, nil

		case key.Matches(msg, keys.Up):
			cmd := m.moveUp()
			return m, cmd

		case key.Matches(msg, keys.Down):
			cmd := m.moveDown()
			return m, cmd

		case key.Matches(msg, keys.Enter):
			return m, m.onEnter()
		}
	}

	return m, nil
}

func (m *Model) moveUp() tea.Cmd {
	switch m.activePane {
	case leftPane:
		if m.pipeIndex > 0 {
			m.pipeIndex--
			return nil
		} else if m.orgIndex > 0 {
			m.orgIndex--
			return m.onOrgChange()
		}
	case centerPane:
		if m.buildIndex > 0 {
			m.buildIndex--
			return m.updateSelectedBuild()
		}
	case rightPane:
	}
	return nil
}

func (m *Model) moveDown() tea.Cmd {
	switch m.activePane {
	case leftPane:
		if m.pipeIndex < len(m.pipelines)-1 {
			m.pipeIndex++
			return nil
		} else if m.orgIndex < len(m.orgs)-1 {
			m.orgIndex++
			return m.onOrgChange()
		}
	case centerPane:
		if m.buildIndex < len(m.builds)-1 {
			m.buildIndex++
			return m.updateSelectedBuild()
		}
	case rightPane:
	}
	return nil
}

func (m *Model) onOrgChange() tea.Cmd {
	m.pipelines = nil
	m.pipeIndex = 0
	m.builds = nil
	m.buildIndex = 0
	m.selectedBuild = nil
	m.loadingPipes = true
	m.loadingBuilds = false
	m.loadingDetail = false
	org := m.selectedOrg()
	if org != nil {
		return loadPipelinesCmd(m.client, org.Slug)
	}
	return nil
}

func (m *Model) updateSelectedBuild() tea.Cmd {
	if b := m.selectedBuildEntry(); b != nil {
		m.selectedBuild = b
		m.loadingDetail = false
		if hasNoJobs(b) {
			org := m.selectedOrg()
			pipe := m.selectedPipeline()
			if org != nil && pipe != nil {
				m.loadingDetail = true
				return loadBuildDetailCmd(m.client, org.Slug, pipe.Slug, b.Number)
			}
		}
	} else {
		m.selectedBuild = nil
	}
	return nil
}

func (m *Model) onEnter() tea.Cmd {
	if m.activePane == leftPane {
		org := m.selectedOrg()
		pipe := m.selectedPipeline()
		if org != nil && pipe != nil {
			m.loadingBuilds = true
			m.builds = nil
			m.buildIndex = 0
			m.selectedBuild = nil
			return loadBuildsCmd(m.client, org.Slug, pipe.Slug)
		}
	}
	return nil
}

func (m *Model) refresh() tea.Cmd {
	m.loadingOrgs = true
	m.orgs = nil
	m.pipelines = nil
	m.builds = nil
	m.selectedBuild = nil
	m.err = nil
	m.errMsg = ""
	return loadOrgsCmd(m.client)
}

func hasNoJobs(b *buildkite.Build) bool {
	return b == nil || len(b.Jobs) == 0
}
