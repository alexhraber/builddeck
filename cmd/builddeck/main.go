package main

import (
	"fmt"
	"os"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/alexhraber/builddeck/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	token := os.Getenv("BUILDKITE_API_TOKEN")
	if token == "" {
		fmt.Fprintln(os.Stderr, "error: BUILDKITE_API_TOKEN environment variable is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generate a token at: https://buildkite.com/user/api-access-tokens")
		fmt.Fprintln(os.Stderr, "Required scopes: read_organizations, read_pipelines, read_builds")
		os.Exit(1)
	}

	client := buildkite.NewClient(token)
	model := tui.NewModel(client)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running builddeck: %v\n", err)
		os.Exit(1)
	}
}
