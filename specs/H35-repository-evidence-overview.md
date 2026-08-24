---
specview:
  status: done
---

# H35 - Repository Evidence Overview

## Goal

Expose native repository Evidence as its own read-only Web plane, preserving native Evidence records as authority while using Intent only for optional work-item navigation enrichment.

## Acceptance

- [x] A local repository context exposes direct navigation to `/project/evidence?id=<exact-local-repository-id>`.
- [x] The Evidence page resolves the exact local Host catalog repository and returns 404 for an unknown repository ID.
- [x] Native `.specview/evidence` records are scanned once and remain the only Evidence authority for the overview.
- [x] The overview shows total, passed, failed/error, and invalid record counts without inventing a repository Evidence state.
- [x] Each valid record exposes work item, revision, check, provider, kind, exact result, observed time, and native record path context.
- [x] Invalid native Evidence remains visible with its validation/decoding error instead of being silently dropped.
- [x] Intent metadata enriches matching Evidence with work-item title and detail links but does not control record existence.
- [x] Evidence remains visible when Intent projection is unavailable; work-item links are simply unavailable.
- [x] Matching Evidence records link to the existing work-item detail where Evidence and Acceptance semantics remain visible.
- [x] Repository, repository-spec, repository-evidence, repository-acceptance, and local session-detail pages preserve the same repository context in History, Evidence, and Acceptance navigation.
- [x] Host and Federation navigation semantics remain unchanged outside repository context.
- [x] Go tests cover Evidence aggregation/enrichment, invalid-record visibility, Intent-unavailable behavior, rendered Web semantics, and unknown repository rejection.
- [x] Chromium E2E covers Repository -> Evidence -> work-item detail and verifies revision/check/provider/result semantics.
- [x] No Evidence schema, Acceptance contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, source-control reader, or write authority changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional CI #1225 passed on the H35 functional head. Production statement coverage was 65.3% total, 73.1% for `internal/projectstate`, 49.3% for `internal/web`, and 72.0% for the unchanged `internal/evidence` package. Formatting, modules, vet, race tests, coverage gate, build, MCP and federation binary smokes, Chromium semantic tests, and Linux/macOS amd64/arm64 release archives all passed.

## Non-goals

- mutating or deleting Evidence records;
- running checks or CI;
- provider-specific Evidence normalization;
- remote federation Evidence aggregation;
- time-series analytics or charts;
- adding repository Evidence to MCP in this slice.
