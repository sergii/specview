# ADR-004: Acceptance is policy over revision-scoped evidence

- Status: Accepted
- Date: 2026-08-23

## Context

ADR-001 separates Intent, Execution, and Evidence. ADR-002 makes Evidence revision-scoped and deliberately leaves sufficiency decisions outside the Evidence domain.

Specview now needs to answer whether a work item can be considered verified for the exact software revision currently under observation.

Hard-coding that decision into Evidence would couple factual verification records to project-specific workflow rules. Hard-coding provider names such as `rspec`, `rubocop`, or `github-actions` would also make policy unstable when tooling changes.

## Decision

Acceptance is a separate deterministic policy layer over normalized Evidence.

```text
policy + work_item_id + current revision + Evidence[]
                         ↓
                    Acceptance
```

Policy declares stable logical checks. Evidence providers remain implementation details.

Example:

```text
required check = unit-tests
provider       = rspec
```

The project may later replace RSpec with another provider without changing the logical acceptance contract.

## States

The first Acceptance states are:

```text
unconfigured
waiting
blocked
accepted
```

`unconfigured` is distinct from `accepted`. Specview must never manufacture trust merely because no policy exists.

`waiting` means requirements remain unsatisfied but no current terminal result blocks the decision.

`blocked` means at least one current-revision required check has a terminal result that policy rejects.

`accepted` means every required logical check is satisfied for the exact current revision.

## Latest evidence rule

Multiple evidence records may exist for one logical check and revision.

Acceptance evaluates the latest observed record for:

```text
work_item_id + revision + check
```

This prevents an older pass from masking a later failure.

`observed_at` is the primary ordering key and evidence ID is the deterministic tie-breaker.

## Stale evidence

Evidence for a different revision cannot satisfy Acceptance.

If evidence exists for a required check but only for an older revision, the requirement is stale and remains unsatisfied.

## Skipped evidence

`skipped` is not equivalent to `passed`.

A requirement may explicitly declare that skipped evidence is acceptable. Otherwise skipped current-revision evidence blocks Acceptance.

## Failure semantics

Acceptance fails closed:

- failed evidence blocks;
- provider error blocks;
- malformed/invalid current-revision evidence blocks;
- unknown evidence state never becomes a pass;
- malformed policy returns an explicit error.

## Consequences

- Evidence remains factual and provider-neutral.
- Acceptance policy can vary by project without changing adapters.
- Source changes automatically make old green evidence insufficient.
- UI can project trust state without mutating workflow state.
- future MCP/A2A interfaces can expose the same deterministic Acceptance decision without embedding an LLM.
- future organization policy distribution can be added without changing Evidence records.

## Non-goals

This ADR does not define test execution, CI orchestration, branch protection, automatic workflow advancement, pull-request merging, or LLM judgment.
