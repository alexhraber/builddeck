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
| `--version` | — | Print version and exit | No |
| (none) | `BUILDKITE_API_TOKEN` | Buildkite API token | Yes |
| (none) | `BUILDKITE_BASE_URL` | API base URL (for testing) | No (defaults to api.buildkite.com) |
| (none) | `BUILDKITE_DEBUG=1` | Log HTTP requests to stderr | No |

## Internal Client API (internal/buildkite)
| Method | HTTP | Path | Returns |
|---|---|---|---|
| `ListOrganizations(ctx)` | GET | `/v2/organizations` | `[]Organization` |
| `ListPipelines(ctx, orgSlug)` | GET | `/v2/organizations/{org}/pipelines` | `[]Pipeline` (paginated up to 500) |
| `ListBuilds(ctx, orgSlug, pipeSlug)` | GET | `/v2/organizations/{org}/pipelines/{pipe}/builds` | `[]Build` (25 per page) |
| `GetBuild(ctx, orgSlug, pipeSlug, buildNum)` | GET | `/.../builds/{number}` | `*Build` |
| `GetTagArtifact(ctx, orgSlug, pipeSlug, buildNum)` | GET | Downloads `.tag` artifact | `string` (version tag) |
| `RebuildBuild(ctx, orgSlug, pipeSlug, buildNum)` | PUT | `/.../builds/{number}/rebuild` | `*Build` |
| `CancelBuild(ctx, orgSlug, pipeSlug, buildNum)` | PUT | `/.../builds/{number}/cancel` | `*Build` |
| `ListAgents(ctx, orgSlug)` | GET | `/v2/organizations/{org}/agents` | `[]Agent` (paginated up to 500) |
| `ListAnnotations(ctx, orgSlug, pipeSlug, buildNum)` | GET | `/.../builds/{number}/annotations` | `[]Annotation` |
| `ListArtifacts(ctx, orgSlug, pipeSlug, buildNum)` | GET | `/.../builds/{number}/artifacts` | `[]Artifact` (paginated) |
| `GetStepLog(ctx, orgSlug, pipeSlug, buildNum, stepID)` | GET | `/.../jobs/{id}/log?content=true` | `*StepLog` (with raw text fallback) |
| `RetryStep(ctx, orgSlug, pipeSlug, buildNum, stepID)` | PUT | `/.../jobs/{id}/retry` | error |
| `UnblockStep(ctx, orgSlug, pipeSlug, buildNum, stepID)` | PUT | `/.../jobs/{id}/unblock` | error |
| `DownloadArtifactURL(ctx, orgSlug, pipeSlug, buildNum, stepID, artifactID)` | GET | `/.../jobs/{id}/artifacts/{id}/download` | `string` (redirect URL) |
| `ListEmojis(ctx, orgSlug)` | GET | `/v2/organizations/{org}/emojis` | `[]EmojiEntry` |
| `GetTagsForCommit(ctx, orgSlug, pipeSlug, commitSHA)` | GET | `/.../builds/{sha}/tags` | `[]Tag` |

## Key TUI Messages (internal/tui)
| Msg Type | Trigger | Effect |
|---|---|---|
| `orgsLoadedMsg` | App init or `R` refresh | Populates left pane |
| `pipelinesLoadedMsg` | Org selected | Populates center pane |
| `buildsLoadedMsg` | Pipeline selected | Populates right pane header |
| `buildDetailMsg` | Build selected (250ms debounced) | Populates jobs/annotations/artifacts/tags |
| `annotationsLoadedMsg` | Build detail loaded | Populates annotations section |
| `artifactsLoadedMsg` | Build detail loaded | Populates artifacts section |
| `agentsLoadedMsg` | `a` keypress | Populates agent/queue saturation view |
| `emojisLoadedMsg` | Org selected | Loads per-org custom emoji |
| `logLoadedMsg` | `L` key or auto-select | Populates log pane |
| `buildActionMsg` | Retry/rebuild/cancel/unblock result | Updates build state |
| `artifactDownloadMsg` | Artifact download completes | Saves file to disk |
| `artifactChecksumMsg` | `.sha256` artifact download | Updates `Artifact.Checksum` field |
| `artifactTagMsg` | `.tag` artifact download | Updates build tag display |
| `buildSelectionDebounceMsg` | 250ms debounce timer | Triggers build detail fetch |
| `errMsg` | Any API failure | Displayed in status bar |
| `tickMsg` | Timer (2s/10s/30s/disabled) | Triggers polling refresh |

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
- **Local state**: `~/.config/builddeck/config.toml` (filter presets, UI preferences)
- **No database, no persistent model, no migration**

