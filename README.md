# builddeck

[![Build status](https://badge.buildkite.com/286d3362748e755bd5497b6c947bf7fa043491f981dddbdef6.svg)](https://buildkite.com/alexhraber/builddeck)
[![Go version](https://img.shields.io/badge/Go-1.26-blue?logo=go)](https://go.dev/doc/devel/release#go1.26)

A Buildkite terminal flight deck — a sleek, live-updating Go TUI that gives platform engineers and release captains a dense, navigable control surface for organizations, pipelines, builds, jobs, annotations, artifacts, and build health.

Think `htop` for Buildkite. Or `k9s` for your CI.

`b7k` is used as a short moniker, but the product, repository, documentation, and command identity are consistently **`builddeck`**.

## Screenshot

![builddeck TUI showing organizations, builds, and build detail panes](assets/builddeck-screenshot.png)

## Why

Buildkite's web UI is powerful, but if you live in the terminal, context-switching to a browser to check build status is friction. `builddeck` brings real-time Buildkite visibility into your terminal with keyboard-driven navigation, live polling, and a dense pane-based layout.

## Install

```bash
go install github.com/alexhraber/builddeck/cmd/builddeck@latest
```

Or build from source:

```bash
git clone https://github.com/alexhraber/builddeck.git
cd builddeck
go build ./cmd/builddeck
```

## Run

```bash
export BUILDKITE_API_TOKEN="your-token-here"
builddeck
```

## Authentication

`builddeck` reads your Buildkite API token from the `BUILDKITE_API_TOKEN` environment variable.

Generate a token at: https://buildkite.com/user/api-access-tokens

Required scopes:
- `read_organizations`
- `read_pipelines`
- `read_builds`

If the token is missing, `builddeck` will exit immediately with a clear error message.

## Current Features

### Data Loaded
- **Organizations** — browse all orgs you have access to
- **Pipelines** — list pipelines per organization (with pagination)
- **Builds** — recent 25 builds per pipeline with build health summary
- **Build Detail** — full metadata: state, branch, commit, message, creator, timestamps, duration
- **Jobs** — all jobs for the selected build with state, label, agent, exit status
- **Logs** — tail the selected/top job log in a dedicated log pane
- **Annotations** — build annotations (info, warning, error, success styles)
- **Artifacts** — build artifacts with filename and size
- **Agents** — organization agent listing with queue saturation view (`a`)
- **Buildkite actions** — retry jobs, rebuild builds, cancel running builds, and unblock blocked jobs from the TUI
- **Open in browser** — `o` opens the current org/pipeline/build in the Buildkite web UI
- **Artifact download** — `d` downloads the first artifact for the selected build
- **Global search** — `ctrl+f` fuzzy search across all loaded organizations, pipelines, builds, and jobs
- **Config file** — token and preferences in `~/.config/builddeck/config.toml`
- **Saved filter presets** — `S` saves current filter, `P` loads a saved preset

### TUI
- **Three-pane layout** — orgs/pipelines | builds | detail+jobs+annotations+artifacts
- **Header bar** — product name, breadcrumb (org/pipeline/build), refresh status
- **Build health summary** — count of running/failed/passed/blocked builds with failure rate
- **State badges** — compact, color-coded state labels (PASS/FAIL/RUN/BLCK/etc.)
- **Active pane highlighting** — clear border on the focused pane
- **Read-only filtering** — `/` filters the active pane across pipelines, builds, or build jobs
- **Live updates** — smart adaptive polling (2s when active builds are running, 10s when idle) with in-flight request guards
- **Graceful degradation** — compact fallback for small terminals
- **Loading/error states** — visible without crashing
- **Agent saturation view** — dedicated view showing queue depth and agent utilization per queue
- **Filter presets** — save and load reusable filter patterns from config

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `←` / `h` | Previous pane |
| `→` / `l` | Next pane |
| `tab` | Next pane |
| `shift+tab` | Previous pane |
| `g` | Jump to top of active list |
| `G` | Jump to bottom of active list |
| `enter` | Select / drill down |
| `R` | Refresh all data |
| `L` | Tail selected/top job logs; press `L` or `esc` to return |
| `r` | Retry selected/top job |
| `b` | Rebuild selected/top build |
| `x` | Cancel selected/top running build |
| `u` | Unblock selected blocked job |
| `o` | Open current resource in browser |
| `d` | Download first artifact |
| `a` | Toggle agent/queue saturation view |
| `/` | Filter active pane |
| `ctrl+f` | Global search across all data |
| `S` | Save current filter as preset |
| `P` | Load a saved filter preset |
| `esc` / `enter` | Close filter input |
| `ctrl+u` | Clear filter input |
| `?` | Toggle help |
| `q` | Quit |

### Refresh Behavior
- **Smart Adaptive Polling**: Dynamically adjusts polling interval:
  - **2 seconds** when any build is in a non-terminal state (running, scheduled, etc.) to show live progress.
  - **10 seconds** when all builds are finished (idle state) to conserve Buildkite API rate limit budget.
- Status bar displays current refresh rate and a green `⚡LIVE` badge during active updates
- In-flight guards prevent duplicate concurrent requests
- Manual refresh (`R`) is always responsive and clears all caches
- Current selection preserved across refreshes when possible
- Falls back gracefully if selected items disappear

### Action Targeting
- On the org/pipeline pane, `L`, `r`, `b`, `x`, and `u` target the top/latest build for the selected pipeline
- On the builds pane, actions target the highlighted build
- On the detail pane, `r` and `u` target the highlighted job, while `b` and `x` target the selected build
- `x` only sends a cancel request for builds currently reported as running
- `u` only unblocks jobs with state `blocked`
- `o` opens the current context: pipeline on left pane, build on center pane, build on detail pane

### Data Flow
- Changing organization resets pipelines, builds, jobs, annotations, artifacts
- Changing pipeline resets builds, jobs, annotations, artifacts
- Changing selected build fetches detail (if jobs missing), annotations, and artifacts
- Filtering never mutates Buildkite data; it narrows already-loaded pipelines, builds, or jobs in the active pane
- Nil-pointer and index safety throughout

## Layout

```
┌───────────────────────────────────────────────────────────────────┐
│ builddeck  my-org / my-pipeline #42  ⟳ loading  14:32:01          │
├─────────────────┬──────────────────────────┬──────────────────────┤
│ Organizations   │ Builds                   │ Build Detail         │
│  ▸ MyOrg        │ [5 │ 1 running │ 3 ...]  │  Number:  #42        │
│                 │                          │  State:   PASS       │
│ Pipelines       │ BUILD BRANCH COMMIT ...  │  Branch:  main       │
│  ▸ my-pipeline  │  ▸ #42 main  abc1234 ..  │  Commit:  abc1234    │
│    other-pipe   │    #41 main  def5678 ..  │                      │
│                 │    #40 release 901abcd . │ Jobs                 │
│                 │                          │   PASS Build [ag-1]  │
│                 │                          │   RUN  Test  [ag-2]  │
│                 │                          │                      │
│                 │                          │  Annotations         │
│                 │                          │   [ctx] Deploy done  │
│                 │                          │                      │
│                 │                          │  Artifacts           │
│                 │                          │   • log.txt (1.2KB)  │
│                 │                          │     sha256:abc...321 │
├─────────────────┴──────────────────────────┴──────────────────────┤
│ Pane: Builds │ Updated: 14:32:01 │ ?:help q:quit R:refresh ...    │
└───────────────────────────────────────────────────────────────────┘
```

## Known Limitations

- **REST API only** — GraphQL support planned for more efficient nested queries
- **Limited pagination** — builds show first 25; pipelines and agents paginate up to 500
- **Annotations are HTML-stripped** — rich content is flattened to text
- **Artifact download** — downloads the first artifact; per-artifact selection not yet implemented
- **Global search** — searches only currently-loaded data, not all pipelines across orgs

## Planned Next Features

- **GraphQL dashboard snapshots** — efficient nested queries for dashboard views
- **Incident command mode** — focused view for diagnosing and resolving build failures

## Development

```bash
go build ./cmd/builddeck     # build
go fmt ./...                  # format
go test ./...                 # test
go vet ./...                  # vet
```

## License

MIT
