package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadMissingFile(t *testing.T) {
	// Use a temp dir so the config file doesn't exist
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("BUILDKITE_API_TOKEN", "test-token-from-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Token != "test-token-from-env" {
		t.Errorf("expected token from env, got %q", cfg.Token)
	}
	if cfg.DownloadDir != "." {
		t.Errorf("expected default download dir '.', got %q", cfg.DownloadDir)
	}
}

func TestLoadAndSaveConfig(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)
	t.Setenv("BUILDKITE_API_TOKEN", "")

	cfg := &Config{
		DownloadDir: "/tmp/artifacts",
		FilterPresets: []FilterPreset{
			{Name: "main builds", Query: "main", Pane: "builds"},
			{Name: "failed", Query: "failed", Pane: "builds"},
		},
	}

	if err := cfg.Save(); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file was created
	path := filepath.Join(tmp, "builddeck", "config.toml")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatalf("config file not created at %s", path)
	}

	// Load it back
	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load after save: %v", err)
	}

	if loaded.DownloadDir != "/tmp/artifacts" {
		t.Errorf("expected download_dir=/tmp/artifacts, got %q", loaded.DownloadDir)
	}
	if len(loaded.FilterPresets) != 2 {
		t.Fatalf("expected 2 filter presets, got %d", len(loaded.FilterPresets))
	}
	if loaded.FilterPresets[0].Name != "main builds" {
		t.Errorf("expected preset name 'main builds', got %q", loaded.FilterPresets[0].Name)
	}
	if loaded.FilterPresets[1].Query != "failed" {
		t.Errorf("expected preset query 'failed', got %q", loaded.FilterPresets[1].Query)
	}
}

func TestParseTOML(t *testing.T) {
	data := `
# Comment
token = "my-token"
download_dir = "/home/user/downloads"

[[filter_preset]]
name = "preset1"
query = "main"
pane = "builds"

[[filter_preset]]
name = "preset2"
query = "release"
pane = "pipelines"
`
	cfg := &Config{}
	if err := parseTOML(cfg, data); err != nil {
		t.Fatalf("parseTOML: %v", err)
	}

	if cfg.Token != "my-token" {
		t.Errorf("expected token my-token, got %q", cfg.Token)
	}
	if cfg.DownloadDir != "/home/user/downloads" {
		t.Errorf("expected /home/user/downloads, got %q", cfg.DownloadDir)
	}
	if len(cfg.FilterPresets) != 2 {
		t.Fatalf("expected 2 presets, got %d", len(cfg.FilterPresets))
	}
	if cfg.FilterPresets[0].Name != "preset1" || cfg.FilterPresets[0].Query != "main" {
		t.Errorf("preset0 mismatch: %+v", cfg.FilterPresets[0])
	}
	if cfg.FilterPresets[1].Pane != "pipelines" {
		t.Errorf("expected pane pipelines, got %q", cfg.FilterPresets[1].Pane)
	}
}

func TestEnvOverridesConfigToken(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmp)

	// Write config with a token
	dir := filepath.Join(tmp, "builddeck")
	os.MkdirAll(dir, 0o700)
	os.WriteFile(filepath.Join(dir, "config.toml"), []byte(`token = "config-token"`), 0o600)

	// Env token should override
	t.Setenv("BUILDKITE_API_TOKEN", "env-token")
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Token != "env-token" {
		t.Errorf("expected env-token to override, got %q", cfg.Token)
	}
}
