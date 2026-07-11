# Semantics

## State Machines

### TUI Pane Navigation
```mermaid
stateDiagram-v2
  [*] --> LeftPane
  LeftPane --> CenterPane : tab / → / l
  CenterPane --> RightPane : tab / → / l
  RightPane --> LeftPane : shift+tab / ← / h
  LeftPane --> RightPane : shift+tab / ← / h
  CenterPane --> LeftPane : shift+tab / ← / h
  RightPane --> CenterPane : shift+tab / ← / h
```

### Build State (from Buildkite)
```mermaid
stateDiagram-v2
  [*] --> SCHEDULED
  SCHEDULED --> RUNNING
  RUNNING --> PASSED
  RUNNING --> FAILED
  RUNNING --> BLOCKED
  BLOCKED --> RUNNING : unblock
  FAILED --> RUNNING : retry
  PASSED --> [*]
  FAILED --> [*]
  BLOCKED --> [*]
```

### Job/Step State (from Buildkite)
```mermaid
stateDiagram-v2
  [*] --> PENDING
  PENDING --> SCHEDULED
  SCHEDULED --> RUNNING
  RUNNING --> PASSED
  RUNNING --> FAILED
  RUNNING --> SOFT_FAILED
  RUNNING --> BLOCKED
  BLOCKED --> RUNNING
  SOFT_FAILED --> PASSED : allowed to fail
  PASSED --> [*]
  FAILED --> [*]
  BLOCKED --> [*]
  SOFT_FAILED --> [*]
```

### Artifact Checksum Loading
```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> Fetching : loadArtifactChecksums()
  Fetching --> Loaded : all .sha256 downloaded + parsed
  Fetching --> Partial : some failed
  Partial --> Loaded : retry on next tick
  Loaded --> Idle : checksums in model
```

### Log Source Reference Parsing
```mermaid
stateDiagram-v2
  [*] --> Idle
  Idle --> Parsing : log content loaded
  Parsing --> Ready : file:line refs extracted
  Ready --> Navigating : n / N pressed
  Navigating --> Ready : scroll to ref
  Ready --> Idle : log view closed
```

## Invariants
| Invariant | Type | Validation |
|---|---|---|
| No API call without valid token | System | Startup exits with error if missing |
| In-flight request deduplication | System | `inFlight` guards per scope (builds, detail, annots, artifacts) |
| Model nil-pointer safety | Data | All accessors return zero values safely |
| Artifact checksum matches pipeline | Data | Parsed from `.sha256` artifact content |
| Polling interval adaptive | System | 2s when any build running, 10s when idle; manual rates: 2s, 5s, 10s, 30s, disabled |
| Selection preserved across refresh | Data | `rightScroll` tracked by build number |
| Emoji grapheme cluster integrity | Data | `uniseg.Graphemes()` for ZWJ sequences |
| Panel width never negative | System | `max(0, calculatedWidth)` in render |
| Build selection debounced | Data | 250ms debounce before fetching build detail |
| Source refs deduplicated | Data | Map keyed on `filepath:line:col` |
| Agent queue parsed from metadata | Data | `ParseAgentQueue()` extracts `queue=` from `meta_data` |
| Emoji bank thread-safe | Data | `sync.RWMutex` on bank reads/writes |
| ANSI stripped before source ref parsing | Data | `stripANSI()` run before regex scan |

## Event Sourcing Schema
N/A — no event sourcing. All state is in-memory Bubble Tea Model.

## Replay Semantics
N/A — no event log to replay. State rebuilds from API on each launch/refresh.

## Error Code Semantics
| Code | Namespace | Meaning | Retry Behavior |
|---|---|---|---|
| `missing_token` | `builddeck` | `BUILDKITE_API_TOKEN` not set | No retry — fatal |
| `auth_failed` | `builddeck` | 401 from API | No retry — token invalid |
| `rate_limited` | `buildkite` | 429 from API | Auto-retry next poll cycle |
| `timeout` | `builddeck` | HTTP client timeout | Auto-retry next poll cycle |
| `download_failed` | `builddeck` | Artifact download error | Manual retry via `d` key |
| `parse_error` | `builddeck` | JSON unmarshal failure | Skip field, continue |
| `no_builds` | `builddeck` | Pipeline has no builds | Show empty state |

## Domain Rules
- **Polling cadence**: Dynamic (2s active / 10s idle), or locked to 2s, 5s, 10s, 30s, or disabled via options overlay
- **Live mode** (`ctrl+l`): Locks polling to 2s regardless of build state
- **Artifact checksum**: Only displayed when `.sha256` companion artifact exists
- **Step selection**: Only `type == "script"` steps are selectable for logs
- **Pane focus**: Only one pane active at a time (visual border highlight)
- **Filter scope**: Filter applies only to active pane's loaded data
- **Download path**: User-specified directory; no temp file cleanup
- **Emoji rendering**: Nerd Font PUA glyphs preferred → Unicode fallback → raw text; grapheme-aware padding via `uniseg`
- **Emoji loading**: Static bank loaded at init; per-org custom emoji loaded on org selection
- **Source refs**: Two regex patterns — `path/file.ext:line:col` and `File "path", line N`; ANSI stripped before parsing
- **Action targeting**: `L`/`r`/`b`/`x`/`u` on left pane target top/latest build; on center pane target highlighted build; on right pane `r`/`u` target highlighted step, `b`/`x` target selected build
- **Cache invalidation**: Changing org resets all downstream caches (pipelines, builds, detail, annotations, artifacts)
- **Config save**: Token never written to config file; only filter presets and UI preferences

## Idempotency Contracts
| Operation | Idempotency Key | Duplicate Behavior |
|---|---|---|
| `R` (refresh) | Build number + pipeline | Re-fetches, preserves selection |
| `d` (download) | Artifact ID + dest path | Overwrites existing file |
| `a` (download all) | All artifact IDs | Parallel downloads, each idempotent |
| `r` (retry job) | Job ID | Buildkite API idempotent |
| `b` (rebuild) | Build number | Buildkite API creates new build |
| `x` (cancel) | Build number | Buildkite API idempotent |
| `u` (unblock) | Job ID | Buildkite API idempotent |

## Language Note
- Primary language: Go 1.26+
- TUI framework: Bubble Tea (v1.3.10) + Lip Gloss (v1.1.0) + Bubbles (v1.0.0)
- HTTP client: stdlib `net/http`
- Grapheme handling: `rivo/uniseg` v0.4.7
- Testing: `testing` + `httptest` + `testify/assert`

## Invariant Validation Strategy
| Invariant | Test Location | Method |
|---|---|---|
| Token required | `cmd/builddeck/main.go` | Exit code 1 + error message |
| No nil panics | `internal/tui/*_test.go` | Table tests with nil models |
| Polling interval | `internal/tui/update.go` | Assert interval in `tickCmd` |
| Checksum parsing | `internal/tui/model_test.go` | Mock `.sha256` response |
| Emoji width | `internal/tui/emoji_test.go` | `lipgloss.Width()` assertions |
| Source ref parsing | `internal/tui/source_refs_test.go` | Regex matching on mock log content |
| Build filtering | `internal/tui/search_test.go` | Substring match across build fields |
| State label mapping | `internal/tui/timefmt_test.go` | State-to-badge mapping table |
| Config TOML parsing | `internal/config/config_test.go` | Round-trip load/save |
| API pagination | `internal/buildkite/client_test.go` | Link header parsing |
