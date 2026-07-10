package main

import (
	"fmt"
	"os"

	"github.com/alexhraber/builddeck/internal/buildkite"
	"github.com/alexhraber/builddeck/internal/config"
	"github.com/alexhraber/builddeck/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

// version is set at build time via ldflags (-X main.version=...)
var version = "dev"

func main() {
	// Print version if requested (before token check)
	if len(os.Args) > 1 && os.Args[1] == "--version" {
		fmt.Printf("builddeck %s\n", version)
		return
	}

	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to load config: %v\n", err)
		cfg = &config.Config{}
	}

	// Token priority: env var > config file
	token := cfg.Token
	if envToken := os.Getenv("BUILDKITE_API_TOKEN"); envToken != "" {
		token = envToken
	}

	if token == "" {
		fmt.Fprintln(os.Stderr, "error: BUILDKITE_API_TOKEN environment variable is required")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "Generate a token at: https://buildkite.com/user/api-access-tokens")
		fmt.Fprintln(os.Stderr, "Required scopes: read_organizations, read_pipelines, read_builds")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintf(os.Stderr, "Or add token to config file: %s\n", config.ConfigPath())
		os.Exit(1)
	}

	client := buildkite.NewClient(token)
	model := tui.NewModelWithConfig(client, cfg)

	p := tea.NewProgram(model, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "error running builddeck: %v\n", err)
		os.Exit(1)
	}
}
