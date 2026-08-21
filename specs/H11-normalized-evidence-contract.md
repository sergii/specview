---
specview:
  status: in_progress
---

# Normalized evidence contract

Introduce the second normalized input boundary in the Specview model:

```text
INTENT | EXECUTION | EVIDENCE
                     ^
                     this slice
```

## Intent

Specview must be able to consume verification results from heterogeneous tools without hard-coding Rails, Go, Rust, CI, security, performance, or hardware semantics into the core.

A verification tool produces evidence. Specview observes and normalizes that evidence. Specview does not become the test runner or CI system in this slice.

## Core rule: evidence is revision-scoped

A green result is meaningful only for the exact software revision it verified.

Therefore every acceptance-eligible evidence record must identify:

```text
work_item + revision + check + provider
```

Examples of opaque revision identifiers:

```text
git:4f9c2ab
workspace:sha256:...
pr:621/head:4f9c2ab
image:sha256:...
firmware:sha256:...
```

Specview does not interpret the revision format in this slice. It treats it as an opaque identity that future Execution/SCM adapters can correlate with the current subject.

Evidence for a different revision is historical/stale evidence, not proof for the current Acceptance decision.

## Normalized evidence

The initial model contains:

```text
Evidence
├── version
├── id
├── work_item_id
├── revision
├── check
├── kind
├── provider
├── result
├── started_at
├── finished_at
├── observed_at
├── summary
├── metrics
├── path/source metadata
└── validation error
```

### Kinds

The core recognizes broad semantic kinds without knowing tool-specific details:

- compile;
- typecheck;
- lint;
- test;
- acceptance;
- contract;
- property;
- fuzz;
- mutation;
- architecture;
- security;
- migration;
- performance;
- hardware;
- review;
- ci;
- custom.

Examples:

```text
RSpec             -> kind=test
RuboCop           -> kind=lint
Brakeman          -> kind=security
Packwerk          -> kind=architecture
Mutant            -> kind=mutation
Pact              -> kind=contract
k6                -> kind=performance
cargo test        -> kind=test
clippy            -> kind=lint
go vet             -> kind=static/custom until a stronger semantic mapping is needed
hardware rig      -> kind=hardware
GitHub Actions    -> kind=ci
```

### Results

Normalized results:

```text
queued
running
passed
failed
error
skipped
```

`failed` means the verification completed and found a problem in the subject.

`error` means the verification mechanism itself could not produce a trustworthy verdict.

`skipped` is not equivalent to `passed`; policy decides whether a skipped check is acceptable.

## Native evidence transport

The first proof uses a passive filesystem transport under the reserved Specview runtime namespace:

```text
.specview/
└── evidence/
    ├── <record>.json
    └── ...
```

`.specview/` is derived/runtime material and should not be committed. It is distinct from `.specview.yaml`, which is durable project configuration.

External tools, wrappers, agents, CI sync adapters, or future dedicated adapters may publish records atomically into this directory. The Specview core reads them through `NativeEvidenceAdapter`.

Example:

```json
{
  "version": 1,
  "id": "ATS-003-rspec-20260821T120000Z",
  "work_item_id": "ATS-003",
  "revision": "git:4f9c2ab",
  "check": "unit-tests",
  "kind": "test",
  "provider": "rspec",
  "result": "passed",
  "started_at": "2026-08-21T11:59:52Z",
  "finished_at": "2026-08-21T12:00:00Z",
  "observed_at": "2026-08-21T12:00:00Z",
  "summary": "184 examples, 0 failures",
  "metrics": {
    "examples": 184,
    "failures": 0
  }
}
```

Publishers should write to a temporary file and rename to `.json` atomically. The native adapter ignores non-JSON and temporary files.

## Adapter boundary

Conceptually:

```text
EvidenceAdapter
├── Name()
├── Scan() -> Evidence[]
└── WatchRoots()
```

Candidate later adapters/providers include:

- JUnit/XML result files;
- RSpec JSON/JUnit reports;
- RuboCop JSON;
- Brakeman JSON;
- Mutant output;
- Go test JSON;
- cargo test / nextest;
- GitHub Checks and Actions;
- CodeQL/SARIF;
- Semgrep/SARIF;
- k6 reports;
- hardware-in-the-loop controllers;
- AI/human review systems.

Those adapters normalize existing output. Tool execution remains a separate concern.

## Acceptance policy is not part of Evidence

Evidence answers:

> What verification fact was observed for this revision?

Policy answers:

> Is this set of evidence sufficient to move this work item through Acceptance?

Therefore the Evidence record does not own fields such as `required: true` or `blocks_merge: true`.

A later policy slice may declare, for example:

```text
Rails project:
  require test + lint + security

IoT project:
  require test + contract + concurrency + hardware

Landing page:
  require build + smoke
```

The same Evidence model serves all of them.

## UI

No dashboard redesign is part of H11.

Future projections may show:

```text
EVIDENCE
RSpec        PASS
RuboCop      PASS
Brakeman     PASS
Mutant       RUNNING
```

but the current board remains unchanged until Acceptance policy is introduced.

## Acceptance

- normalized Evidence types are independent of specific tool implementations;
- evidence is linked to a work item and an opaque revision identity;
- stale evidence can be distinguished from evidence for the current revision;
- `failed`, `error`, and `skipped` remain semantically distinct;
- native filesystem evidence is observed from `.specview/evidence/`;
- invalid evidence records remain observable as validation errors instead of crashing the store;
- temporary/non-JSON files are ignored;
- adapter scan and store behavior are covered by tests;
- `.specview/` runtime material is ignored by Git;
- Specview does not execute verification tools in this slice;
- Specview does not decide Acceptance policy in this slice;
- UI remains unchanged.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/decisions/ADR-002-revision-scoped-evidence.md`
