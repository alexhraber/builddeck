# Interfaces

## Contract Principles
- Prefer explicit types over `interface{}` / `any` for all API shapes.
- Every API call returns typed structs from `internal/buildkite/types.go`.
- Every failure path returns a `tea.Msg` error type visible in the TUI status bar.
- No mutating API calls without user confirmation (keybinding guard).

## Generated Contract Depth
- API/CLI contracts with request/response schemas are in `internal/buildkite/types.go`.
- All state is in-memory — no write path or storage ownership.
- Retry behavior is handled by Bubble Tea's polling loop (automatic re-fetch on next tick).
- Typed failure classes: `errMsg` for API errors, displayed in status bar.

## CLI Surface
| Flag | Env Var | Description | Required |
|---|---|---|---|
| (none) | `BUILDKITE_API_TOKEN` | Buildkite API token | Yes |
| (none) | `BUILDKITE_BASE_URL` | API base URL (for testing) | No (defaults to api.buildkite.com) |

## Internal Client API (internal/buildkite)
| Method | HTTP | Path | Returns |
|---|---|---|---|
| `GetOrgs()` | GET | `/v2/organizations` | `[]Organization` |
| `GetPipelines(org)` | GET | `/v2/organizations/{org}/pipelines` | `[]Pipeline` |
| `GetBuilds(org, pipeline)` | GET | `/v2/organizations/{org}/pipelines/{pipeline}/builds` | `[]Build` |
| `GetBuild(org, pipeline, number)` | GET | `/v2/organizations/{org}/pipelines/{pipeline}/builds/{number}` | `*Build` |
| `GetStepLog(org, pipeline, build, job)` | GET | `/v2/organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{id}/log?content=true` | `string` |
| `ListArtifacts(org, pipeline, build)` | GET | `/v2/organizations/{org}/pipelines/{pipeline}/builds/{number}/artifacts` | `[]Artifact` |
| `DownloadArtifactURL(org, pipeline, build, job, artifact)` | GET | `/v2/organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{id}/artifacts/{id}/download` | `string (URL)` |
| `ListEmoji(org)` | GET | `/v2/organizations/{org}/emoji` | `[]EmojiEntry` |
| `SearchBuilds(org, query)` | GET | `/v2/organizations/{org}/builds?q={query}` | `[]Build` |

## Key TUI Messages (internal/tui)
| Msg Type | Trigger | Effect |
|---|---|---|
| `orgsLoadedMsg` | App init or `R` refresh | Populates left pane |
| `pipelinesLoadedMsg` | Org selected | Populates center pane |
| `buildsLoadedMsg` | Pipeline selected | Populates right pane header |
| `buildDetailLoadedMsg` | Build selected | Populates jobs/annotations/artifacts |
| `logLoadedMsg` | `L` key or auto-select | Populates log pane |
| `errMsg` | Any API failure | Displayed in status bar |
| `tickMsg` | Timer (2s/10s) | Triggers polling refresh |
| `artifactChecksumMsg` | `.sha256` artifact download | Updates `Artifact.Checksum` field |

## Outbound Dependencies
| Dependency | Purpose | SLA | Timeout | Circuit-Breaker |
|---|---|---|---|---|
| api.buildkite.com | All Buildkite data | Best effort (SaaS) | 30s per request | N/A (client stops on auth failure) |

## Inbound Contracts
- CLI entrypoint: `cmd/builddeck/main.go` — validates token, inits logger, starts Bubble Tea
- No API / RPC entrypoints
- No event/webhook consumers

## Data Ownership
- **Source of truth**: Buildkite REST API — all data is ephemeral in-memory cache
- **Local state**: `~/.config/builddeck/config.toml` (filter presets, preferences)
- **No database, no persistent model, no migration**

## Key Struct Definitions

### Artifact (internal/buildkite/types.go)
```go
type Artifact struct {
    ID          string `json:"id"`
    Filename    string `json:"filename"`
    FileSize    int    `json:"file_size"`
    DownloadURL string `json:"download_url"`
    StepID      string `json:"job_id"`
    Checksum    string `json:"-"` // populated by loadArtifactChecksums()
}
```

### Build (internal/buildkite/types.go)
```go
type Build struct {
    Number    int          `json:"number"`
    State     string       `json:"state"`
    Branch    string       `json:"branch"`
    Commit    string       `json:"commit"`
    Message   string       `json:"message"`
    Creator   *Creator     `json:"creator"`
    Steps     []Step       `json:"jobs"`
    Pipeline  *PipelineRef `json:"pipeline"`
    StartedAt *time.Time   `json:"started_at"`
    FinishedAt *time.Time  `json:"finished_at"`
    Annotations []Annotation `json:"annotations"`
}
```

### Organization (internal/buildkite/types.go)
```go
type Organization struct {
    Slug string `json:"slug"`
    Name string `json:"name"`
}
```

## Error Taxonomy
```go
var (
    ErrMissingToken   = errors.New("BUILDKITE_API_TOKEN not set")
    ErrAPIError       = errors.New("api_error")
    ErrEmptyResponse  = errors.New("empty_response")
    ErrAuthFailed     = errors.New("authentication_failed")
    ErrDownloadFailed = errors.New("artifact_download_failed")
)
```

## Failure Semantics
| Failure Class | Retry/Backoff | Client Contract | Observability |
|---|---|---|---|
| Auth failure (401) | No retry | Exit with error | stderr + exit code 1 |
| Rate limit (429) | Auto-retry next poll cycle | Error in status bar | Status bar + next tick |
| Timeout (>=30s) | Auto-retry next poll cycle | Error in status bar | Status bar + next tick |
| Download failure | Manual retry via `d` key | Error in status bar | Status bar message |
| Parsing error | No retry | Skip field, continue | Graceful degradation |

## Timeout Budget
| Hop | Budget (ms) | Notes |
|---|---|---|
| TUI -> Buildkite API | 30000 | HTTP client timeout per request |
| Buildkite API -> internal | N/A | Response body streaming |

## Interface Versioning
- Version strategy: pinned to Buildkite REST API v2
- Backward-compatibility guarantees: all fields use `json:"-"` for unknown fields
- Deprecation window: N/A — client updated in lockstep with API changes