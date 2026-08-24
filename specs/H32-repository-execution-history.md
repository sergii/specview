---
specview:
  status: done
---

# H32 - Repository-scoped Execution History

## Goal

Let a user move from one local repository view directly into that repository's logical execution timeline without searching the Host-wide history, while preserving the existing global history and execution-history contracts.

## Acceptance

- [x] `/history` without a repository scope remains the existing Host-wide execution history.
- [x] `/history?repository=<id>` accepts one exact local Host catalog repository ID and renders only execution-history entries for that repository.
- [x] Unknown repository IDs return 404 instead of falling back, fuzzy matching, or showing another repository.
- [x] Scoped history clearly identifies the repository and exposes navigation back to the live local repository view and to all Host history.
- [x] The shared History navigation on `/project` and `/project/spec` carries the exact current repository ID into the scoped history route.
- [x] Host and Federation views keep the normal unscoped `/history` navigation target.
- [x] History remains active in the shared Host navigation for scoped history URLs.
- [x] Active and ended logical sessions retain their existing ordering, identity, process diagnostics, and repository attribution.
- [x] Go Web semantics cover exact filtering and unknown repository rejection.
- [x] Chromium E2E covers repository -> scoped history -> all history navigation.
- [x] No executionhistory, Host catalog, SQLite, MCP, federation, Evidence, Acceptance, repository config, or network contract changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional head `8e55aa2f028f58c30a560fd45b78fc8733106ad3` passed CI #1190 (`32736768079`) completely:

- formatting/modules/vet/race ✅
- total production coverage: **64.9%** ✅
- `internal/web`: **45.8%**
- `internal/executionhistory`: **76.5%**
- build ✅
- MCP and federation binary smokes ✅
- Chromium repository -> scoped history -> all history navigation ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

## Non-goals

- changing execution-history persistence or ordering;
- adding MCP filtering parameters;
- cross-Host execution history;
- fuzzy repository lookup;
- time-range filtering;
- agent/session search;
- remote repository navigation.
