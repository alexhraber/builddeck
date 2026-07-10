package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/alexhraber/builddeck/internal/buildkite"
)

const minTermWidth = 60
const minTermHeight = 15

func (m Model) View() string {
	if !m.ready {
		return "Initializing builddeck..."
	}

	if m.width < minTermWidth || m.height < minTermHeight {
		return m.compactView()
	}

	if m.showHelp {
		return m.helpView()
	}

	if m.showLogs {
		return m.logsView()
	}

	if m.showAgents {
		return m.agentView()
	}

	headerHeight := 1
	statusHeight := 1
	mainHeight := m.height - headerHeight - statusHeight
	if mainHeight < 5 {
		mainHeight = 5
	}

	leftW := m.width/4 - 8
	centerW := m.width/3 + 14
	rightW := m.width - leftW - centerW

	header := m.headerView(m.width)
	left := m.leftPaneView(leftW, mainHeight)
	center := m.centerPaneView(centerW, mainHeight)
	right := m.rightPaneView(rightW, mainHeight)
	status := m.statusBarView(m.width)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	mainView := lipgloss.JoinVertical(lipgloss.Left, header, panes, status)

	// Overlay global search results if present
	if m.globalSearching || len(m.globalSearchResult) > 0 {
		return m.globalSearchOverlay(mainView)
	}

	// Overlay preset picker if present
	if m.showPresetPicker {
		return m.presetPickerOverlay(mainView)
	}

	// Overlay agent/build stats if present
	if m.showStatsOverlay != "" {
		return m.statsOverlay(mainView)
	}

	// Overlay options if present
	if m.showOptions {
		return m.optionsOverlay(mainView)
	}

	return mainView
}

func (m Model) compactView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("builddeck"))
	b.WriteString("\n")

	org := m.selectedOrg()
	pipe := m.selectedPipeline()
	bd := m.selectedBuild

	if org != nil {
		b.WriteString(fmt.Sprintf("Org: %s", org.Slug))
	}
	if pipe != nil {
		b.WriteString(fmt.Sprintf("  Pipe: %s", pipe.Slug))
	}
	if bd != nil {
		b.WriteString(fmt.Sprintf("  #%d %s", bd.Number, bd.State))
	}
	b.WriteString("\n")

	if m.loadingOrgs || m.loadingBuilds {
		b.WriteString(loadingStyle.Render("Loading..."))
	} else if m.err != nil {
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", m.errMsg)))
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("Terminal too small (%dx%d). Need %dx%d.", m.width, m.height, minTermWidth, minTermHeight)))

	return b.String()
}

func (m Model) headerView(w int) string {
	var parts []string

	parts = append(parts, titleStyle.Render("builddeck"))

	if org := m.selectedOrg(); org != nil {
		parts = append(parts, subtitleStyle.Render(org.Slug))
		if pipe := m.selectedPipeline(); pipe != nil {
			parts = append(parts, subtitleStyle.Render("/"))
			parts = append(parts, subtitleStyle.Render(pipe.Slug))
			if bd := m.selectedBuild; bd != nil {
				parts = append(parts, subtitleStyle.Render(fmt.Sprintf("#%d", bd.Number)))
			}
		}
	}

	if m.loadingOrgs || m.loadingBuilds || m.loadingDetail {
		parts = append(parts, loadingStyle.Render("⟳ loading"))
	}

	if !m.lastRefresh.IsZero() {
		parts = append(parts, dimStyle.Render(m.lastRefresh.Format("15:04:05")))
	}

	line := strings.Join(parts, " ")
	return lipgloss.NewStyle().Width(w).Render(line)
}

