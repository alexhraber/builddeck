package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up           key.Binding
	Down         key.Binding
	Left         key.Binding
	Right        key.Binding
	Tab          key.Binding
	ShiftTab     key.Binding
	Enter        key.Binding
	Top          key.Binding
	Bottom       key.Binding
	Search       key.Binding
	Refresh      key.Binding
	Help         key.Binding
	Logs         key.Binding
	LiveMode     key.Binding
	AgentStats   key.Binding
	RetryStep    key.Binding
	Cancel       key.Binding
	Quit         key.Binding
	Unblock      key.Binding
	OpenBrowser  key.Binding
	Download     key.Binding
	Agents       key.Binding
	GlobalSearch key.Binding
	SavePreset   key.Binding
	LoadPreset   key.Binding
	Options      key.Binding
	OpenRepo     key.Binding
	OpenCommit   key.Binding
}

var keys = keyMap{
	Up: key.NewBinding(
		key.WithKeys("up", "k"),
		key.WithHelp("↑/k", "up"),
	),
	Down: key.NewBinding(
		key.WithKeys("down", "j"),
		key.WithHelp("↓/j", "down"),
	),
	Left: key.NewBinding(
		key.WithKeys("left", "h"),
		key.WithHelp("←/h", "prev pane"),
	),
	Right: key.NewBinding(
		key.WithKeys("right", "l"),
		key.WithHelp("→/l", "next pane"),
	),
	Tab: key.NewBinding(
		key.WithKeys("tab"),
		key.WithHelp("tab", "next pane"),
	),
	ShiftTab: key.NewBinding(
		key.WithKeys("shift+tab"),
		key.WithHelp("shift+tab", "prev pane"),
	),
	Enter: key.NewBinding(
		key.WithKeys("enter"),
		key.WithHelp("enter", "select"),
	),
	Top: key.NewBinding(
		key.WithKeys("g"),
		key.WithHelp("g", "top of list"),
	),
	Bottom: key.NewBinding(
		key.WithKeys("G"),
		key.WithHelp("G", "bottom of list"),
	),
	Search: key.NewBinding(
		key.WithKeys("/"),
		key.WithHelp("/", "search"),
	),
	Refresh: key.NewBinding(
		key.WithKeys("R"),
		key.WithHelp("R", "refresh"),
	),
	Help: key.NewBinding(
		key.WithKeys("?"),
		key.WithHelp("?", "help"),
	),
	Logs: key.NewBinding(
		key.WithKeys("L"),
		key.WithHelp("L", "logs"),
	),
	LiveMode: key.NewBinding(
		key.WithKeys("ctrl+l"),
		key.WithHelp("ctrl+l", "live mode"),
	),
	AgentStats: key.NewBinding(
		key.WithKeys("s"),
		key.WithHelp("s", "agent stats"),
	),
	RetryStep: key.NewBinding(
		key.WithKeys("r"),
		key.WithHelp("r", "rebuild/rerun"),
	),
	Cancel: key.NewBinding(
		key.WithKeys("x"),
		key.WithHelp("x", "cancel"),
	),
	Quit: key.NewBinding(
		key.WithKeys("q", "ctrl+c"),
		key.WithHelp("q", "quit"),
	),
	Unblock: key.NewBinding(
		key.WithKeys("u"),
		key.WithHelp("u", "unblock step"),
	),
	OpenBrowser: key.NewBinding(
		key.WithKeys("o"),
		key.WithHelp("o", "open in browser"),
	),
	Download: key.NewBinding(
		key.WithKeys("d"),
		key.WithHelp("d", "download artifact"),
	),
	Agents: key.NewBinding(
		key.WithKeys("a"),
		key.WithHelp("a", "agents view"),
	),
	GlobalSearch: key.NewBinding(
		key.WithKeys("ctrl+f"),
		key.WithHelp("ctrl+f", "global search"),
	),
	SavePreset: key.NewBinding(
		key.WithKeys("S"),
		key.WithHelp("S", "save filter preset"),
	),
	LoadPreset: key.NewBinding(
		key.WithKeys("P"),
		key.WithHelp("P", "load filter preset"),
	),
	Options: key.NewBinding(
		key.WithKeys("O"),
		key.WithHelp("Shift+O", "options"),
	),
	OpenRepo: key.NewBinding(
		key.WithKeys("ctrl+o"),
		key.WithHelp("ctrl+o", "open repo"),
	),
	OpenCommit: key.NewBinding(
		key.WithKeys("ctrl+d"),
		key.WithHelp("ctrl+d", "open commit"),
	),
}

func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{
		k.Up, k.Down, k.Left, k.Right, k.Tab, k.ShiftTab,
		k.Enter, k.Top, k.Bottom, k.Search, k.Refresh,
		k.LiveMode, k.Logs, k.AgentStats, k.RetryStep, k.Cancel, k.Unblock,
		k.OpenBrowser, k.Download, k.Agents, k.GlobalSearch,
		k.OpenCommit, k.OpenRepo, k.SavePreset, k.LoadPreset, k.Options, k.Help, k.Quit,
	}
}

func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Tab, k.ShiftTab, k.Enter},
		{k.Top, k.Bottom, k.Search, k.GlobalSearch, k.Refresh, k.LiveMode, k.Logs, k.AgentStats},
		{k.RetryStep, k.Cancel, k.Unblock},
		{k.OpenBrowser, k.Download, k.Agents, k.Options, k.OpenRepo, k.OpenCommit},
		{k.SavePreset, k.LoadPreset},
		{k.Help, k.Quit},
	}
}
