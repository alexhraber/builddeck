# builddeck

A Buildkite terminal flight deck — a sleek, live-updating Go TUI that gives platform engineers and release captains a dense, navigable control surface for organizations, pipelines, builds, jobs, and build health.

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

## MVP Features

- **Organizations & Pipelines** — browse all orgs and pipelines you have access to
- **Recent Builds** — list the 25 most recent builds for any pipeline
- **Build Details** — view build metadata (state, branch, commit, creator, duration, timestamps)
- **Job List** — see all jobs for a build with state, label, agent, and exit status
- **Live Updates** — data refreshes every 5 seconds automatically
- **Pane Navigation** — tab between Orgs/Pipelines, Builds, and Detail panes
- **Read-Only** — no mutating actions; safe to explore without risk

## Keybindings

| Key | Action |
|-----|--------|
| `↑` / `k` | Move up |
| `↓` / `j` | Move down |
| `tab` | Next pane |
| `shift+tab` | Previous pane |
| `enter` | Select / drill down |
| `r` | Refresh all data |
| `?` | Toggle help |
| `q` | Quit |

## Layout

```
┌─────────────────┬──────────────────────────┬──────────────────────┐
│ Organizations   │ Builds                   │ Build Detail         │
│  ▸ MyOrg        │  BUILD BRANCH COMMIT ... │  Number:  #42        │
│                 │  ▸ #42  main    abc1234   │  State:   PASSED     │
│ Pipelines       │    #41  main    def5678  │  Branch:  main       │
│  ▸ my-pipeline  │    #40  release 901abcd  │  Commit:  abc1234    │
│    other-pipe   │                          │                      │
│                 │                          │  Jobs                │
│                 │                          │   PASSED  Build      │
│                 │                          │   RUNNING  Test      │
├─────────────────┴──────────────────────────┴──────────────────────┤
│ Pane: Orgs/Pipes  │  Updated: 14:32:01  │  [?] help  [q] quit   │
└───────────────────────────────────────────────────────────────────┘
```

## Limitations

- **Read-only** — no retry, cancel, rebuild, unblock, or other mutating actions yet
- **REST API only** — GraphQL support planned for more efficient nested queries
- **No log tailing** — build log streaming is not yet supported
- **No artifact browsing** — artifact download is not yet supported
- **No config file** — authentication is via environment variable only
- **Basic pagination** — shows the first page of results; scroll pagination is not yet implemented
- **No search/filter** — no way to filter builds or pipelines by name or state

## Planned Next Features

- `x` — Cancel build
- `R` — Retry job
- `b` — Rebuild build
- `u` — Unblock job
- `o` — Open in browser
- `l` — Tail logs
- `d` — Download artifact
- Queue/agent saturation views
- Bottleneck diagnosis
- Incident command mode
- GraphQL support for efficient nested dashboard queries
- Config file for token and preferences
- Build/pipeline search and filter

## License

MIT
