package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.ready {
		return "Initializing builddeck..."
	}

	if m.showHelp {
		return m.helpView()
	}

	width := m.width
	height := m.height

	leftW := width / 4
	centerW := width / 3
	rightW := width - leftW - centerW
	if rightW < 20 {
		rightW = 20
	}

	mainHeight := height - 3

	left := m.leftPaneView(leftW, mainHeight)
	center := m.centerPaneView(centerW, mainHeight)
	right := m.rightPaneView(rightW, mainHeight)

	panes := lipgloss.JoinHorizontal(lipgloss.Top, left, center, right)
	status := m.statusBarView(width)

	return lipgloss.JoinVertical(lipgloss.Left, panes, status)
}

func (m Model) leftPaneView(w, h int) string {
	style := borderStyle
	if m.activePane == leftPane {
		style = activeBorderStyle
	}

	innerW := w - 4
	if innerW < 1 {
		innerW = 1
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Organizations"))
	b.WriteString("\n")

	if m.loadingOrgs {
		b.WriteString(loadingStyle.Render("Loading..."))
	} else if len(m.orgs) == 0 {
		b.WriteString(dimStyle.Render("No organizations"))
	} else {
		for i, org := range m.orgs {
			if i == m.orgIndex {
				b.WriteString(selectedItemStyle.Render("▶ " + org.Name))
			} else {
				b.WriteString(normalItemStyle.Render("  " + org.Name))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString("\n")
	b.WriteString(titleStyle.Render("Pipelines"))
	b.WriteString("\n")

	if m.loadingPipes {
		b.WriteString(loadingStyle.Render("Loading..."))
	} else if m.selectedOrg() == nil {
		b.WriteString(dimStyle.Render("Select an organization"))
	} else if len(m.pipelines) == 0 {
		b.WriteString(dimStyle.Render("No pipelines"))
	} else {
		for i, pipe := range m.pipelines {
			if i == m.pipeIndex {
				b.WriteString(selectedItemStyle.Render("▶ " + pipe.Name))
			} else {
				b.WriteString(normalItemStyle.Render("  " + pipe.Name))
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

	if m.loadingBuilds {
		b.WriteString(loadingStyle.Render("Loading..."))
	} else if m.selectedPipeline() == nil {
		b.WriteString(dimStyle.Render("Select a pipeline"))
	} else if len(m.builds) == 0 {
		b.WriteString(dimStyle.Render("No builds"))
	} else {
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-8s %-8s %-9s %-8s %s\n", "BUILD", "BRANCH", "COMMIT", "STATE", "CREATOR")))
		for i, build := range m.builds {
			creator := "—"
			if build.Creator != nil {
				creator = build.Creator.Name
			}
			branch := build.Branch
			if len(branch) > 10 {
				branch = branch[:10]
			}
			line := fmt.Sprintf("%-8d %-10s %-9s %-8s %s",
				build.Number,
				branch,
				shortSHA(build.Commit),
				stateBadge(build.State),
				creator,
			)
			if i == m.buildIndex {
				b.WriteString(selectedItemStyle.Render("▶ " + line))
			} else {
				b.WriteString(normalItemStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}

	return style.Width(w).Height(h).Render(b.String())
}

func (m Model) rightPaneView(w, h int) string {
	style := borderStyle
	if m.activePane == rightPane {
		style = activeBorderStyle
	}

	var b strings.Builder

	b.WriteString(titleStyle.Render("Build Detail"))
	b.WriteString("\n")

	if m.loadingDetail {
		b.WriteString(loadingStyle.Render("Loading build details..."))
	} else if m.selectedBuild == nil {
		b.WriteString(dimStyle.Render("Select a build"))
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
		b.WriteString(fmt.Sprintf("Message: %s\n", bd.Message))
		b.WriteString(fmt.Sprintf("Creator: %s\n", creator))
		b.WriteString(fmt.Sprintf("Created: %s\n", FormatTime(bd.CreatedAt)))
		b.WriteString(fmt.Sprintf("Started: %s\n", FormatTime(bd.StartedAt)))
		b.WriteString(fmt.Sprintf("Finished:%s\n", FormatTime(bd.FinishedAt)))
		b.WriteString(fmt.Sprintf("Duration:%s\n", FormatDuration(bd.StartedAt, bd.FinishedAt)))

		b.WriteString("\n")
		b.WriteString(titleStyle.Render("Jobs"))
		b.WriteString("\n")

		if len(bd.Jobs) == 0 {
			b.WriteString(dimStyle.Render("No jobs"))
		} else {
			for _, job := range bd.Jobs {
				label := job.Label
				if label == "" {
					label = job.Command
				}
				if len(label) > 30 {
					label = label[:30]
				}
				b.WriteString(fmt.Sprintf(" %s %s", stateBadge(job.State), label))

				if job.Agent != nil {
					b.WriteString(dimStyle.Render(fmt.Sprintf(" [%s]", job.Agent.Name)))
				}
				if job.ExitStatus != nil {
					b.WriteString(dimStyle.Render(fmt.Sprintf(" exit:%d", *job.ExitStatus)))
				}
				b.WriteString("\n")
			}
		}
	}

	return style.Width(w).Height(h).Render(b.String())
}

func (m Model) statusBarView(w int) string {
	var parts []string

	if m.err != nil {
		parts = append(parts, errorStyle.Render(fmt.Sprintf("ERR: %s", m.errMsg)))
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

	parts = append(parts, helpStyle.Render("[?] help  [q] quit  [r] refresh  [tab] pane"))

	return statusStyle.Width(w).Render(strings.Join(parts, "  │  "))
}

func (m Model) helpView() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("builddeck — Help"))
	b.WriteString("\n\n")
	b.WriteString("Navigation:\n")
	b.WriteString("  ↑/k         Move up\n")
	b.WriteString("  ↓/j         Move down\n")
	b.WriteString("  tab         Next pane\n")
	b.WriteString("  shift+tab   Previous pane\n")
	b.WriteString("  enter       Select / drill down\n")
	b.WriteString("\n")
	b.WriteString("Actions:\n")
	b.WriteString("  r           Refresh all data\n")
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
