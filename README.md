# builddeck · `b7k`

[![Build status](https://badge.buildkite.com/286d3362748e755bd5497b6c947bf7fa043491f981dddbdef6.svg)](https://buildkite.com/alexhraber/builddeck)
[![Go version](https://img.shields.io/badge/Go-1.26-blue?logo=go)](https://go.dev/doc/devel/release#go1.26)
[![Go Report Card](https://goreportcard.com/badge/github.com/alexhraber/builddeck)](https://goreportcard.com/report/github.com/alexhraber/builddeck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

**`htop` for Buildkite. `k9s` for your CI.**

`builddeck` is a single-binary Go TUI that puts your entire Buildkite org on a keyboard-driven terminal dashboard — orgs, pipelines, builds, jobs, logs, agents, queues, annotations, artifacts, and build health, all updating live.

Stash the browser. Own your terminal.

---

## Screenshots

![Three-pane TUI showing orgs, builds, and build detail](assets/builddeck-screenshot1.png)

![Same pipeline in a different theme](assets/builddeck-screenshot2.png)

---

## Install

```bash
go install github.com/alexhraber/builddeck/cmd/builddeck@latest
```

Or clone and build:

```bash
git clone https://github.com/alexhraber/builddeck.git
cd builddeck
go build ./cmd/builddeck
```

Standalone binaries are available on the [releases page](https://github.com/alexhraber/builddeck/releases).

**Requirements:** Go 1.26+, a Buildkite API token, and a terminal with 256-color support. For full emoji, install a [Nerd Font v3](https://www.nerdfonts.com/) (e.g. `JetBrainsMono Nerd Font`).

---

## Quick Start

```bash
export BUILDKITE_API_TOKEN="bkt_abc123..."
builddeck
```

Token scopes needed: `read_organizations`, `read_pipelines`, `read_builds` (and optionally `write_builds` for retry/rebuild/cancel/unblock actions).

Generate one at [buildkite.com/user/api-access-tokens](https://buildkite.com/user/api-access-tokens).

---

## Features

### 📡 Data at Your Fingertips

| Surface | What you see |
|---|---|
| **Organizations** | All orgs your token has access to |
| **Pipelines** | Filterable list with custom emoji rendering |
| **Builds** | Last 25 builds per pipeline, with live health summary bar |
| **Build Detail** | State badge, branch, commit SHA, message, creator, timestamps, duration, **semver tag** |
| **Steps / Jobs** | All jobs with state, label, agent name, exit code |
| **Logs** | Full-screen log viewer with **source reference navigation** (`n`/`N` to hop `file:line` refs) |
| **Annotations** | Color-coded (info/warning/error/success) |
| **Artifacts** | Filename, size, **SHA256 checksum**, download picker |
| **Agents** | Per-org agent list with **queue saturation view** |
| **Custom Emoji** | Per-org Buildkite emoji, loaded live from the API |

### 🎮 Actions

| Key | Action |
|---|---|
| `r` | Retry highlighted step |
| `b` | Rebuild entire build |
| `x` | Cancel running build |
| `u` | Unblock blocked step |
| `o` | Open resource in browser |
| `ctrl+o` | Open repo in browser |
| `ctrl+d` | Open commit in browser |
| `L` | Tail job logs |
| `d` | Open artifact download picker |
| `a` | Toggle agent / queue view |
| `s` | Show stats overlay |

### 🧠 Smart Polling

- **Adaptive**: 2s when builds are running, 10s when idle — conserves API rate limits
- **Live mode** (`ctrl+l`): locks to 2s polling for demos and firefights
- **In-flight guards**: never duplicates concurrent API calls for the same scope
- **250ms debounce**: smooth scrolling when rapidly browsing builds
- **Selection preservation**: your cursor stays on the right build across refreshes

### 🎨 6 Themes

| Theme | Vibe |
|---|---|
| Tokyo Night | Default — deep blue-purple |
| Dracula | Dark vampiric purple |
| Gruvbox Dark | Warm retro ochre |
| Nord | Arctic frost blue |
| Monokai | Bright saturated neon |
| Cyberpunk | Hot pink / electric cyan |

Cycle with `Shift+O` → Theme.

### 📦 Artifact SHA256 Checksums

If your pipeline produces a `.sha256` companion artifact, builddeck downloads it, parses the hash, and displays it inline next to the matching artifact. This is the **real sha256sum output** — not a hash of the download URL — so it matches local verification exactly.

**Pipeline contract:**

```yaml
- label: ":lock: Checksum"
  command: |
    buildkite-agent artifact download builddeck /tmp/
    sha256sum /tmp/builddeck | tee builddeck.sha256
    buildkite-agent artifact upload builddeck.sha256
```

### 🔗 Semver Tag Display

If your pipeline has a tag step that creates git tags from conventional commits, builddeck queries Buildkite's tags API and displays the tag in the build detail pane.

### 🔍 Log-to-Source Navigation

Logs are parsed for `file:line` and `file:line:col` references. Press `n` / `N` to jump between source references, then `o` to open the file in your browser.

### 🌐 Emoji Renderer

Three-layer emoji engine:
1. **200+ Nerd Font PUA glyphs** for tech terms (docker, go, python, rust, aws, git, buildkite…)
2. **1902 Unicode emoji entries** extracted from Buildkite's own webapp emoji bundle
3. **Live custom emoji** loaded per-org from the Buildkite API

Graceful degradation if your terminal doesn't have Nerd Font support.

### 🔍 Global Search

`ctrl+f` to search across all loaded orgs, pipelines, builds, and jobs — capped at 50 results, color-coded by type.

### 💾 Filter Presets

Press `S` to save the current pane filter as a named preset, `P` to load one. Persisted to config file.

---

## Keybindings

### Navigation

| Key | Action |
|---|---|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `←` / `h` | Previous pane |
| `→` / `l` | Next pane |
| `tab` | Next pane |
| `shift+tab` | Previous pane |
| `g` / `G` | Top / bottom |
| `enter` | Select / drill down |

### Actions

| Key | Action |
|---|---|
| `R` | Refresh all data |
| `L` | Tail job log |
| `r` | Retry job |
| `b` | Rebuild build |
| `x` | Cancel running build |
| `u` | Unblock blocked job |
| `o` | Open in browser |
| `ctrl+o` | Open repo in browser |
| `ctrl+d` | Open commit in browser |
| `d` | Open artifact download picker |
| `a` | Toggle agent / queue view |
| `s` | Stats overlay |

### Search & Filter

| Key | Action |
|---|---|
| `/` | Filter active pane |
| `ctrl+f` | Global search |
| `ctrl+u` | Clear filter input |
| `S` | Save filter as preset |
| `P` | Load filter preset |

### Overlays

| Key | Action |
|---|---|
| `Shift+O` | Options (theme, refresh, density, sort) |
| `?` | Help overlay |
| `esc` | Close overlay / back |

### Log View

| Key | Action |
|---|---|
| `↑` / `↓` | Scroll |
| `n` / `N` | Next / previous source ref |
| `o` | Open source ref in browser |
| `L` / `esc` | Close log view |

### Artifact Picker

| Key | Action |
|---|---|
| `↑` / `↓` | Navigate |
| `enter` | Download selected |
| `a` | Download all |
| `esc` | Close |

---

## Configuration

`builddeck` stores preferences in `~/.config/builddeck/config.toml`:

```toml
[[filter_preset]]
name = "main-branch"
query = "main"
pane = "builds"

[[filter_preset]]
name = "failing"
query = "failed"
pane = "builds"
```

### Environment Variables

| Variable | Required | Purpose |
|---|---|---|
| `BUILDKITE_API_TOKEN` | Yes | API authentication |
| `BUILDKITE_BASE_URL` | No | Override API base URL (testing) |
| `BUILDKITE_DEBUG=1` | No | Log HTTP requests to stderr |

---

## Architecture

A single Go binary. No server, no database, no background processes.

```
cmd/builddeck/         CLI entrypoint
internal/
  buildkite/           REST API client + type definitions
  config/              TOML config loader
  tui/                 Bubble Tea model, update, view
    model.go           State machine + message types
    update.go          Event handling, API calls, keybindings (~1950 lines)
    views.go           All rendering (~1370 lines)
    styles.go          6 Lip Gloss themes
    keys.go            Key binding definitions
    search.go          Pane filtering + global search
    summary.go         Build health summary stats
    timefmt.go         Time / duration formatting
    emoji.go           Emoji bank (Nerd Font + Unicode + API)
    source_refs.go     Log-to-source reference parser
```

**Stack:** Go 1.26+, [Bubble Tea](https://github.com/charmbracelet/bubbletea),
[Lip Gloss](https://github.com/charmbracelet/lipgloss),
[Bubbles](https://github.com/charmbracelet/bubbles)

### Data Flow

```
Terminal ↔ Bubble Tea event loop (model/update/view)
            ↕ internal/tui (rendering + key handling)
            ↕ internal/buildkite (REST client)
            ↕ Buildkite API (api.buildkite.com)
              ↕ filesystem (~/.config/builddeck, artifact downloads)
```

### Design Decisions

| Decision | Rationale |
|---|---|
| Bubble Tea over web | Fast startup, keyboard-native, no browser needed |
| REST API only | Simpler than GraphQL; sufficient for feature set |
| Nerd Font icons | Rich visual language; graceful Unicode fallback |
| Adaptive polling | 2s active / 10s idle to conserve API rate limit budget |
| In-flight guards | No duplicate concurrent API calls for same scope |
| SHA256 from companion artifacts | Real content hash matching pipeline output |
| 250ms debounce on selection | Smooth scrolling without spiking API calls |
| Minimal TOML parser | No external dependency for simple config shape |

### Tertiary Validation System

builddeck enriches build data with results from **your pipeline's own steps** — Buildkite doesn't provide these:

- **Tag step** → semver tag from conventional commits → displayed next to commit SHA
- **Checksum step** → `.sha256` artifact → hash displayed next to matching artifact

This is builddeck's superpower: the TUI surfaces what your pipeline creates, not just what Buildkite manages.

---

## Security

- **Read-only by default**: retry/rebuild/cancel/unblock each require explicit keypress
- **Token in env only**: `BUILDKITE_API_TOKEN` never written to disk, never logged
- **HTTPS everywhere**: all API and artifact traffic over TLS
- **No secrets in binaries**: `govulncheck` + `gosec` run on every PR
- **Supply chain**: `go.sum` pinned, weekly dependency scan via scheduled pipeline

---

## Known Limitations

- **REST API only**: no GraphQL (planned for nested dashboard queries)
- **Limited pagination**: builds show first 25; pipelines and agents paginate up to 500
- **Annotations HTML-stripped**: rich content flattened to plain text
- **Global search**: searches currently-loaded data only, not all pipelines across orgs

---

## Troubleshooting

| Symptom | Fix |
|---|---|
| `BUILDKITE_API_TOKEN not set` | `export BUILDKITE_API_TOKEN=xxx` |
| 401 Unauthorized | Token expired or missing scopes — regenerate at buildkite.com/user/api-access-tokens |
| Emoji show as boxes | Install a [Nerd Font v3](https://www.nerdfonts.com/) |
| Artifact checksums missing | Add a Checksum step that uploads `.sha256` artifacts |
| "No builds" | Pipeline may have no recent builds; check branch filter |
| Terminal too small | Minimum ~80×24 |

---

## Development

```bash
go build ./cmd/builddeck    # build
go test ./...                # test
go vet ./...                 # validate
golangci-lint run ./...      # lint
gosec ./...                  # security scan
```

---

## License

MIT
