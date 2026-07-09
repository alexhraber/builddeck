# Operations

## Operational Model
`builddeck` operates as a local terminal process. There is no server, daemon, scheduler, queue worker, database migration, regional rollout, or on-call service owned by this repository. Operational readiness means the binary installs, starts, authenticates, degrades visibly on Buildkite/API/config errors, and can be released or rolled back as a normal Go CLI.

## Runtime Checklist
- [x] Binary builds from `./cmd/builddeck`.
- [x] Missing token exits with actionable setup guidance.
- [x] Buildkite API failures surface in the TUI without crashing.
- [x] Polling adapts between active and idle states.
- [x] In-flight guards prevent duplicate refresh/action requests.
- [x] Local config directory and file permissions are restrictive on save.
- [x] README documents install, run, token scopes, keybindings, limitations, and development commands.

## Service Level Objectives
| SLI | Target | Measurement | Owner |
|---|---|---|---|
| Startup correctness | Token-present startup reaches TUI; token-missing startup exits clearly | local run or tests | maintainer |
| Local command proof | format/test/vet/build pass before promotion | CI/local commands | maintainer |
| Buildkite request behavior | endpoint errors include operation context | client tests and manual TUI use | maintainer |
| UI responsiveness | refresh/action requests do not duplicate while in flight | TUI tests/manual run | maintainer |
| Documentation accuracy | README/help/spec keymaps match code | review and grep | maintainer |

## Monitoring and Signals
Because `builddeck` is local-first, monitoring is user-visible rather than centralized:

| Signal | Source | Operator Action |
|---|---|---|
| Missing/invalid token | stderr or Buildkite API error | set `BUILDKITE_API_TOKEN` or update config |
| Buildkite HTTP failure | TUI error/status message | retry, check token scopes, check Buildkite status |
| Stale or slow refresh | refresh status and last-updated display | manual refresh with `R`, inspect network/API limits |
| Action failure | TUI action status/error | verify selected target and token permissions |
| Config failure | startup warning | fix TOML or remove bad config |

## Health Checks
- Liveness: process accepts key events and renders the TUI.
- Readiness: token present, config parsed, initial organization request can run.
- Dependency health: Buildkite API responses for selected org/pipeline/build endpoints.
- Synthetic transaction: launch with test token/stubbed client in tests, exercise organization/pipeline/build selection and action messages.

## Incident Response
| Scenario | Triage | Mitigation |
|---|---|---|
| Token missing or wrong scopes | read startup/API message | create Buildkite token with documented scopes |
| Buildkite outage or network failure | confirm errors across endpoints | wait/retry; avoid destructive repeated action keys |
| Keymap/doc drift | compare README, `internal/tui/keys.go`, and specs | update all three surfaces in one change |
| Broken release binary | reproduce `go build` and TUI startup | revert to prior git revision or reinstall prior version |
| Artifact download writes unexpected file | inspect download directory/config | change `download_dir`; remove local file |

## Rollout Strategy
- Release unit: Go module binary installed by `go install` or built from source.
- Promotion gates: Go format/test/vet/build and Decapod validation/classification.
- Rollback: install/build a previous commit or tag.
- Blast radius: one user machine and the Buildkite resources visible to that user's token.

## Capacity Planning
- API load is bounded by current loaded org/pipeline/build scope, 25 recent builds per pipeline, pipeline/agent pagination capped at 500, and adaptive polling.
- Active builds poll more frequently for freshness; idle builds poll less frequently to conserve rate limit budget.
- Large organizations can stress terminal rendering and Buildkite API limits; future GraphQL snapshot work should be justified by measured REST pressure.

## Logging and Diagnostics
- Startup/config errors print to stderr.
- Runtime errors are represented in TUI status/error fields.
- Buildkite error messages include endpoint path, status code, and API-provided message when available.
- The client must not log bearer tokens.

## Secrets Management
| Secret | Source | Storage | Rotation | Consumer |
|---|---|---|---|---|
| Buildkite API token | `BUILDKITE_API_TOKEN` or optional config token | environment preferred; config may be read but saved config omits token | rotate in Buildkite and update env/config | `buildkite.Client` |

## Security Testing
| Test Type | Cadence | Tooling |
|---|---|---|
| Unit/integration | each PR | `go test ./...` |
| Static analysis | each PR | `go vet ./...` |
| Dependency vulnerability review | release or dependency update | `govulncheck` when available |
| Secret scan | before release | review diff for tokens and config examples |

## Compliance and Audit
- Regulatory scope is limited to user-local terminal use and the user's Buildkite access.
- Audit records for Buildkite mutations are owned by Buildkite; `builddeck` should show enough context for users to understand what they triggered.
- Release evidence lives in CI/local command output, Decapod validation output, and PR review.

## Pre-Promotion Checklist
- [ ] `gofmt -l .` returns no files.
- [ ] `go test ./...` passes.
- [ ] `go vet ./...` passes.
- [ ] `go build -o /tmp/builddeck-proof ./cmd/builddeck` succeeds.
- [ ] README, in-app help, and specs agree on feature scope and keybindings.
- [ ] No committed token, downloaded artifact, or local user config is included.
- [ ] `decapod validate --format json` passes or ambient blockers are explicitly reported.
