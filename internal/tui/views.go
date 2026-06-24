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

	headerHeight := 1
	statusHeight := 1
	mainHeight := m.height - headerHeight - statusHeight
	if mainHeight < 5 {
		mainHeight = 5
	}

	leftW := m.width / 4
	centerW := m.width / 3
	rightW := m.width - leftW - centerW

	header := m.headerView(m.width)
	left := m.leftPaneView(leftW, mainHeight)
	center := m.centerPaneView(centerW, mainHeight)
	right := m.rightPaneView(rightW, mainHeight)
	status := m.statusBarView(m.width)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)

	return lipgloss.JoinVertical(lipgloss.Left, header, panes, status)
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
		for i, org := range m.orgs {
			cursor := "  "
			if i == m.orgIndex {
				cursor = "▶ "
			}
			name := org.Name
			if len(name) > w-6 {
				name = name[:w-6]
			}
			if i == m.orgIndex {
				b.WriteString(selectedItemStyle.Render(cursor + name))
			} else {
				b.WriteString(normalItemStyle.Render(cursor + name))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Pipelines"))
	b.WriteString("\n")

	if m.loadingPipes {
		b.WriteString(loadingStyle.Render("Loading..."))
		b.WriteString("\n")
	} else if m.selectedOrg() == nil {
		b.WriteString(dimStyle.Render("Select an organization"))
		b.WriteString("\n")
	} else if len(m.pipelines) == 0 {
		b.WriteString(dimStyle.Render("No pipelines"))
		b.WriteString("\n")
	} else {
		for i, pipe := range m.pipelines {
			cursor := "  "
			if i == m.pipeIndex {
				cursor = "▶ "
			}
			name := pipe.Name
			if len(name) > w-6 {
				name = name[:w-6]
			}
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
	b.WriteString("\n")

	if len(m.builds) > 0 {
		summary := SummarizeBuilds(m.builds)
		b.WriteString(m.renderBuildSummary(summary))
		b.WriteString("\n")
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
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-8s %-10s %-9s %-9s %-12s %s\n", "BUILD", "BRANCH", "COMMIT", "STATE", "CREATOR", "DURATION")))
		for i, build := range m.builds {
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
		if len(creator) > 10 {
			creator = creator[:10]
		}
	}
	branch := build.Branch
	if len(branch) > 10 {
		branch = branch[:10]
	}
	duration := FormatDuration(build.StartedAt, build.FinishedAt)

	line := fmt.Sprintf("%-8d %-10s %-9s %-9s %-12s %s",
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

		b.WriteString(fmt.Sprintf("Number:  #%d\n", bd.Number))
		b.WriteString(fmt.Sprintf("State:   %s\n", stateBadge(bd.State)))
		b.WriteString(fmt.Sprintf("Branch:  %s\n", bd.Branch))
		b.WriteString(fmt.Sprintf("Commit:  %s\n", shortSHA(bd.Commit)))
		msg := bd.Message
		if len(msg) > w-12 {
			msg = msg[:w-12] + "…"
		}
		b.WriteString(fmt.Sprintf("Message: %s\n", msg))
		b.WriteString(fmt.Sprintf("Creator: %s\n", creator))
		b.WriteString(fmt.Sprintf("Created: %s\n", FormatTime(bd.CreatedAt)))
		b.WriteString(fmt.Sprintf("Started: %s\n", FormatTime(bd.StartedAt)))
		b.WriteString(fmt.Sprintf("Finished:%s\n", FormatTime(bd.FinishedAt)))
		b.WriteString(fmt.Sprintf("Duration:%s\n", FormatDuration(bd.StartedAt, bd.FinishedAt)))

		b.WriteString("\n")
		b.WriteString(titleStyle.Render("Jobs"))
		b.WriteString("\n")

		if len(bd.Jobs) == 0 && m.loadingDetail {
			b.WriteString(loadingStyle.Render("Loading..."))
			b.WriteString("\n")
		} else if len(bd.Jobs) == 0 {
			b.WriteString(dimStyle.Render("No jobs"))
			b.WriteString("\n")
		} else {
			for _, job := range bd.Jobs {
				if job.Type == "waiter" {
					continue
				}
				label := job.Label
				if label == "" {
					label = job.Command
				}
				if len(label) > w-20 {
					label = label[:w-20]
				}
				b.WriteString(fmt.Sprintf(" %s %s", stateBadge(job.State), label))

				if job.Agent != nil {
					b.WriteString(dimStyle.Render(fmt.Sprintf(" [%s]", job.Agent.Name)))
				}
				if job.ExitStatus != nil {
					exitStyle := dimStyle
					if *job.ExitStatus != 0 {
						exitStyle = errorStyle
					}
					b.WriteString(exitStyle.Render(fmt.Sprintf(" exit:%d", *job.ExitStatus)))
				}
				b.WriteString("\n")
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

	paneName := "Orgs/Pipes"
	switch m.activePane {
	case centerPane:
		paneName = "Builds"
	case rightPane:
		paneName = "Detail"
	}
	parts = append(parts, fmt.Sprintf("Pane: %s", paneName))

	if !m.lastRefresh.IsZero() {
		parts = append(parts, fmt.Sprintf("Updated: %s", m.lastRefresh.Format("15:04:05")))
	}

	parts = append(parts, helpStyle.Render("?:help  q:quit  r:refresh  tab:pane  g/G:top/bot"))

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
	b.WriteString("  r           Refresh all data\n")
	b.WriteString("  /           Search (not yet implemented)\n")
	b.WriteString("  ?           Toggle this help\n")
	b.WriteString("  q           Quit\n")
	b.WriteString("\n")
	b.WriteString("Planned (not yet implemented):\n")
	b.WriteString("  x           Cancel build\n")
	b.WriteString("  R           Retry job\n")
	b.WriteString("  b           Rebuild build\n")
	b.WriteString("  u           Unblock job\n")
	b.WriteString("  o           Open in browser\n")
	b.WriteString("  l           Tail logs\n")
	b.WriteString("  d           Download artifact\n")
	b.WriteString("\n\n")
	b.WriteString(dimStyle.Render("Press ? or esc to close"))

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Render(b.String())
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
