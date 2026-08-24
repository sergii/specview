---
specview:
  status: done
---

# H33 - Repository Acceptance Overview

## Goal

Expose the repository-level Acceptance plane as one read-only overview derived from the existing repository policy, revision resolution, Evidence records, and per-work-item Acceptance semantics.

## Acceptance

- [x] A local repository context exposes direct navigation to `/project/acceptance?id=<exact-local-repository-id>`.
- [x] The overview resolves the exact local Host catalog repository and returns 404 for an unknown repository ID.
- [x] Repository Acceptance evaluates all valid work items using the existing `projectstate`, Evidence, revision, and Acceptance authorities without inventing a new state model.
- [x] Native Evidence is scanned once for one repository overview and grouped by work item before evaluation.
- [x] The overview shows accepted, waiting, blocked, and Evidence counts for configured policy.
- [x] Unconfigured policy remains explicitly unconfigured and does not require source-control inspection or invent acceptance authority.
- [x] Dirty or otherwise unresolved revision fails closed: configured work items are waiting and evaluation is marked pending.
- [x] Invalid Intent artifacts are excluded from Acceptance evaluation and counted separately rather than promoted into an Acceptance state.
- [x] Every evaluated work item links to the existing work-item detail where full Evidence and Acceptance checks remain visible.
- [x] Repository, repository-spec, and repository-acceptance pages carry the same exact repository context in History and Acceptance navigation.
- [x] Host and Federation navigation semantics remain unchanged outside repository context.
- [x] Go tests cover aggregate accepted/waiting semantics, dirty-worktree fail-closed behavior, rendered overview semantics, and unknown repository rejection.
- [x] Chromium E2E covers Host -> Repository -> Acceptance overview -> accepted work-item detail.
- [x] No Evidence schema, Acceptance decision contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, or write authority changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional head `62d1787ea4d5945d35b00c75501201de2c749ee9` passed CI #1200 (`32740617440`) completely:

- formatting/modules/vet/race ✅
- total production coverage: **65.0%** ✅
- `internal/projectstate`: **69.3%**
- `internal/web`: **47.1%**
- `internal/acceptance`: **92.2%**
- `internal/evidence`: **72.0%**
- build ✅
- MCP and federation binary smokes ✅
- Chromium Host -> Repository -> Acceptance overview -> accepted work-item detail ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

## Non-goals

- running checks or CI;
- mutating Acceptance state;
- changing per-work-item Acceptance semantics;
- adding repository-level Acceptance to MCP in this slice;
- remote Host Acceptance aggregation;
- time-series Evidence analytics;
- treating GitHub provider checks as normalized Evidence.