func (m Model) leftPaneView(w, h int) string {
	style := borderStyle
	if m.activePane == leftPane {
		style = activeBorderStyle
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Organizations"))
	b.WriteString("\n")

	if m.loadingOrgs {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if len(m.orgs) == 0 {
		b.WriteString(dimStyle.Render("No organizations"))
		b.WriteString("\n")
	} else {
		orgIcon := loadPipelineEmoji("github")
		for i, org := range m.orgs {
			cursor := "  "
			if i == m.orgIndex {
				cursor = "▶ "
			}
			orgName := renderEmoji(org.Name)
			if len(orgName) > w-10 {
				orgName = orgName[:w-10]
			}
			if w := lipgloss.Width(orgIcon); w < 4 {
				orgIcon += strings.Repeat(" ", 4-w)
			}
			line := fmt.Sprintf("%s%s%s", cursor, orgIcon, orgName)
			if i == m.orgIndex {
				b.WriteString(selectedItemStyle.Render(line))
			} else {
				b.WriteString(normalItemStyle.Render(line))
			}
			b.WriteString("\n")
		}
	}

	if !m.denseMode {
		b.WriteString("\n")
	}
	b.WriteString(titleStyle.Render("Pipelines"))
	if query := normalizedQueryForPane(m, leftPane); query != "" {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" /%s", query)))
	}
	b.WriteString("\n")

	pipelineIndices := m.filteredPipelineIndices()
	if m.loadingPipes {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if m.selectedOrg() == nil {
		b.WriteString(dimStyle.Render("Select an organization"))
		b.WriteString("\n")
	} else if len(m.pipelines) == 0 {
		b.WriteString(dimStyle.Render("No pipelines"))
		b.WriteString("\n")
	} else if len(pipelineIndices) == 0 {
		b.WriteString(dimStyle.Render("No matching pipelines"))
		b.WriteString("\n")
	} else {
		for _, i := range pipelineIndices {
			pipe := m.pipelines[i]
			cursor := "  "
			if i == m.pipeIndex {
				cursor = "▶ "
			}
			pipeName := renderEmoji(pipe.Name)
			if len(pipeName) > w-10 {
				pipeName = pipeName[:w-10]
			}
			emojiName := pipe.Emoji
			if emojiName == "" {
				emojiName = "buildkite"
			}
			badge := loadPipelineEmoji(emojiName)
			if w := lipgloss.Width(badge); w < 4 {
				badge += strings.Repeat(" ", 4-w)
			}
			name := badge + pipeName
			if i == m.pipeIndex {
				b.WriteString(selectedItemStyle.Render(cursor + name))
			} else {
				b.WriteString(normalItemStyle.Render(cursor + name))
			}
			b.WriteString("\n")
		}
	}

	return style.Width(w).Height(h).Render(b.String())
}

func (m Model) centerPaneView(w, h int) string {
	style := borderStyle
	if m.activePane == centerPane {
		style = activeBorderStyle
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Builds"))

	if m.selectedOrg() != nil && m.selectedPipeline() != nil {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" — %s/%s", m.selectedOrg().Slug, m.selectedPipeline().Slug)))
	}
	if query := normalizedQueryForPane(m, centerPane); query != "" {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" /%s", query)))
	}
	b.WriteString("\n")

	buildIndices := m.filteredBuildIndices()
	if len(buildIndices) > 0 {
		summary := SummarizeBuilds(buildsByIndex(m.builds, buildIndices))
		b.WriteString(m.renderBuildSummary(summary))
		if !m.denseMode {
			b.WriteString("\n")
		}
	}

	if m.loadingBuilds {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if m.selectedPipeline() == nil {
		b.WriteString(dimStyle.Render("Select a pipeline"))
		b.WriteString("\n")
	} else if len(m.builds) == 0 {
		b.WriteString(dimStyle.Render("No builds"))
		b.WriteString("\n")
	} else if len(buildIndices) == 0 {
		b.WriteString(dimStyle.Render("No matching builds"))
		b.WriteString("\n")
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-8s %-16s %-9s %-9s %-15s %s", "BUILD", "BRANCH", "COMMIT", "STATE", "CREATOR", "DURATION")) + "\n")
		for _, i := range buildIndices {
			build := m.builds[i]
			b.WriteString(m.renderBuildRow(i, build))
		}
	}

	return style.Width(w).Height(h).Render(b.String())
}

func (m Model) renderBuildSummary(s BuildSummary) string {
	var parts []string
	parts = append(parts, fmt.Sprintf("%d builds", s.Total))
	if s.Running > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Bold(true).Render(fmt.Sprintf("%d running", s.Running)))
	}
	if s.Failed > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true).Render(fmt.Sprintf("%d failed", s.Failed)))
	}
	if s.Passed > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Render(fmt.Sprintf("%d passed", s.Passed)))
	}
	if s.Blocked > 0 {
		parts = append(parts, lipgloss.NewStyle().Foreground(lipgloss.Color("199")).Render(fmt.Sprintf("%d blocked", s.Blocked)))
	}
	if rate := s.FailureRate(); rate > 0 {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("(%.0f%% fail)", rate)))
	}
	return dimStyle.Render("[") + strings.Join(parts, dimStyle.Render(" │ ")) + dimStyle.Render("]")
}

