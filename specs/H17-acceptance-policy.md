---
specview:
  status: in_progress
---

# H17 - Acceptance Policy

## Goal

Introduce a policy layer that answers one question without changing Specview into a test runner or workflow engine:

> Is the evidence currently available for this work item sufficient for this exact revision?

The architecture remains:

```text
INTENT | EXECUTION | EVIDENCE
                         ↓
                    ACCEPTANCE
```

Evidence reports observed verification facts. Acceptance declares which logical checks are required and evaluates those facts for the current revision.

## Authority

Acceptance is a derived decision.

It does not own:

- specification content;
- current source revision;
- execution sessions;
- test execution;
- CI execution;
- provider-specific semantics.

The inputs are:

```text
work_item_id
current revision
acceptance policy
normalized Evidence[]
```

The output is a deterministic decision that can be deleted and recomputed.

## Logical checks, not providers

Policy depends on the stable logical `check` field introduced by H11, not the concrete tool that produced the result.

Example:

```text
required check = unit-tests
provider today = rspec
provider later = minitest
```

Changing providers must not require rewriting project workflow policy.

## Decision states

The first normalized states are:

```text
unconfigured
waiting
blocked
accepted
```

### unconfigured

No acceptance requirements are configured.

Specview must not imply that an unconfigured project is accepted.

### waiting

No blocking result exists, but at least one required logical check is not yet satisfied for the current revision.

Examples:

- missing evidence;
- evidence exists only for an older revision;
- check is queued;
- check is running.

### blocked

At least one required logical check has a terminal current-revision result that policy does not accept.

Examples:

- failed;
- error;
- invalid evidence record;
- skipped when the requirement does not explicitly allow skipped.

`blocked` takes precedence over `waiting` when multiple requirements are evaluated.

### accepted

Every required logical check is satisfied for the current revision.

A previous-revision pass is never sufficient.

## Requirement model

The initial domain model is intentionally small:

```text
Requirement
├── check
└── allow_skipped
```

`allow_skipped` defaults to false.

This keeps the H11 rule intact: `skipped` is never silently interpreted as `passed`; policy must explicitly decide whether it is acceptable.

Future policy features may add alternatives, groups, thresholds, or environment-specific requirements, but the first slice should not anticipate those prematurely.

## Multiple evidence records

Evidence is append/history oriented. More than one record may exist for the same logical check and revision.

The evaluator uses the latest observed record for:

```text
work_item_id + revision + check
```

`observed_at` is the primary ordering key. Evidence ID is the deterministic tie-breaker.

A later failed run therefore supersedes an earlier pass for the same logical check and revision.

## Stale evidence

If a required check has evidence for the work item but none for the current revision, the check is `stale` and the overall decision is `waiting` unless another check blocks it.

Stale evidence remains useful as history but cannot satisfy Acceptance.

## Failure semantics

Acceptance evaluation must fail closed.

Unknown or invalid current-revision evidence must never become a pass.

A malformed policy must return an explicit error rather than silently dropping a requirement.

Duplicate or blank logical check names are invalid policy.

## Configuration

Acceptance requirements are optional durable project configuration in `.specview.yaml`.

The initial syntax stays deliberately provider-neutral and small enough for the current strict config parser:

```yaml
acceptance:
  required:
    - unit-tests
    - lint
    - security
    - hardware-in-loop
  allow_skipped:
    - hardware-in-loop
```

Rules:

- `required` contains stable logical check names;
- `allow_skipped` is optional;
- every `allow_skipped` check must also appear in `required`;
- duplicate required or allow-skipped checks are invalid;
- unknown Acceptance keys are rejected;
- a config without `acceptance:` remains backward compatible and produces an unconfigured policy.

The domain conversion preserves required-check order and maps `allow_skipped` into the corresponding normalized `Requirement`.

## UI projection

Acceptance is a projection, not a workflow column.

Repository/work-item views may later show:

```text
ACCEPTANCE
unit-tests       PASS
lint             PASS
security         FAIL

BLOCKED
```

No drag-and-drop, status mutation, merge button, or automatic workflow advancement is introduced by H17.

## Revision resolution safety

Repository projection must not equate a dirty workspace with its Git HEAD.

A clean Git worktree may use:

```text
git:<head-sha>
```

A dirty worktree requires a workspace revision/fingerprint or must remain unevaluable. Existing evidence for the clean HEAD must never be reused to mark modified local files as accepted.

This fail-closed revision resolver is part of the repository-integration sub-slice.

## Implementation status

Completed:

- `internal/acceptance` domain package;
- normalized policy, requirement, decision, and check-state types;
- revision-scoped deterministic evaluator;
- latest-evidence selection per logical check;
- stale/missing/running/queued distinction;
- blocked precedence;
- explicit skipped policy;
- invalid evidence fails closed;
- unit tests for evaluator semantics;
- durable `.specview.yaml` Acceptance configuration;
- config compatibility and validation tests;
- conversion from config to normalized Acceptance policy.

Still to complete in H17:

- safe current-revision resolution for clean vs dirty worktrees;
- repository projection integration;
- minimal Acceptance UI;
- end-to-end Evidence -> Acceptance acceptance test;
- full CI/release gate on the completed slice.

## Acceptance criteria

- [x] Acceptance is a separate domain from Evidence.
- [x] Policy requires logical checks rather than concrete providers.
- [x] Evaluation is scoped to one work item and exact revision.
- [x] previous-revision evidence cannot satisfy current Acceptance.
- [x] latest current-revision evidence wins for one logical check.
- [x] failed/error/invalid required evidence blocks Acceptance.
- [x] missing/stale/queued/running required evidence waits.
- [x] skipped is rejected unless the requirement explicitly allows it.
- [x] no configured requirements produce `unconfigured`, not `accepted`.
- [x] malformed policy fails explicitly.
- [x] project configuration can declare acceptance requirements.
- [ ] dirty worktrees cannot inherit Acceptance from clean HEAD evidence.
- [ ] repository projection can evaluate the current revision against Evidence.
- [ ] UI exposes the decision without becoming a workflow editor.
- [ ] gofmt, module verification, go vet, race tests, build, and release cross-build pass on the completed slice.

## Out of scope

- executing tests or CI;
- automatically changing specification status;
- automatically merging pull requests;
- provider-specific required checks;
- branch protection replacement;
- review assignment;
- policy authoring UI;
- organization-wide policy distribution;
- cross-host policy federation;
- LLM-based acceptance decisions.
