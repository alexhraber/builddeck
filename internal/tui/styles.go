package tui

import (
	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	activeBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("69"))

	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("252"))

	subtitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("246"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("243"))

	selectedItemStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("230"))

	normalItemStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))

	statusStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("244"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("69"))

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196"))

	loadingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))
)

func stateColor(state string) lipgloss.Color {
	switch state {
	case "passed":
		return lipgloss.Color("82")
	case "failed", "failing":
		return lipgloss.Color("196")
	case "running", "active":
		return lipgloss.Color("214")
	case "scheduled", "waiting":
		return lipgloss.Color("69")
	case "canceled", "cancelled":
		return lipgloss.Color("243")
	case "skipped":
		return lipgloss.Color("243")
	case "blocked":
		return lipgloss.Color("199")
	case "not_run":
		return lipgloss.Color("243")
	default:
		return lipgloss.Color("252")
	}
}

func stateBadge(state string) string {
	return lipgloss.NewStyle().
		Foreground(stateColor(state)).
		Bold(true).
		Render(stateLabel(state))
}

func stateLabel(state string) string {
	switch state {
	case "passed":
		return "PASSED"
	case "failed":
		return "FAILED"
	case "running":
		return "RUNNING"
	case "scheduled":
		return "SCHED "
	case "canceled", "cancelled":
		return "CNCLD"
	case "skipped":
		return "SKIPPD"
	case "blocked":
		return "BLOCKED"
	case "waiting":
		return "WAITNG"
	case "not_run":
		return "NOTRUN"
	case "failing":
		return "FAILING"
	case "active":
		return "ACTIVE"
	default:
		return state
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