func (m Model) renderBuildRow(i int, build buildkite.Build) string {
	creator := "—"
	if build.Creator != nil && build.Creator.Name != "" {
		creator = build.Creator.Name
		if len(creator) > 13 {
			creator = creator[:13]
		}
	}
	branch := build.Branch
	if len(branch) > 16 {
		branch = branch[:16]
	}
	duration := FormatDuration(build.StartedAt, build.FinishedAt)

	line := fmt.Sprintf("%-8d %-16s %-9s %-9s %-15s %s",
		build.Number,
		branch,
		shortSHA(build.Commit),
		stateBadge(build.State),
		creator,
		duration,
	)

	cursor := "  "
	if i == m.buildIndex {
		cursor = "▶ "
	}

	if i == m.buildIndex {
		return selectedItemStyle.Render(cursor+line) + "\n"
	}
	return normalItemStyle.Render(cursor+line) + "\n"
}

func (m Model) rightPaneView(w, h int) string {
	style := borderStyle
	if m.activePane == rightPane {
		style = activeBorderStyle
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Build Detail"))
	b.WriteString("\n")

	if m.loadingDetail && m.selectedBuild == nil {
		b.WriteString(loadingStyle.Render("Loading build details..."))
		b.WriteString("\n")
	} else if m.selectedBuild == nil {
		b.WriteString(dimStyle.Render("Select a build"))
		b.WriteString("\n")
	} else {
		bd := m.selectedBuild
		creator := "—"
		if bd.Creator != nil {
			creator = bd.Creator.Name
		}

		field := func(label, value string) string {
			return fmt.Sprintf(" %-9s%s\n", label+":", value)
		}

		b.WriteString(field("Number", fmt.Sprintf("#%d", bd.Number)))
		b.WriteString(field("State", stateBadge(bd.State)))
		b.WriteString(field("Branch", bd.Branch))
		b.WriteString(field("Commit", shortSHA(bd.Commit)))
		msg := formatBuildMessage(bd.Message)
		if len(msg) > w-14 {
			msg = msg[:w-14]
		}
		b.WriteString(field("Message", msg))
		b.WriteString(field("Creator", renderEmoji(creator)))
		b.WriteString(field("Created", FormatTime(bd.CreatedAt)))
		b.WriteString(field("Started", FormatTime(bd.StartedAt)))
		b.WriteString(field("Finished", FormatTime(bd.FinishedAt)))
		b.WriteString(field("Duration", FormatDuration(bd.StartedAt, bd.FinishedAt)))

		if !m.denseMode {
			b.WriteString("\n")
		}
		b.WriteString(titleStyle.Render("Jobs"))
		if query := normalizedQueryForPane(m, rightPane); query != "" {
			b.WriteString(subtitleStyle.Render(fmt.Sprintf(" /%s", query)))
		}
		b.WriteString("\n")

		jobs := m.filteredJobs()
		if len(bd.Jobs) == 0 && m.loadingDetail {
			b.WriteString(loadingStyle.Render("Loading..."))
			b.WriteString("\n")
		} else if len(bd.Jobs) == 0 {
			b.WriteString(dimStyle.Render("No jobs"))
			b.WriteString("\n")
		} else if len(jobs) == 0 {
			b.WriteString(dimStyle.Render("No matching jobs"))
			b.WriteString("\n")
		} else {
			jobIndex := 0
			for _, job := range jobs {
				if job.Type == "waiter" {
					continue
				}
				label := job.Name
				if label == "" {
					label = job.Label
				}
				if label == "" {
					label = job.Command
				}
				label = renderEmoji(label)
				if len(label) > w-20 {
					label = label[:w-20]
				}
				cursor := "  "
				if m.activePane == rightPane && jobIndex == m.rightScroll {
					cursor = "▶ "
				}
				line := fmt.Sprintf("%s %-6s%s", cursor, stateBadge(job.State), label)

				if job.Agent != nil {
					line += dimStyle.Render(fmt.Sprintf(" %s", job.Agent.Name))
				}
				if job.ExitStatus != nil {
					exitStyle := dimStyle
					if *job.ExitStatus != 0 {
						exitStyle = errorStyle
					}
					line += exitStyle.Render(fmt.Sprintf(" %d", *job.ExitStatus))
				}
				if m.activePane == rightPane && jobIndex == m.rightScroll {
					b.WriteString(selectedItemStyle.Render(line))
				} else {
					b.WriteString(normalItemStyle.Render(line))
				}
				b.WriteString("\n")
				jobIndex++
			}
		}

		b.WriteString(m.renderAnnotations())
		b.WriteString(m.renderArtifacts(w))
	}

	return style.Width(w).Height(h).Render(b.String())
}

func (m Model) renderAnnotations() string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Annotations"))
	b.WriteString("\n")

	if m.loadingAnnotations {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if len(m.annotations) == 0 {
		b.WriteString(dimStyle.Render("No annotations"))
		b.WriteString("\n")
	} else {
		for _, ann := range m.annotations {
			style := dimStyle
			switch ann.Style {
			case "success":
				style = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			case "error":
				style = errorStyle
			case "warning":
				style = loadingStyle
			case "info":
				style = helpStyle
			}
			ctx := ann.Context
			if ctx == "" {
				ctx = "default"
			}
			b.WriteString(style.Render(fmt.Sprintf(" [%s]", ctx)))
			body := stripHTMLTags(ann.BodyHTML)
			if len(body) > 40 {
				body = body[:40] + "…"
			}
			if body != "" {
				b.WriteString(" " + body)
			}
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) renderArtifacts(w int) string {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Artifacts"))
	if len(m.artifacts) > 0 {
		b.WriteString(dimStyle.Render("  d:download"))
	}
	b.WriteString("\n")

	if m.loadingArtifacts {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if len(m.artifacts) == 0 {
		b.WriteString(dimStyle.Render("No artifacts"))
		b.WriteString("\n")
	} else {
		for _, art := range m.artifacts {
			filename := art.Filename
			if len(filename) > w-15 {
				filename = filename[:w-15] + "…"
			}
			size := formatFileSize(art.FileSize)
			b.WriteString(fmt.Sprintf(" %s %s", dimStyle.Render("•"), filename))
			b.WriteString(dimStyle.Render(fmt.Sprintf(" (%s)", size)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (m Model) statusBarView(w int) string {
	var parts []string

	if m.err != nil {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("ERR: %s", m.errMsg)))
	}

	if m.searchMsg != "" {
		parts = append(parts, loadingStyle.Render(m.searchMsg))
	}

	if m.searching {
		parts = append(parts, loadingStyle.Render(fmt.Sprintf("Filter: /%s", m.filterQuery)))
	} else if m.filterQuery != "" {
		parts = append(parts, dimStyle.Render(fmt.Sprintf("Filter: /%s", m.filterQuery)))
	}

	paneName := "Orgs/Pipes"
	switch m.activePane {
	case centerPane:
		paneName = "Builds"
	case rightPane:
		paneName = "Detail"
	}
	parts = append(parts, fmt.Sprintf("Pane: %s", paneName))

	if !m.lastRefresh.IsZero() {
		rate := m.currentPollInterval()
		rateLabel := fmt.Sprintf("Updated: %s (%s)", m.lastRefresh.Format("15:04:05"), rate)
		if m.liveMode {
			rateLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("#22c55e")).Bold(true).Render("⚡LIVE") + "  " + rateLabel
		} else if rate == pollIntervalActive {
			rateLabel = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("live") + "  " + rateLabel
		}
		parts = append(parts, rateLabel)
	}

	parts = append(parts, helpStyle.Render("↑/k ↓/j ←/h →/l  ?:help  q:quit  R:refresh"))

	return statusStyle.Width(w).Render(strings.Join(parts, "  │  "))
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("builddeck — Keybindings"))
	b.WriteString("\n\n")
	b.WriteString("Navigation:\n")
	b.WriteString("  ↑/k         Move up\n")
	b.WriteString("  ↓/j         Move down\n")
	b.WriteString("  ←/h         Previous pane\n")
	b.WriteString("  →/l         Next pane\n")
	b.WriteString("  tab         Next pane\n")
	b.WriteString("  shift+tab   Previous pane\n")
	b.WriteString("  g           Jump to top of list\n")
	b.WriteString("  G           Jump to bottom of list\n")
	b.WriteString("  enter       Select / drill down\n")
	b.WriteString("\n")
	b.WriteString("Actions:\n")
	b.WriteString("  R           Refresh all data\n")
	b.WriteString("  r           Rebuild (builds pane) / rerun (detail pane)\n")
	b.WriteString("  x           Cancel selected/top running build\n")
	b.WriteString("  u           Unblock selected blocked job\n")
	b.WriteString("  L           Tail selected/top job logs\n")
	b.WriteString("  o           Open current resource in browser\n")
	b.WriteString("  ctrl+o      Open pipeline's git repo in browser\n")
	b.WriteString("  ctrl+d      Open commit diff in browser\n")
	b.WriteString("  d           Download first artifact\n")
	b.WriteString("  /           Filter active pane\n")
	b.WriteString("  esc/enter   Close filter input\n")
	b.WriteString("  ctrl+u      Clear filter input\n")
	b.WriteString("\n")
	b.WriteString("Views:\n")
	b.WriteString("  s           Stats for selected build/job\n")
	b.WriteString("  ctrl+l      Toggle live mode (force 2s refresh)\n")
	b.WriteString("  a           Toggle agent/queue saturation view\n")
	b.WriteString("  ctrl+f      Global search across all data\n")
	b.WriteString("  S           Save current filter as preset\n")
	b.WriteString("  P           Load a saved filter preset\n")
	b.WriteString("  ?           Toggle this help\n")
	b.WriteString("  q           Quit\n")
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press ? or esc to close"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Render(b.String())
}

func (m Model) agentView() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Agent & Queue Saturation"))
	org := m.selectedOrg()
	if org != nil {
		b.WriteString(subtitleStyle.Render(fmt.Sprintf(" — %s", org.Slug)))
	}
	b.WriteString("\n")

	contentHeight := m.height - 3
	if contentHeight < 5 {
		contentHeight = 5
	}

	var content strings.Builder

	if m.loadingAgents {
		content.WriteString(loadingStyle.Render("Loading agents..."))
		content.WriteString("\n")
	} else if len(m.agents) == 0 {
		content.WriteString(dimStyle.Render("No agents found"))
		content.WriteString("\n")
	} else {
		// Queue summary
		queueMap := make(map[string]int)
		queueConnected := make(map[string]int)
		for _, agent := range m.agents {
			q := agent.Queue
			queueMap[q]++
			if agent.ConnectedState == "connected" {
				queueConnected[q]++
			}
		}

		content.WriteString(titleStyle.Render("Queue Summary"))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render(fmt.Sprintf("%-20s %-10s %-10s %s", "QUEUE", "TOTAL", "ONLINE", "SATURATION")))
		content.WriteString("\n")
		for queue, total := range queueMap {
			connected := queueConnected[queue]
			saturation := 0
			if total > 0 {
				saturation = (connected * 100) / total
			}
			satStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			if saturation < 50 {
				satStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
			} else if saturation < 80 {
				satStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			}
			content.WriteString(fmt.Sprintf("%-20s %-10d %-10d %s",
				queue,
				total,
				connected,
				satStyle.Render(fmt.Sprintf("%d%%", saturation)),
			))
			content.WriteString("\n")
		}

		content.WriteString("\n")
		content.WriteString(titleStyle.Render(fmt.Sprintf("Agents (%d)", len(m.agents))))
		content.WriteString("\n")
		content.WriteString(dimStyle.Render(fmt.Sprintf("%-25s %-15s %-12s %-10s %-10s %s", "NAME", "HOSTNAME", "STATE", "VERSION", "OS", "QUEUE")))
		content.WriteString("\n")

		for i, agent := range m.agents {
			name := agent.Name
			if len(name) > 24 {
				name = name[:24]
			}
			hostname := agent.Hostname
			if len(hostname) > 14 {
				hostname = hostname[:14]
			}

			stateStyle := dimStyle
			stateStr := agent.ConnectedState
			if agent.ConnectedState == "connected" {
				stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
				stateStr = "● connected"
			} else {
				stateStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
				stateStr = "○ " + agent.ConnectedState
			}
			if len(stateStr) > 11 {
				stateStr = stateStr[:11]
			}

			ver := agent.Version
			if len(ver) > 9 {
				ver = ver[:9]
			}
			agentOS := agent.OS
			if len(agentOS) > 9 {
				agentOS = agentOS[:9]
			}

			cursor := "  "
			if i == m.rightScroll {
				cursor = "▶ "
			}

			line := fmt.Sprintf("%s%-25s %-15s %s %-10s %-10s %s",
				cursor,
				name,
				hostname,
				stateStyle.Render(fmt.Sprintf("%-12s", stateStr)),
				ver,
				agentOS,
				agent.Queue,
			)

			if i == m.rightScroll {
				content.WriteString(selectedItemStyle.Render(line))
			} else {
				content.WriteString(normalItemStyle.Render(line))
			}
			content.WriteString("\n")
		}
	}

	b.WriteString(activeBorderStyle.Width(m.width).Height(contentHeight).Render(content.String()))
	b.WriteString("\n")

	var statusParts []string
	statusParts = append(statusParts, fmt.Sprintf("%d agents", len(m.agents)))
	statusParts = append(statusParts, helpStyle.Render("a/esc:back  ↑/k:up  ↓/j:down  R:refresh  o:open  q:quit"))
	b.WriteString(statusStyle.Width(m.width).Render(strings.Join(statusParts, "  │  ")))

	return b.String()
}

func (m Model) globalSearchOverlay(base string) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("Global Search"))
	b.WriteString("\n")

	if m.globalSearching {
		b.WriteString(loadingStyle.Render(fmt.Sprintf("Search: %s▌", m.globalSearchQuery)))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("Search: %s", m.globalSearchQuery)))
	}
	b.WriteString("\n\n")

	if len(m.globalSearchResult) == 0 && m.globalSearchQuery != "" {
		b.WriteString(dimStyle.Render("No results"))
		b.WriteString("\n")
	} else {
		for i, r := range m.globalSearchResult {
			if i >= 20 {
				b.WriteString(dimStyle.Render(fmt.Sprintf("  ... and %d more", len(m.globalSearchResult)-20)))
				b.WriteString("\n")
				break
			}
			typeStyle := dimStyle
			switch r.Type {
			case "org":
				typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("69"))
			case "pipeline":
				typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("214"))
			case "build":
				typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
			case "job":
				typeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("199"))
			}
			b.WriteString(fmt.Sprintf("  %s %s", typeStyle.Render(fmt.Sprintf("[%-8s]", r.Type)), renderEmoji(r.Label)))
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("esc:close  enter:search  ctrl+u:clear"))

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(m.width - 10).
		Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) presetPickerOverlay(base string) string {
	presets := m.activeFilterPresets()

	var b strings.Builder
	b.WriteString(titleStyle.Render("  Filter Presets  "))
	b.WriteString("\n\n")

	if len(presets) == 0 {
		b.WriteString(dimStyle.Render("  No saved presets"))
		b.WriteString("\n")
	} else {
		usableWidth := 50
		for i, p := range presets {
			cursor := "  "
			if i == m.presetPickerIndex {
				cursor = "❯ "
			}
			paneLabel := dimStyle.Render(fmt.Sprintf("[%s]", p.Pane))
			line := fmt.Sprintf("%s %s %s", p.Name, paneLabel, dimStyle.Render("→ /"+p.Query))
			rawLine := cursor + line
			visWidth := lipgloss.Width(rawLine)
			if visWidth < usableWidth {
				rawLine += strings.Repeat(" ", usableWidth-visWidth)
			}
			if i == m.presetPickerIndex {
				b.WriteString(selectedItemStyle.Render(rawLine))
			} else {
				b.WriteString(normalItemStyle.Render(rawLine))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  enter:select  │  esc:close  │  ↑/↓:navigate"))

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(activeTheme.BorderActive)).
		Padding(1, 2).
		Width(60).
		Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) optionsOverlay(base string) string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("  Builddeck Options  "))
	b.WriteString("\n\n")

	// Themes options
	themeNames := make([]string, len(Themes))
	for i, t := range Themes {
		themeNames[i] = t.Name
	}
	themeVal := fmt.Sprintf("◀  %s  ▶", themeNames[m.themeIndex])

	// Refresh Rate options
	refreshNames := []string{"Dynamic", "2s (Live)", "5s", "10s (Idle)", "30s", "Disabled"}
	refreshVal := fmt.Sprintf("◀  %s  ▶", refreshNames[m.refreshRateIndex])

	// Layout Density
	densityVal := "◀  Spacious  ▶"
	if m.denseMode {
		densityVal = "◀  Dense     ▶"
	}

	// Sorting
	sortingVal := "◀  Newest First  ▶"
	if m.sortAsc {
		sortingVal = "◀  Oldest First  ▶"
	}

	options := []struct {
		Label string
		Value string
	}{
		{"Theme           ", themeVal},
		{"Refresh Rate    ", refreshVal},
		{"Layout Density  ", densityVal},
		{"Build Sorting   ", sortingVal},
	}

	for i, opt := range options {
		cursor := "  "
		if i == m.optionsCursor {
			cursor = "❯ "
		}

		line := fmt.Sprintf("%-16s  %s", opt.Label, opt.Value)
		rawLine := cursor + line
		usableWidth := 56
		visWidth := lipgloss.Width(rawLine)
		if visWidth < usableWidth {
			rawLine += strings.Repeat(" ", usableWidth-visWidth)
		}

		if i == m.optionsCursor {
			b.WriteString(selectedItemStyle.Render(rawLine) + "\n")
		} else {
			b.WriteString(normalItemStyle.Render(rawLine) + "\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(dimStyle.Render("  esc:close  │  ↑/↓:navigate  │  ←/→:change value"))

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(activeTheme.BorderActive)).
		Padding(1, 2).
		Width(65).
		Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func (m Model) statsOverlay(base string) string {
	var b strings.Builder

	switch m.showStatsOverlay {
	case "agent":
		b.WriteString(titleStyle.Render("  Agent Stats  "))
		b.WriteString("\n\n")
		job := m.selectedRightPaneJob()
		if job == nil {
			b.WriteString(dimStyle.Render("  No job selected"))
		} else {
			label := job.Name
			if label == "" {
				label = job.Label
			}
			if label == "" {
				label = job.Command
			}
			b.WriteString(fmt.Sprintf("  %s  %s\n", stateBadge(job.State), renderEmoji(label)))
			b.WriteString(fmt.Sprintf("    Duration: %s\n", FormatDuration(job.StartedAt, job.FinishedAt)))
			if job.Agent != nil {
				ag := job.Agent
				b.WriteString(fmt.Sprintf("    Agent:    %s\n", ag.Name))
				b.WriteString(fmt.Sprintf("    Hostname: %s\n", ag.Hostname))
				b.WriteString(fmt.Sprintf("    OS:       %s\n", ag.OS))
				b.WriteString(fmt.Sprintf("    Version:  %s\n", ag.Version))
				if ag.IPAddress != "" {
					b.WriteString(fmt.Sprintf("    IP:       %s\n", ag.IPAddress))
				}
				b.WriteString(fmt.Sprintf("    Queue:    %s\n", ag.Queue))
				if len(ag.Metadata) > 0 {
					b.WriteString("    Metadata:\n")
					for _, meta := range ag.Metadata {
						b.WriteString(fmt.Sprintf("      • %s\n", renderEmoji(meta)))
					}
				}
			} else {
				b.WriteString(dimStyle.Render("    No agent assigned\n"))
			}
		}

	case "build":
		b.WriteString(titleStyle.Render("  Build Stats  "))
		b.WriteString("\n\n")
		bd := m.selectedBuild
		if bd == nil {
			b.WriteString(dimStyle.Render("  No build selected"))
		} else {
			pipe := m.selectedPipeline()
			pipeName := ""
			if pipe != nil {
				pipeName = pipe.Slug
			}
			b.WriteString(fmt.Sprintf("  Pipeline: %s\n", pipeName))
			b.WriteString(fmt.Sprintf("  Build:    #%d\n", bd.Number))
			b.WriteString(fmt.Sprintf("  State:    %s\n", bd.State))
			b.WriteString(fmt.Sprintf("  Branch:   %s\n", bd.Branch))
			b.WriteString(fmt.Sprintf("  Commit:   %s\n", shortSHA(bd.Commit)))
			msg := renderEmoji(strings.SplitN(bd.Message, "\n", 2)[0])
			b.WriteString(fmt.Sprintf("  Message:  %s\n", msg))
			if bd.Creator != nil {
				b.WriteString(fmt.Sprintf("  Creator:  %s\n", renderEmoji(bd.Creator.Name)))
			}
			b.WriteString(fmt.Sprintf("  Created:  %s\n", FormatTime(bd.CreatedAt)))
			b.WriteString(fmt.Sprintf("  Started:  %s\n", FormatTime(bd.StartedAt)))
			b.WriteString(fmt.Sprintf("  Finished: %s\n", FormatTime(bd.FinishedAt)))
			b.WriteString(fmt.Sprintf("  Duration: %s\n", FormatDuration(bd.StartedAt, bd.FinishedAt)))
		}
	}

	b.WriteString(dimStyle.Render("  esc/s:close  q:quit"))

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(activeTheme.BorderActive)).
		Padding(1, 2).
		Render(b.String())

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, overlay)
}

func stripHTMLTags(s string) string {
	var result []rune
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
			continue
		}
		if r == '>' {
			inTag = false
			continue
		}
		if !inTag {
			result = append(result, r)
		}
	}
	return strings.TrimSpace(string(result))
}

