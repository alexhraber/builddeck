package tui

import (
	"github.com/charmbracelet/lipgloss"
)

type Theme struct {
	Name           string
	BorderActive   string
	BorderInactive string
	Title          string
	Subtitle       string
	Dim            string
	SelectedBg     string
	SelectedFg     string
	NormalFg       string
	Accent         string
	Success        string
	Warning        string
	Failure        string
	Info           string
	Blocked        string
}

var Themes = []Theme{
	{
		Name:           "Tokyo Night",
		BorderActive:   "#7dcfff",
		BorderInactive: "#24283b",
		Title:          "#bb9af3",
		Subtitle:       "#7aa2f7",
		Dim:            "#565f89",
		SelectedBg:     "#414868",
		SelectedFg:     "#e0af68",
		NormalFg:       "#a9b1d6",
		Accent:         "#ff9e64",
		Success:        "#9ece6a",
		Warning:        "#e0af68",
		Failure:        "#f7768e",
		Info:           "#7dcfff",
		Blocked:        "#bb9af3",
	},
	{
		Name:           "Dracula",
		BorderActive:   "#ff79c6",
		BorderInactive: "#44475a",
		Title:          "#bd93f9",
		Subtitle:       "#8be9fd",
		Dim:            "#6272a4",
		SelectedBg:     "#44475a",
		SelectedFg:     "#50fa7b",
		NormalFg:       "#f8f8f2",
		Accent:         "#ffb86c",
		Success:        "#50fa7b",
		Warning:        "#f1fa8c",
		Failure:        "#ff5555",
		Info:           "#8be9fd",
		Blocked:        "#bd93f9",
	},
	{
		Name:           "Gruvbox Dark",
		BorderActive:   "#fe8019",
		BorderInactive: "#3c3836",
		Title:          "#fabd2f",
		Subtitle:       "#83a598",
		Dim:            "#928374",
		SelectedBg:     "#504945",
		SelectedFg:     "#b8bb26",
		NormalFg:       "#ebdbb2",
		Accent:         "#fe8019",
		Success:        "#b8bb26",
		Warning:        "#fabd2f",
		Failure:        "#fb4934",
		Info:           "#83a598",
		Blocked:        "#d3869b",
	},
	{
		Name:           "Nord",
		BorderActive:   "#88c0d0",
		BorderInactive: "#3b4252",
		Title:          "#81a1c1",
		Subtitle:       "#8fbcbb",
		Dim:            "#4c566a",
		SelectedBg:     "#434c5e",
		SelectedFg:     "#a3be8c",
		NormalFg:       "#d8dee9",
		Accent:         "#ebcb8b",
		Success:        "#a3be8c",
		Warning:        "#ebcb8b",
		Failure:        "#bf616a",
		Info:           "#88c0d0",
		Blocked:        "#b48ead",
	},
	{
		Name:           "Monokai",
		BorderActive:   "#a6e22e",
		BorderInactive: "#3e3d32",
		Title:          "#f92672",
		Subtitle:       "#66d9ef",
		Dim:            "#75715e",
		SelectedBg:     "#49483e",
		SelectedFg:     "#a6e22e",
		NormalFg:       "#f8f8f2",
		Accent:         "#fd971f",
		Success:        "#a6e22e",
		Warning:        "#e6db74",
		Failure:        "#f92672",
		Info:           "#66d9ef",
		Blocked:        "#ae81ff",
	},
	{
		Name:           "Cyberpunk",
		BorderActive:   "#ff007f",
		BorderInactive: "#2b002b",
		Title:          "#00f0ff",
		Subtitle:       "#ff007f",
		Dim:            "#5a185a",
		SelectedBg:     "#ff007f",
		SelectedFg:     "#001015",
		NormalFg:       "#00ff99",
		Accent:         "#fffb00",
		Success:        "#00ff99",
		Warning:        "#fffb00",
		Failure:        "#ff007f",
		Info:           "#00f0ff",
		Blocked:        "#bd00ff",
	},
}

var (
	activeTheme Theme

	borderStyle       lipgloss.Style
	activeBorderStyle lipgloss.Style
	titleStyle        lipgloss.Style
	subtitleStyle     lipgloss.Style
	dimStyle          lipgloss.Style
	selectedItemStyle lipgloss.Style
	normalItemStyle   lipgloss.Style
	statusStyle       lipgloss.Style
	helpStyle         lipgloss.Style
	errorStyle        lipgloss.Style
	loadingStyle      lipgloss.Style
	sourceRefStyle    lipgloss.Style
	sourceRefSelStyle lipgloss.Style
)

func initStyles(theme Theme) {
	activeTheme = theme

	borderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderInactive))

	activeBorderStyle = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.BorderActive))

	titleStyle = lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color(theme.Title))

	subtitleStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Subtitle))

	dimStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Dim))

	selectedItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SelectedFg)).
		Bold(true)

	normalItemStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.NormalFg))

	statusStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Dim))

	helpStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Accent))

	errorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Failure))

	loadingStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Warning))

	sourceRefStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.Accent)).
		Underline(true)

	sourceRefSelStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.SelectedFg)).
		Background(lipgloss.Color(theme.SelectedBg)).
		Bold(true)
}

func init() {
	initStyles(Themes[0])
}

func stateColor(state string) lipgloss.Color {
	switch state {
	case "passed":
		return lipgloss.Color(activeTheme.Success)
	case "failed", "failing":
		return lipgloss.Color(activeTheme.Failure)
	case "running", "active":
		return lipgloss.Color(activeTheme.Info)
	case "scheduled", "waiting":
		return lipgloss.Color(activeTheme.Warning)
	case "canceled", "cancelled":
		return lipgloss.Color(activeTheme.Dim)
	case "skipped":
		return lipgloss.Color(activeTheme.Dim)
	case "blocked":
		return lipgloss.Color(activeTheme.Blocked)
	case "not_run":
		return lipgloss.Color(activeTheme.Dim)
	case "timed_out":
		return lipgloss.Color(activeTheme.Failure)
	case "broken":
		return lipgloss.Color(activeTheme.Failure)
	default:
		return lipgloss.Color(activeTheme.NormalFg)
	}
}

func stateBadge(state string) string {
	color := stateColor(state)
	return lipgloss.NewStyle().
		Background(color).
		Foreground(lipgloss.Color(activeTheme.BorderInactive)).
		Bold(true).
		Padding(0, 1).
		Render(stateLabel(state))
}

func stateLabel(state string) string {
	switch state {
	case "passed":
		return "PASS"
	case "failed":
		return "FAIL"
	case "running":
		return "EXEC"
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
