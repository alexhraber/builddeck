# Project Specs

Canonical path: `.decapod/generated/specs/`.
These files are the project-local contract for humans and agents.

## Snapshot
- Project: builddeck
- Outcome: A production-quality Buildkite terminal flight deck — a sleek, live-updating Go TUI that gives platform engineers and release captains a dense, navigable control surface for organizations, pipelines, builds, jobs, logs, annotations, artifacts, and build health.
- Detected languages: Go
- Detected surfaces: Go CLI (Bubble Tea TUI)
- Current version: built from main branch, latest commit on 2026-07-09

## How to use this folder
- [INTENT.md](./INTENT.md): what success means, acceptance criteria, and explicit non-goals.
- [ARCHITECTURE.md](./ARCHITECTURE.md): topology, runtime model, data boundaries, ADR trail, and delivery plan.
- [INTERFACES.md](./INTERFACES.md): API/CLI contracts, internal message types, struct definitions, and failure semantics.
- [VALIDATION.md](./VALIDATION.md): proof commands, quality gates, evidence artifacts, and promotion flow.
- [SEMANTICS.md](./SEMANTICS.md): state machines, invariants, event flow, domain rules, and idempotency.
- [OPERATIONS.md](./OPERATIONS.md): deployment model, SLIs, secrets, and security testing for a CLI tool.
- [SECURITY.md](./SECURITY.md): threat model, auth/authz, data classification, and supply chain posture.

## Canonical `.decapod/` Layout
- `.decapod/data/`: canonical control-plane state (SQLite + ledgers).
- `.decapod/generated/specs/`: **Living project specs** for humans and agents.
- `.decapod/generated/context/`: deterministic context capsules.
- `.decapod/generated/policy/context_capsule_policy.json`: repo-native JIT context policy contract.
- `.decapod/generated/artifacts/provenance/`: promotion manifests and convergence checklist.
- `.decapod/generated/artifacts/custody/`: epistemic custody artifacts (assumptions, contradictions, deferred questions).
- `.decapod/generated/artifacts/inventory/`: deterministic release inventory.
- `.decapod/generated/artifacts/diagnostics/`: opt-in diagnostics artifacts.
- `.decapod/workspaces/`: isolated todo-scoped git worktrees.

## Day-0 Onboarding Checklist
- [x] Replace all placeholders in all 8 spec files.
- [x] Confirm primary user outcome and acceptance criteria in [INTENT.md](./INTENT.md).
- [x] Confirm topology and runtime model in [ARCHITECTURE.md](./ARCHITECTURE.md).
- [x] Document all inbound/outbound contracts in [INTERFACES.md](./INTERFACES.md).
- [x] Define validation gates and CI proof surfaces in [VALIDATION.md](./VALIDATION.md).
- [x] Define state machines and invariants in [SEMANTICS.md](./SEMANTICS.md).
- [x] Define SLOs, alerting, and incident process in [OPERATIONS.md](./OPERATIONS.md).
- [x] Define threat model and auth/authz decisions in [SECURITY.md](./SECURITY.md).
- [x] Ensure architecture diagram, docs, changelog, and tests are mapped to promotion gates.
- [x] Run all validation/test commands and attach evidence artifacts.

## Current Status
All specs are aligned with the codebase as of the latest commit. Run `decapod validate` to verify gates pass.