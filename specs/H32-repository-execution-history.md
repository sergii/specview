---
specview:
  status: in_progress
---

# H32 - Repository-scoped Execution History

## Goal

Let a user move from one local repository view directly into that repository's logical execution timeline without searching the Host-wide history, while preserving the existing global history and execution-history contracts.

## Acceptance

- [ ] `/history` without a repository scope remains the existing Host-wide execution history.
- [ ] `/history?repository=<id>` accepts one exact local Host catalog repository ID and renders only execution-history entries for that repository.
- [ ] Unknown repository IDs return 404 instead of falling back, fuzzy matching, or showing another repository.
- [ ] Scoped history clearly identifies the repository and exposes navigation back to the live local repository view and to all Host history.
- [ ] The shared History navigation on `/project` and `/project/spec` carries the exact current repository ID into the scoped history route.
- [ ] Host and Federation views keep the normal unscoped `/history` navigation target.
- [ ] History remains active in the shared Host navigation for scoped history URLs.
- [ ] Active and ended logical sessions retain their existing ordering, identity, process diagnostics, and repository attribution.
- [ ] Go Web semantics cover exact filtering and unknown repository rejection.
- [ ] Chromium E2E covers repository -> scoped history -> all history navigation.
- [ ] No executionhistory, Host catalog, SQLite, MCP, federation, Evidence, Acceptance, repository config, or network contract changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- changing execution-history persistence or ordering;
- adding MCP filtering parameters;
- cross-Host execution history;
- fuzzy repository lookup;
- time-range filtering;
- agent/session search;
- remote repository navigation.
