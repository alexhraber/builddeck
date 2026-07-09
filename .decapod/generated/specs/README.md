# Project Specs

Canonical path: `.decapod/generated/specs/`.
These files are the project-local contract for humans and agents.

## Snapshot
- Project: builddeck
- Outcome: a Buildkite terminal flight deck for platform engineers and release captains who need dense, keyboard-driven visibility and safe Buildkite actions from the terminal.
- Primary language: Go
- Runtime surface: Go CLI / Bubble Tea TUI
- External system: Buildkite REST API v2

## Canonical Spec Set
- [INTENT.md](./INTENT.md): product outcome, acceptance criteria, assumptions, and non-goals.
- [ARCHITECTURE.md](./ARCHITECTURE.md): executable topology, runtime model, data boundaries, and ADR trail.
- [INTERFACES.md](./INTERFACES.md): CLI, config, Buildkite API, TUI action, browser, and filesystem contracts.
- [VALIDATION.md](./VALIDATION.md): proof commands, quality gates, and evidence artifacts.
- [SEMANTICS.md](./SEMANTICS.md): UI state transitions, selection invariants, action targeting, and idempotency.
- [OPERATIONS.md](./OPERATIONS.md): local runtime operations, failure handling, release posture, and support signals.
- [SECURITY.md](./SECURITY.md): token handling, trust boundaries, sensitive data rules, and supply-chain posture.

## Current Product Boundary
`builddeck` is a local terminal application. It does not host a service, run a daemon, own a server-side datastore, or receive webhooks. Shared state comes from Buildkite; local state is limited to user config and optional downloaded artifacts.

## Canonical `.decapod/` Layout
- `.decapod/data/`: Decapod control-plane state.
- `.decapod/generated/specs/`: living project specs for humans and agents.
- `.decapod/generated/context/`: deterministic context capsules.
- `.decapod/generated/policy/context_capsule_policy.json`: repo-native JIT context policy contract.
- `.decapod/generated/artifacts/provenance/`: promotion manifests and convergence checklist.
- `.decapod/generated/artifacts/custody/`: epistemic custody artifacts.
- `.decapod/workspaces/`: isolated todo-scoped git worktrees.

## Spec Maintenance Checklist
- [x] Product intent is grounded in README and executable code.
- [x] Architecture maps to `cmd/builddeck`, `internal/buildkite`, `internal/tui`, and `internal/config`.
- [x] Interfaces enumerate CLI/env/config, Buildkite REST endpoints, TUI actions, browser open, and artifact download behavior.
- [x] Validation gates use Go format, test, vet, build, and Decapod validation.
- [x] Semantics define pane navigation, refresh, filtering, log, action, download, and preset behavior.
- [x] Operations define local CLI support, dependency failures, release checks, and no-daemon runtime assumptions.
- [x] Security defines token handling, Buildkite trust boundary, local config storage, artifact download risk, and dependency review.
