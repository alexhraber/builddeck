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
	case "timed_out":
		return lipgloss.Color("196")
	case "broken":
		return lipgloss.Color("196")
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
		return "PASS"
	case "failed":
		return "FAIL"
	case "running":
		return "RUN"
	case "scheduled":
		return "SCHD"
	case "canceled", "cancelled":
		return "CNCL"
	case "skipped":
		return "SKIP"
	case "blocked":
		return "BLCK"
	case "waiting":
		return "WAIT"
	case "not_run":
		return "NRUN"
	case "failing":
		return "FLNG"
	case "active":
		return "ACTV"
	case "timed_out":
		return "TMOU"
	case "broken":
		return "BRKN"
	default:
		if len(state) > 4 {
			return state[:4]
		}
		return state
	}
}

func shortSHA(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}