func formatFileSize(bytes int) string {
	if bytes < 1024 {
		return fmt.Sprintf("%dB", bytes)
	}
	if bytes < 1024*1024 {
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	}
	return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
}

func (m Model) logsView() string {
	var b strings.Builder

	headerH := 1

	bd := m.selectedBuild
	jobName := "unknown job"
	if bd != nil {
		for _, job := range bd.Jobs {
			if job.ID == m.logJobID {
				jobName = job.Name
				if jobName == "" {
					jobName = job.Label
				}
				if jobName == "" {
					jobName = job.Command
				}
				break
			}
		}
	}

	title := titleStyle.Render(fmt.Sprintf("Logs — %s", renderEmoji(jobName)))
	if bd != nil {
		title += " " + subtitleStyle.Render(fmt.Sprintf("#%d", bd.Number))
	}

	if m.loadingLog {
		title += " " + loadingStyle.Render("⟳ loading logs...")
	}

	b.WriteString(title)
	b.WriteString("\n")

	contentHeight := m.height - headerH - 1
	if contentHeight < 2 {
		contentHeight = 2
	}

	var logBox strings.Builder
	if m.loadingLog {
		logBox.WriteString(loadingStyle.Render("Retrieving build logs from Buildkite API..."))
	} else if m.currentLog == "" {
		logBox.WriteString(dimStyle.Render("No log output recorded for this job."))
	} else {
		lines := strings.Split(m.currentLog, "\n")
		start := m.logScroll
		if start < 0 {
			start = 0
		}
		if start > len(lines) {
			start = len(lines)
		}
		end := start + contentHeight - 2
		if end > len(lines) {
			end = len(lines)
		}

		for i := start; i < end; i++ {
			logBox.WriteString(lines[i])
			logBox.WriteString("\n")
		}
	}

	logBorder := activeBorderStyle
	if m.loadingLog {
		logBorder = borderStyle
	}

	b.WriteString(logBorder.Width(m.width).Height(contentHeight).Render(logBox.String()))
	b.WriteString("\n")

	var statusParts []string
	if m.currentLog != "" {
		lines := strings.Split(m.currentLog, "\n")
		scrollPercent := 0
		maxScroll := len(lines) - (contentHeight - 2)
		if maxScroll > 0 {
			scrollPercent = (m.logScroll * 100) / maxScroll
			if scrollPercent > 100 {
				scrollPercent = 100
			}
		}
		statusParts = append(statusParts, fmt.Sprintf("Line %d/%d (%d%%)", m.logScroll+1, len(lines), scrollPercent))
	}

	statusParts = append(statusParts, helpStyle.Render("L/esc:back  ↑/k:up  ↓/j:down  g:top  G:bottom  q:quit"))

	b.WriteString(statusStyle.Width(m.width).Render(strings.Join(statusParts, "  │  ")))

	return b.String()
}

