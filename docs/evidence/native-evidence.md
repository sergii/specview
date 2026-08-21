# Native evidence transport

`NativeEvidenceAdapter` is the first passive Evidence transport for Specview.

It exists to prove the normalized Evidence contract without coupling Specview to any particular test runner, linter, CI platform, language, or hardware rig.

## Location

```text
repo/
├── .specview.yaml
├── specs/
└── .specview/
    └── evidence/
        └── *.json
```

`.specview/` is runtime/derived state and is ignored by Git. It is separate from `.specview.yaml`, which is durable configuration.

## Producer contract

A producer writes one JSON object per observed verification record.

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

## Logical check vs provider

`check` is the stable project-level verification capability that Acceptance policy may require.

`provider` is the concrete tool that produced the evidence.

```text
check=unit-tests provider=rspec
check=lint       provider=rubocop
check=security   provider=brakeman
```

Policy should normally depend on `check`, not `provider`. A project can replace RSpec with another test provider without rewriting workflow semantics.

`kind` is a broad grouping dimension such as `test`, `lint`, `static_analysis`, `security`, or `hardware`.

## Revision identity

`revision` is mandatory and opaque to the Evidence package.

Examples:

```text
git:4f9c2ab
workspace:sha256:9b...
pr:621/head:4f9c2ab
image:sha256:a8...
firmware:sha256:f2...
```

A future Execution/SCM layer determines the current revision. Evidence can then be classified as current or stale by identity comparison.

## Result semantics

```text
queued   check is waiting to execute
running  check is executing
passed   verification completed successfully
failed   verification completed and found a subject failure
error    verifier failed to produce a trustworthy verdict
skipped  verifier intentionally did not run
```

A `passed` result can be factual for a revision, but whether that logical check is required or sufficient belongs to policy.

## Atomic publication

Publishers should avoid exposing partially written JSON.

Recommended pattern:

```text
write <id>.tmp
fsync/close when appropriate
rename <id>.tmp -> <id>.json
```

The adapter scans only `.json` records and ignores temporary and unrelated files.

## Validation

Version 1 is strict:

- required identity fields must be present;
- kind and result must be recognized;
- terminal results require `finished_at`;
- timestamps cannot move backwards;
- unknown JSON fields are surfaced as validation errors rather than silently ignored.

An invalid record remains observable in the Evidence store with an error. One malformed producer record must not make all other evidence disappear.

## Adapter boundary

```text
external verification tool
        |
        | writes/exports result
        v
NativeEvidenceAdapter
        |
        v
normalized Evidence[]
        |
        +--> future SQLite projection
        +--> future Acceptance policy
        +--> future Evidence UI
```

Specview does not execute the external verification tool in this slice.

## Future adapters

The same normalized model can be produced directly from richer sources:

- RSpec JSON or JUnit;
- Go test JSON;
- cargo-nextest;
- RuboCop JSON;
- Brakeman JSON;
- SARIF from CodeQL/Semgrep;
- GitHub Checks/Actions;
- mutation testing reports;
- Pact/Schemathesis;
- k6;
- hardware-in-the-loop systems;
- AI or human review services.

Native JSON remains useful as a universal bridge for tools that do not warrant a dedicated adapter.
