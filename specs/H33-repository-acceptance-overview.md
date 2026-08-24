---
specview:
  status: in_progress
---

# H33 - Repository Acceptance Overview

## Goal

Expose the repository-level Acceptance plane as one read-only overview derived from the existing repository policy, revision resolution, Evidence records, and per-work-item Acceptance semantics.

## Acceptance

- [ ] A local repository context exposes direct navigation to `/project/acceptance?id=<exact-local-repository-id>`.
- [ ] The overview resolves the exact local Host catalog repository and returns 404 for an unknown repository ID.
- [ ] Repository Acceptance evaluates all valid work items using the existing `projectstate`, Evidence, revision, and Acceptance authorities without inventing a new state model.
- [ ] Native Evidence is scanned once for one repository overview and grouped by work item before evaluation.
- [ ] The overview shows accepted, waiting, blocked, and Evidence counts for configured policy.
- [ ] Unconfigured policy remains explicitly unconfigured and does not require source-control inspection or invent acceptance authority.
- [ ] Dirty or otherwise unresolved revision fails closed: configured work items are waiting and evaluation is marked pending.
- [ ] Invalid Intent artifacts are excluded from Acceptance evaluation and counted separately rather than promoted into an Acceptance state.
- [ ] Every evaluated work item links to the existing work-item detail where full Evidence and Acceptance checks remain visible.
- [ ] Repository, repository-spec, and repository-acceptance pages carry the same exact repository context in History and Acceptance navigation.
- [ ] Host and Federation navigation semantics remain unchanged outside repository context.
- [ ] Go tests cover aggregate accepted/waiting semantics, dirty-worktree fail-closed behavior, rendered overview semantics, and unknown repository rejection.
- [ ] Chromium E2E covers Host -> Repository -> Acceptance overview -> accepted work-item detail.
- [ ] No Evidence schema, Acceptance decision contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, or write authority changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- running checks or CI;
- mutating Acceptance state;
- changing per-work-item Acceptance semantics;
- adding repository-level Acceptance to MCP in this slice;
- remote Host Acceptance aggregation;
- time-series Evidence analytics;
- treating GitHub provider checks as normalized Evidence.
