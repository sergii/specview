---
specview:
  status: in_progress
---

# H35 - Repository Evidence Overview

## Goal

Expose native repository Evidence as its own read-only Web plane, preserving native Evidence records as authority while using Intent only for optional work-item navigation enrichment.

## Acceptance

- [ ] A local repository context exposes direct navigation to `/project/evidence?id=<exact-local-repository-id>`.
- [ ] The Evidence page resolves the exact local Host catalog repository and returns 404 for an unknown repository ID.
- [ ] Native `.specview/evidence` records are scanned once and remain the only Evidence authority for the overview.
- [ ] The overview shows total, passed, failed/error, and invalid record counts without inventing a repository Evidence state.
- [ ] Each valid record exposes work item, revision, check, provider, kind, exact result, observed time, and native record path context.
- [ ] Invalid native Evidence remains visible with its validation/decoding error instead of being silently dropped.
- [ ] Intent metadata enriches matching Evidence with work-item title and detail links but does not control record existence.
- [ ] Evidence remains visible when Intent projection is unavailable; work-item links are simply unavailable.
- [ ] Matching Evidence records link to the existing work-item detail where Evidence and Acceptance semantics remain visible.
- [ ] Repository, repository-spec, repository-evidence, repository-acceptance, and local session-detail pages preserve the same repository context in History, Evidence, and Acceptance navigation.
- [ ] Host and Federation navigation semantics remain unchanged outside repository context.
- [ ] Go tests cover Evidence aggregation/enrichment, invalid-record visibility, Intent-unavailable behavior, rendered Web semantics, and unknown repository rejection.
- [ ] Chromium E2E covers Repository -> Evidence -> work-item detail and verifies revision/check/provider/result semantics.
- [ ] No Evidence schema, Acceptance contract, repository config, Host catalog, SQLite, federation wire model, MCP contract, source-control reader, or write authority changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- mutating or deleting Evidence records;
- running checks or CI;
- provider-specific Evidence normalization;
- remote federation Evidence aggregation;
- time-series analytics or charts;
- adding repository Evidence to MCP in this slice.
