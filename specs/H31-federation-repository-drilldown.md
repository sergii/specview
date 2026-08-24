---
specview:
  status: done
---

# H31 - Federation Repository Drill-down

## Goal

Make one exact Host-scoped repository instance inspectable from the existing federation Web projection, including source Host context, active sessions, and worktrees, while preserving local-vs-remote authority boundaries.

## Acceptance

- [x] Each repository instance on `/federation` links to a deterministic read-only detail route using its source Host ID and RepositoryInstance ID.
- [x] The detail page resolves only facts already present in the current federation projection.
- [x] The detail page shows repository name/root, source Host attribution, observation age, source repository ID, active state, agents, sessions, and worktrees when present.
- [x] A local repository instance exposes a link to the existing live local `/project?id=...` view using `source_repository_id`.
- [x] A remote repository instance never turns its source repository ID into a local `/project` link.
- [x] Cached remote instances remain inspectable while a peer is stale or unreachable, with freshness and last transport error visible.
- [x] Missing Host or RepositoryInstance identities fail explicitly instead of falling back to another repository.
- [x] The shared Host navigation keeps Federation active on nested federation routes.
- [x] Go Web semantics cover local and remote drill-down authority behavior.
- [x] Chromium E2E covers federation instance navigation and local-vs-remote link behavior.
- [x] No HostSnapshot, federation projection, peer cache, Host catalog, SQLite, MCP, Evidence, Acceptance, repository config, or network wire contract changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional head `3eca9a38522d06e7a535106ae65e580c8ba90e01` passed CI #1184 (`32734576149`) completely:

- formatting/modules/vet/race ✅
- total production coverage: **64.8%** ✅
- `internal/web`: **44.6%**
- `internal/federation`: **81.8%**
- `internal/federationruntime`: **80.0%**
- build ✅
- MCP and federation binary smokes ✅
- Chromium semantic E2E for local and remote federation repository drill-down ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

The earlier CI #1182 stopped at formatting because `internal/web/federation_repository_page_test.go` required `gofmt`; the functional head above includes that correction and passed the full pipeline.

## Non-goals

- remote Host navigation or proxying;
- remote execution control;
- new federation persistence;
- global repository identity;
- historical federation snapshots;
- changing repository correlation semantics;
- live polling on the detail page.
