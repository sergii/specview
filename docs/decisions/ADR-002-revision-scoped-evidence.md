# ADR-002: Evidence is revision-scoped

- Status: Accepted
- Date: 2026-08-21

## Context

Specview treats Evidence as the proof layer between implementation and Acceptance.

Verification results become unsafe if they are detached from the exact software they verified. A test result can remain green while the working tree, commit, container image, firmware, generated artifact, or pull-request head has already changed.

A dashboard that merely says `RSpec PASS` or `CI PASS` without identifying the verified subject can therefore display stale confidence.

## Decision

Every acceptance-eligible Evidence record must identify an opaque revision of the subject it verified.

The minimal identity is:

```text
work_item_id
revision
check
provider
```

Examples of revision identities include:

```text
git:<commit-sha>
workspace:sha256:<fingerprint>
pr:<number>/head:<commit-sha>
image:sha256:<digest>
firmware:sha256:<digest>
```

The Evidence core treats `revision` as opaque. It does not assume Git is the only revision system.

Future SCM and Execution adapters are responsible for determining the current revision/fingerprint of a work item. Future Acceptance policy compares required Evidence with that current revision.

## Consequences

- Evidence from a previous revision remains useful as history but cannot prove the current revision is acceptable.
- A new source change can automatically make previously green evidence stale without deleting it.
- Git commits, dirty worktrees, container images, generated artifacts, and hardware firmware can use the same contract.
- Evidence adapters do not need to know workflow stages.
- Acceptance policy does not need to know tool-specific output formats.

## Native transport

The first passive transport uses:

```text
.specview/evidence/*.json
```

This directory is runtime/derived material and is not the canonical source of project intent. Publishers should use atomic write-and-rename semantics.

## Non-goals

This ADR does not define how a workspace fingerprint is calculated, which checks are required, how tools are executed, or when a work item advances to Acceptance or Review. Those concerns belong to Execution and policy layers.
