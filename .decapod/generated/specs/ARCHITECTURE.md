# Architecture

## Direction
Terminal UI (Go CLI with Bubble Tea)

## What This Project Is
`builddeck` is a Go CLI project that renders a Bubble Tea / Lip Gloss TUI. It reads from the Buildkite REST API and displays live build data in a three-pane layout. There is no server, no database, and no persistent state beyond a local config file.

Architectural principles:
- **Simplicity**: Each package has one responsibility. The TUI layer is separated from the API client and config.
- **Modularity**: `internal/buildkite` (API client), `internal/config` (config loading), `internal/tui` (Bubble Tea model/view/update) — clear dependency direction.
- **Reliability**: Graceful nil-pointer guards, loading/error states in every pane, in-flight request deduplication.

## Current Facts
- Runtime/languages: Go (1.26+)
- UI framework: Bubble Tea + Lip Gloss + Bubbles
- Buildkite API client: hand-written REST client in `internal/buildkite`
- Render layout: Three-pane with overlays (artifact picker, preset picker, help, search)
- Auth: `BUILDKITE_API_TOKEN` env var (read at startup, not a long-running session)
- Config: `~/.config/builddeck/config.toml` via `internal/config`

## Architecture Map
```
cmd/builddeck/           — CLI entrypoint (flag parsing, token check)
internal/
  buildkite/
    client.go            — REST API client (GetOrgs, GetPipelines, GetBuilds, ...)
    types.go             — Buildkite API type definitions
    client_test.go       — HTTP mock tests
  config/
    config.go            — Config file loading and merging with env
    config_test.go
  tui/
    model.go             — Bubble Tea Model struct + Msg types
    update.go            — update() — all commands, key handling, API calls
    views.go             — view() — all rendering (panes, overlays, status bar)
    emoji.go             — Nerd Font + Unicode emoji bank and renderer
    styles.go            — Lip Gloss style definitions
    model_test.go
    search_test.go
```

## Data Flows
1. **Startup**: `main()` loads config, creates API client, initializes Bubble Tea program.
2. **Org load**: On init, `fetchOrgsCmd` calls `client.GetOrgs()`, dispatches `orgsLoadedMsg`.
3. **Pipeline load**: On org select, `client.GetPipelines()` fetches pipelines for that org.
4. **Build load**: On pipeline select, `client.GetBuilds()` fetches recent 25 builds.
5. **Build detail**: On build select, fetches jobs, annotations, and artifacts in parallel.
6. **Logs**: `client.GetStepLog()` fetches individual job log with content query param + raw text fallback.
7. **Polling**: `tickMsg` fires every 2s (active) or 10s (idle), re-fetching builds and detail.
8. **Checksums**: `loadArtifactChecksums()` downloads `.sha256` artifacts and parses the hash.

## Strongest Existing Primitives
- **`internal/buildkite.Client`**: Typed REST client with configurable base URL, automatic pagination, and structured error handling.
- **`internal/tui.Model`**: Central state tree with nil-safe accessors (`selectedOrg()`, `selectedPipeline()`, etc.).
- **`internal/tui/renderEmoji()`**: Grapheme-aware emoji renderer supporting Nerd Font PUA glyphs + Unicode emoji.
- **`internal/tui/emoji.go`**: 200+ entry emoji bank for Buildkite custom emoji and language icons.

## Topology
```text
User Terminal
  ↕ Bubble Tea event loop (model/update/view)
  ↕ internal/tui (all rendering + key handling)
  ↕ internal/buildkite (REST client)
  ↕ Buildkite API (api.buildkite.com)
    ↕ filesystem (~/.config/builddeck/config.toml, artifact downloads)
```

## Store Boundaries
```mermaid
flowchart LR
  BK[Buildkite API] --> C[Client Cache]
  C --> M[TUI Model State]
  C --> FS[Local Config + Downloads]
```

No write store — all state is cached in-memory in the Bubble Tea Model.

## Happy Path Sequence
```text
User runs builddeck
  → Token present → load config
  → Fetch orgs → render left pane
  → Select org → fetch pipelines
  → Select pipeline → fetch builds
  → Select build → fetch jobs, annotations, artifacts (incl .sha256)
  → d → artifact picker overlay → enter → download selected artifact
  → L → log pane → view real-time log output
  → q → quit
```

## Error Path
```mermaid
sequenceDiagram
  participant TUI
  participant Client
  participant API
  TUI->>Client: GetBuilds()
  Client->>API: GET /builds
  API--xClient: 401 / 429 / timeout
  Client-->>TUI: errorMsg
  TUI-->>TUI: Show error in status bar, retry on next tick
```

All API errors are non-fatal. The TUI shows the error in the status bar and retries on the next polling cycle.

## Execution Path
- Ingress: CLI arg parsing (none currently) → env var token check → config load
- Core: Bubble Tea event loop — `Init()` fetches orgs, `Update()` handles messages + keys, `View()` renders panes
- Data: API responses deserialized directly into typed structs, cached in model, filtered in-place
- Verification: `t.Log()` output captured during testing, CI gates via Buildkite pipeline

## Concurrency and Runtime Model
- Execution model: Single-threaded Bubble Tea event loop (goroutine-safe via `tea.Cmd`)
- Isolation boundaries: All API calls are `tea.Cmd` functions that run in goroutines, results dispatched as `tea.Msg`
- Backpressure strategy: `isFetching` guard prevents concurrent duplicate requests
- Shared state synchronization: Bubble Tea guarantees serial access to `Model.Update()`

## Deployment Topology
- Runtime units: Single binary (`builddeck`) — no deployment needed
- Distribution: GitHub releases via `gh release create/upload`
- Installation: `go install` or download binary from releases page
- Rollback: Reinstall previous version

## Data and Contracts
- Inbound contracts: `BUILDKITE_API_TOKEN` env var, `~/.config/builddeck/config.toml`
- Outbound dependencies: Buildkite REST API at `https://api.buildkite.com`
- Data ownership boundaries: All user data owned by Buildkite; builddeck only reads/caches
- Schema evolution: Buildkite types evolve upstream — client handles gracefully via `json:"-"` skip fields

## ADR Register
| ADR | Title | Status | Rationale | Date |
|---|---|---|---|---|
| ADR-001 | Bubble Tea for TUI | Accepted | Mature Go TUI framework with Lip Gloss styling | 2026-01 |
| ADR-002 | REST API only | Accepted | Simpler than GraphQL, sufficient for MVP scope | 2026-01 |
| ADR-003 | Nerd Font icons | Accepted | Rich visual language for build states and languages | 2026-03 |
| ADR-004 | SHA256 from .sha256 artifacts | Accepted | Real content hash matching pipeline checksum | 2026-07 |

## Delivery Plan (first 3 slices)
- Slice 1 (MVP): Org/pipeline/build browsing, log viewing, basic polling
- Slice 2: Artifact download picker, SHA256 checksums, global search
- Slice 3: Filter presets, agent saturation view, help overlay

## Risks and Mitigations
| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Buildkite API rate limiting | Medium | Medium | Adaptive polling (2s/10s), in-flight guards |
| Nerd Font missing | Low | Medium | Unicode fallback, tests for render width |
| API schema changes | Low | High | Parse leniently, test with live data |
| Terminal size too small | Medium | Low | Graceful degradation, minimum size check |