const maxMsgLen = 20

// formatBuildMessage truncates a build message to maxMsgLen characters and
// appends a PR reference if one is found in the original message.
// PR references are detected as "(#digits)" at the end of the first line.
func formatBuildMessage(msg string) string {
	firstLine := msg
	if idx := strings.IndexByte(msg, '\n'); idx >= 0 {
		firstLine = msg[:idx]
	}
	firstLine = strings.TrimSpace(firstLine)

	prRef := ""
	body := firstLine
	if end := strings.LastIndex(firstLine, ")"); end >= 0 && end == len(firstLine)-1 {
		start := strings.LastIndex(firstLine[:end], "(")
		if start >= 0 {
			candidate := firstLine[start : end+1]
			if len(candidate) > 3 && candidate[0] == '(' && candidate[1] == '#' {
				digits := candidate[2 : len(candidate)-1]
				allDigits := len(digits) > 0
				for _, c := range digits {
					if c < '0' || c > '9' {
						allDigits = false
						break
					}
				}
				if allDigits {
					prRef = candidate
					body = strings.TrimSpace(firstLine[:start])
				}
			}
		}
	}

	if prRef != "" {
		if len(body) > maxMsgLen {
			body = body[:maxMsgLen] + "…"
		}
		return renderEmoji(body + " " + prRef)
	}

	if len(body) > maxMsgLen {
		body = body[:maxMsgLen] + "…"
	}
	return renderEmoji(body)
}
