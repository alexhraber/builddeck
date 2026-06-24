# builddeck

A Buildkite terminal flight deck — a sleek, live-updating Go TUI that gives platform engineers and release captains a dense, navigable control surface for organizations, pipelines, builds, jobs, annotations, artifacts, and build health.

Think `htop` for Buildkite. Or `k9s` for your CI.

`b7k` is used as a short moniker, but the product, repository, documentation, and command identity are consistently **`builddeck`**.

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

## Current Features (Read-Only)

### Data Loaded
- **Organizations** — browse all orgs you have access to
- **Pipelines** — list pipelines per organization (with pagination)
- **Builds** — recent 25 builds per pipeline with build health summary
- **Build Detail** — full metadata: state, branch, commit, message, creator, timestamps, duration
- **Jobs** — all jobs for the selected build with state, label, agent, exit status
- **Annotations** — build annotations (info, warning, error, success styles)
- **Artifacts** — build artifacts with filename and size
- **Agents** — organization agent listing (available in API client)

### TUI
- **Three-pane layout** — orgs/pipelines | builds | detail+jobs+annotations+artifacts
- **Header bar** — product name, breadcrumb (org/pipeline/build), refresh status
- **Build health summary** — count of running/failed/passed/blocked builds with failure rate
- **State badges** — compact, color-coded state labels (PASS/FAIL/RUN/BLCK/etc.)
- **Active pane highlighting** — clear border on the focused pane
- **Live updates** — 5-second polling with in-flight request guards
- **Graceful degradation** — compact fallback for small terminals
- **Loading/error states** — visible without crashing

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
| `r` | Refresh all data |
| `/` | Search (placeholder — not yet implemented) |
| `?` | Toggle help |
| `q` | Quit |

### Refresh Behavior
- Polls selected pipeline builds every 5 seconds
- In-flight guards prevent duplicate concurrent requests
- Manual refresh (`r`) is always responsive
- Current selection preserved across refreshes when possible
- Falls back gracefully if selected items disappear

### Data Flow
- Changing organization resets pipelines, builds, jobs, annotations, artifacts
- Changing pipeline resets builds, jobs, annotations, artifacts
- Changing selected build fetches detail (if jobs missing), annotations, and artifacts
- Nil-pointer and index safety throughout

## Layout

```
┌────────────────────────────────────────────────────────────────────┐
│ builddeck  my-org / my-pipeline #42  ⟳ loading  14:32:01          │
├─────────────────┬──────────────────────────┬──────────────────────┤
│ Organizations   │ Builds                   │ Build Detail         │
│  ▸ MyOrg        │ [5 │ 1 running │ 3 ...] │  Number:  #42       │
│                 │                          │  State:   PASS       │
│ Pipelines       │ BUILD BRANCH COMMIT ... │  Branch:  main       │
│  ▸ my-pipeline  │  ▸ #42 main  abc1234 .. │  Commit:  abc1234    │
│    other-pipe   │    #41 main  def5678 .. │                      │
│                 │    #40 release 901abcd . │  Jobs                │
│                 │                          │   PASS Build [ag-1] │
│                 │                          │   RUN  Test  [ag-2] │
│                 │                          │                      │
│                 │                          │  Annotations         │
│                 │                          │   [ctx] Deploy done  │
│                 │                          │                      │
│                 │                          │  Artifacts           │
│                 │                          │   • log.txt (1.2KB)  │
├─────────────────┴──────────────────────────┴──────────────────────┤
│ Pane: Builds │ Updated: 14:32:01 │ ?:help q:quit r:refresh ...   │
└───────────────────────────────────────────────────────────────────┘
```

## Known Limitations

- **Read-only** — no retry, cancel, rebuild, unblock, or other mutating actions
- **REST API only** — GraphQL support planned for more efficient nested queries
- **No log tailing** — build log streaming is not yet supported
- **No artifact download** — only listing; download is not yet implemented
- **No config file** — authentication is via environment variable only
- **Limited pagination** — builds show first 25; pipelines and agents paginate up to 500
- **No search/filter** — `/` key shows placeholder message
- **Annotations are HTML-stripped** — rich content is flattened to text
- **Agent view not yet in TUI** — agent data is in the API client only

## Planned Next Features

- **Log tailing** — stream build/job logs in a sub-pane or split view
- **Retry / rebuild** — `R` to retry a job, `b` to rebuild a build
- **Cancel builds** — `x` to cancel a running build
- **Unblock jobs** — `u` to unblock a blocked job
- **Open in browser** — `o` to open the current resource in the Buildkite web UI
- **Artifact download** — `d` to download an artifact
- **Queue/agent saturation views** — dedicated pane for agent utilization and queue depth
- **Filtering/search** — `/` to filter builds, pipelines, or jobs by name/state
- **GraphQL dashboard snapshots** — efficient nested queries for dashboard views
- **Incident command mode** — focused view for diagnosing and resolving build failures
- **Config file** — token and preferences in `~/.config/builddeck/config.toml`
- **Build/pipeline search** — fuzzy search across all pipelines and builds

## Development

```bash
go build ./cmd/builddeck     # build
go fmt ./...                  # format
go test ./...                 # test
go vet ./...                  # vet
```

## License

MIT
