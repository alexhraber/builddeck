package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents the builddeck configuration.
type Config struct {
	Token         string
	FilterPresets []FilterPreset
	DownloadDir   string
}

// FilterPreset is a saved filter pattern with a label.
type FilterPreset struct {
	Name  string
	Query string
	Pane  string // "pipelines", "builds", "jobs"
}

// configDir returns the XDG or default config directory.
func configDir() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "builddeck")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "builddeck")
}

// ConfigPath returns the path to the config file.
func ConfigPath() string {
	return filepath.Join(configDir(), "config.toml")
}

// Load reads the config file and environment. Env vars override config file values.
func Load() (*Config, error) {
	cfg := &Config{
		DownloadDir: ".",
	}

	path := ConfigPath()
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("reading config %s: %w", path, err)
	}

	if err == nil {
		if parseErr := parseTOML(cfg, string(data)); parseErr != nil {
			return nil, fmt.Errorf("parsing config %s: %w", path, parseErr)
		}
	}

	// Environment variables override config file
	if token := os.Getenv("BUILDKITE_API_TOKEN"); token != "" {
		cfg.Token = token
	}
	return cfg, nil
}

// Save writes the config (without token for security) to disk.
func (c *Config) Save() error {
	dir := configDir()
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	var b strings.Builder
	b.WriteString("# builddeck configuration\n")
	b.WriteString("# Token is read from BUILDKITE_API_TOKEN env var (not stored here)\n\n")

	if c.DownloadDir != "" && c.DownloadDir != "." {
		b.WriteString(fmt.Sprintf("download_dir = %q\n", c.DownloadDir))
	}

	if len(c.FilterPresets) > 0 {
		b.WriteString("\n")
		for _, fp := range c.FilterPresets {
			b.WriteString("[[filter_preset]]\n")
			b.WriteString(fmt.Sprintf("name = %q\n", fp.Name))
			b.WriteString(fmt.Sprintf("query = %q\n", fp.Query))
			b.WriteString(fmt.Sprintf("pane = %q\n", fp.Pane))
			b.WriteString("\n")
		}
	}

	return os.WriteFile(ConfigPath(), []byte(b.String()), 0o600)
}

// parseTOML is a minimal TOML parser sufficient for our config format.
// It handles simple key=value pairs and [[array_of_tables]].
func parseTOML(cfg *Config, data string) error {
	var currentPreset *FilterPreset
	inPreset := false

	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		if line == "[[filter_preset]]" {
			if inPreset && currentPreset != nil {
				cfg.FilterPresets = append(cfg.FilterPresets, *currentPreset)
			}
			currentPreset = &FilterPreset{}
			inPreset = true
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		value = unquote(value)

		if inPreset && currentPreset != nil {
			switch key {
			case "name":
				currentPreset.Name = value
			case "query":
				currentPreset.Query = value
			case "pane":
				currentPreset.Pane = value
			}
		} else {
			switch key {
			case "token":
				cfg.Token = value
			case "download_dir":
				cfg.DownloadDir = value
			}
		}
	}

	if inPreset && currentPreset != nil {
		cfg.FilterPresets = append(cfg.FilterPresets, *currentPreset)
	}

	return nil
}

func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}
