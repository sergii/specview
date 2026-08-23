---
specview:
  status: done
---

# Normalized evidence contract

Introduce the second normalized input boundary in the Specview model:

```text
INTENT | EXECUTION | EVIDENCE
                     ^
                     this slice
```

## Intent

Specview consumes verification results from heterogeneous tools without hard-coding Rails, Go, Rust, CI, security, performance, or hardware semantics into the core.

A verification tool produces evidence. Specview observes and normalizes that evidence. Specview does not become the test runner or CI system in this slice.

## Core rule: evidence is revision-scoped

A green result is meaningful only for the exact software revision it verified.

Every trustworthy evidence record therefore identifies:

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

Specview does not interpret the revision format in this slice. Future Execution/SCM adapters correlate the opaque identity with the current subject.

Evidence for a different revision is historical/stale evidence, not proof for the current Acceptance decision.

## Normalized evidence

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
├── source path
└── validation error
```

### Check vs provider vs kind

These fields intentionally mean different things.

```text
check
  stable logical verification capability required by project policy

provider
  concrete tool that produced the result

kind
  broad semantic category used for grouping and presentation
```

Example:

```text
check      = unit-tests
provider   = rspec
kind       = test
```

A project policy should normally require `unit-tests`, not `rspec`. The provider may later change from RSpec to Minitest without changing the workflow contract.

Another example:

```text
check      = static-analysis
provider   = go-vet
kind       = static_analysis
```

This separation is central to adapter portability.

### Kinds

The core recognizes broad semantic kinds without knowing tool-specific details:

- compile;
- typecheck;
- static_analysis;
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
go vet             -> kind=static_analysis
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

`failed` means verification completed and found a problem in the subject.

`error` means the verification mechanism itself could not produce a trustworthy verdict.

`skipped` is not equivalent to `passed`; policy decides whether a skipped check is acceptable.

## Revision predicates

The Evidence domain can answer factual questions only:

```text
MatchesRevision(revision)
PassedForRevision(revision)
```

It deliberately does not expose an `AcceptanceEligible` policy concept. Evidence reports facts; policy decides sufficiency.

## Native evidence transport

The first proof uses a passive filesystem transport under the reserved Specview runtime namespace:

```text
.specview/
└── evidence/
    ├── <record>.json
    └── ...
```

`.specview/` is derived/runtime material and is ignored by Git. It is distinct from `.specview.yaml`, which is durable project configuration.

External tools, wrappers, agents, CI sync adapters, or future dedicated adapters may publish records atomically into this directory. The core reads them through `NativeEvidenceAdapter`.

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

Therefore Evidence does not own fields such as `required: true` or `blocks_merge: true`.

A later policy slice may declare logical checks, for example:

```text
Rails:
  require unit-tests + lint + security

IoT:
  require unit-tests + protocol-contract + concurrency + hardware-in-loop

Landing page:
  require build + smoke
```

Concrete providers can vary while these policy names remain stable.

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

## Implementation

Completed:

- `internal/evidence` domain package;
- normalized `Record`, `Kind`, and `Result` types;
- revision matching and current-pass predicates;
- `EvidenceAdapter` interface;
- concurrent-safe Evidence `Store`;
- `NativeEvidenceAdapter` for strict JSON records;
- invalid records preserved with validation errors;
- non-JSON/temporary files ignored;
- `.specview/` runtime namespace ignored by Git;
- native transport documentation;
- ADR-002 for revision-scoped evidence.

## Acceptance

- normalized Evidence types are independent of specific tool implementations;
- evidence is linked to a work item and an opaque revision identity;
- stale evidence can be distinguished from evidence for the current revision;
- `failed`, `error`, and `skipped` remain semantically distinct;
- native filesystem evidence is observed from `.specview/evidence/`;
- invalid evidence records remain observable instead of crashing the store;
- temporary/non-JSON files are ignored;
- adapter scan and store behavior are covered by tests;
- `.specview/` runtime material is ignored by Git;
- Specview does not execute verification tools in this slice;
- Specview does not decide Acceptance policy in this slice;
- UI remains unchanged.

## Verification

GitHub Actions code gate passed with:

```text
gofmt            PASS
go mod verify     PASS
go vet            PASS
go test -race     PASS
go build           PASS
```

## Follow-up

The next logical slice is Acceptance policy: map project-specific required logical checks plus current revision into an Acceptance state without coupling policy to providers.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/decisions/ADR-002-revision-scoped-evidence.md`
- `docs/evidence/native-evidence.md`
