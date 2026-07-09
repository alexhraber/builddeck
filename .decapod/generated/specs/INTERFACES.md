# Interfaces

## Contract Principles
- Keep every external boundary explicit: CLI/env/config, Buildkite REST, terminal key events, browser open, and filesystem writes.
- Buildkite mutations must be deliberate keypresses with visible target semantics.
- Failures must include the operation context and remain recoverable in the TUI.
- Local files belong to the user; committed project files must not contain Buildkite tokens.

## CLI and Config Contracts
| Interface | Input | Output | Failure Behavior | Ownership |
|---|---|---|---|---|
| `builddeck` command | no required args | Bubble Tea alternate-screen TUI | exits non-zero if token missing or TUI fails | `cmd/builddeck` |
| `BUILDKITE_API_TOKEN` | bearer token | selected over config token | missing token prints setup guidance | user environment |
| Config file | `~/.config/builddeck/config.toml` or `$XDG_CONFIG_HOME/builddeck/config.toml` | token, `download_dir`, `[[filter_preset]]` values | parse/read warnings; fallback config | `internal/config` |
| Config save | filter presets/download dir | TOML file mode `0600`, directory mode `0700` | surfaced save error | `internal/config` |

## Buildkite REST Contracts
## Outbound Dependencies
| Client Method | HTTP Method / Path Shape | Purpose | Response Type | Retry/Idempotency |
|---|---|---|---|---|
| `ListOrganizations` | `GET /organizations` | list accessible organizations | `[]Organization` | safe to retry |
| `ListPipelines` | `GET /organizations/{org}/pipelines` | list pipelines, paginated up to 500 | `[]Pipeline` | safe to retry |
| `ListBuilds` | `GET /organizations/{org}/pipelines/{pipeline}/builds` | load recent 25 builds | `[]Build` | safe to retry |
| `GetBuild` | `GET /organizations/{org}/pipelines/{pipeline}/builds/{number}` | load build detail/jobs | `Build` | safe to retry |
| `ListAnnotations` | `GET /organizations/{org}/pipelines/{pipeline}/builds/{number}/annotations` | load annotations | `[]Annotation` | safe to retry |
| `ListArtifacts` | `GET /organizations/{org}/pipelines/{pipeline}/builds/{number}/artifacts` | load artifacts | `[]Artifact` | safe to retry |
| `GetJobLog` | `GET /organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{job}/log` | load job log content | `JobLog` | safe to retry |
| `ListAgents` | `GET /organizations/{org}/agents` | load agents, paginated up to 500 | `[]Agent` | safe to retry |
| `RetryJob` | `PUT /organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{job}/retry` | retry selected/top job | empty/Buildkite response | not idempotent; explicit keypress only |
| `UnblockJob` | `PUT /organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{job}/unblock` | unblock selected blocked job | empty/Buildkite response | not idempotent; only blocked jobs |
| `RebuildBuild` | `PUT /organizations/{org}/pipelines/{pipeline}/builds/{number}/rebuild` | rebuild selected/top build | `Build` if returned | not idempotent; explicit keypress only |
| `CancelBuild` | `PUT /organizations/{org}/pipelines/{pipeline}/builds/{number}/cancel` | cancel selected/top running build | `Build` if returned | constrained to running builds |
| `DownloadArtifactURL` | `GET /organizations/{org}/pipelines/{pipeline}/builds/{number}/jobs/{job}/artifacts/{artifact}/download` | resolve redirect URL | redirect location | safe to retry; download writes are local |

## TUI Action Contracts
## Inbound Contracts
| Key | Contract | Targeting |
|---|---|---|
| arrows / `h` `j` `k` `l` / `tab` | move focus or selection | active pane |
| `enter` | select/drill down | active pane item |
| `R` | refresh all data | current org/pipeline/build context where possible |
| `L` | open/close selected/top job log pane | left pane targets latest build; builds pane highlighted build; detail pane highlighted job |
| `r` | retry job | selected/top job; detail pane uses highlighted job |
| `b` | rebuild build | selected/top build |
| `x` | cancel build | selected/top build only when running |
| `u` | unblock job | selected/top blocked job |
| `o` | open Buildkite URL | pipeline/build/detail context |
| `d` | download first artifact | selected build, first listed artifact |
| `a` | toggle agent/queue saturation view | selected organization |
| `/` | filter active pane | loaded pipelines/builds/jobs only |
| `ctrl+f` | global search | loaded orgs/pipelines/builds/jobs only |
| `S` / `P` | save/load filter preset | local config |
| `?` | toggle help | TUI-local |
| `q` / `ctrl+c` | quit | process exits |

## Data Ownership
| Data | System of Record | Local Form | Mutation Path |
|---|---|---|---|
| Organizations, pipelines, builds, jobs, annotations, artifacts, agents, logs | Buildkite | in-memory structs | Buildkite REST API |
| Filter presets | User local config | TOML | `internal/config.Save` |
| Download directory | User local config | TOML | `internal/config.Save` |
| Downloaded artifacts | User filesystem | chosen path | artifact download flow |
| Decapod specs/governance | Decapod repo state | `.decapod/*` | Decapod-governed workflow |

## Failure Semantics
| Failure Class | User Contract | Observability |
|---|---|---|
| Missing token | exit before TUI with token setup message | stderr |
| Config read/parse failure | warn and continue with default config | stderr |
| Buildkite HTTP error | show endpoint/status/message context | TUI error/status |
| Network timeout | show request context; preserve current UI state | TUI error/status |
| Malformed JSON | show decode context | TUI error/status |
| No selected target | show no-op/status message, do not mutate | TUI status |
| Artifact download failure | show download error, keep UI responsive | TUI status |

## Timeout and Rate Budget
| Boundary | Budget | Notes |
|---|---|---|
| Buildkite HTTP client | 30 seconds | defined in `buildkite.NewClient` |
| Active-build polling | 2 seconds | only while non-terminal builds are present |
| Idle polling | 10 seconds | reduces API pressure |
| Manual refresh | immediate command | guarded against duplicate in-flight refreshes |

## Interface Versioning
- Public CLI identity is `builddeck`; short moniker `b7k` is informal only.
- Buildkite REST path contracts are versioned by Buildkite API v2.
- Config is intentionally minimal TOML; new keys must be optional and backward compatible.
- Keybindings in README, in-app help, and this spec must change together.
