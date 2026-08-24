---
specview:
  status: in_progress
---

# H31 - Federation Repository Drill-down

## Goal

Make one exact Host-scoped repository instance inspectable from the existing federation Web projection, including source Host context, active sessions, and worktrees, while preserving local-vs-remote authority boundaries.

## Acceptance

- [ ] Each repository instance on `/federation` links to a deterministic read-only detail route using its source Host ID and RepositoryInstance ID.
- [ ] The detail page resolves only facts already present in the current federation projection.
- [ ] The detail page shows repository name/root, source Host attribution, observation age, source repository ID, active state, agents, sessions, and worktrees when present.
- [ ] A local repository instance exposes a link to the existing live local `/project?id=...` view using `source_repository_id`.
- [ ] A remote repository instance never turns its source repository ID into a local `/project` link.
- [ ] Cached remote instances remain inspectable while a peer is stale or unreachable, with freshness and last transport error visible.
- [ ] Missing Host or RepositoryInstance identities fail explicitly instead of falling back to another repository.
- [ ] The shared Host navigation keeps Federation active on nested federation routes.
- [ ] Go Web semantics cover local and remote drill-down authority behavior.
- [ ] Chromium E2E covers federation instance navigation and local-vs-remote link behavior.
- [ ] No HostSnapshot, federation projection, peer cache, Host catalog, SQLite, MCP, Evidence, Acceptance, repository config, or network wire contract changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- remote Host navigation or proxying;
- remote execution control;
- new federation persistence;
- global repository identity;
- historical federation snapshots;
- changing repository correlation semantics;
- live polling on the detail page.