## Key Struct Definitions

### Artifact (internal/buildkite/types.go)
```go
type Artifact struct {
    ID          string `json:"id"`
    Filename    string `json:"filename"`
    FileSize    int    `json:"file_size"`
    Dirname     string `json:"dirname"`
    ContentType string `json:"mime_type"`
    DownloadURL string `json:"download_url"`
    StepID      string `json:"job_id"`
    State       string `json:"state"`
    WebURL      string `json:"url"`
    Checksum    string `json:"-"`   // populated by loadArtifactChecksums()
    Tag         string `json:"-"`   // populated by loadArtifactTags()
    CreatedAt   time.Time `json:"created_at"`
}
```

### Build (internal/buildkite/types.go)
```go
type Build struct {
    ID         string       `json:"id"`
    Number     int          `json:"number"`
    State      string       `json:"state"`
    Branch     string       `json:"branch"`
    Tag        string       `json:"tag"`
    Commit     string       `json:"commit"`
    Message    string       `json:"message"`
    Creator    *Creator     `json:"creator"`
    Steps      []Step       `json:"jobs"`
    Pipeline   *PipelineRef `json:"pipeline"`
    WebURL     string       `json:"web_url"`
    StartedAt  *time.Time   `json:"started_at"`
    FinishedAt *time.Time   `json:"finished_at"`
    CreatedAt  *time.Time   `json:"created_at"`
    PipelineID string       `json:"pipeline_id"`
}
```

### Organization (internal/buildkite/types.go)
```go
type Organization struct {
    ID      string `json:"id"`
    Slug    string `json:"slug"`
    Name    string `json:"name"`
    WebURL  string `json:"web_url"`
}
```

### Pipeline (internal/buildkite/types.go)
```go
type Pipeline struct {
    ID         string `json:"id"`
    Slug       string `json:"slug"`
    Name       string `json:"name"`
    Repository string `json:"repository"`
    WebURL     string `json:"web_url"`
    Emoji      string `json:"emoji"`
}
```

### Step (internal/buildkite/types.go)
```go
type Step struct {
    ID              string     `json:"id"`
    Type            string     `json:"type"`
    State           string     `json:"state"`
    Name            string     `json:"name"`
    Label           string     `json:"label"`
    Command         string     `json:"command"`
    AgentQueryRules []string   `json:"agent_query_rules"`
    ExitStatus      *int       `json:"exit_status"`
    Agent           string     `json:"agent"`
    WebURL          string     `json:"url"`
    UnblockableID   string     `json:"unblockable_id"`
    StartedAt       *time.Time `json:"started_at"`
    FinishedAt      *time.Time `json:"finished_at"`
}
```

### Agent (internal/buildkite/types.go)
```go
type Agent struct {
    ID             string   `json:"id"`
    Name           string   `json:"name"`
    Hostname       string   `json:"hostname"`
    Version        string   `json:"version"`
    ConnectedState string   `json:"connection_state"`
    OS             string   `json:"os"`
    IPAddress      string   `json:"ip_address"`
    Metadata       []string `json:"meta_data"`
    WebURL         string   `json:"url"`
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
