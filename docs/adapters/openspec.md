# OpenSpecAdapter

`OpenSpecAdapter` projects OpenSpec repositories into the normalized Specview artifact model without changing OpenSpec files.

## Upstream model

OpenSpec separates two temporal areas:

```text
openspec/
├── specs/       # current source of truth
└── changes/     # proposed changes in flight
```

Current specs describe how the system behaves now. Changes package proposed modifications until they are archived and their deltas are folded into current specs.

The default OpenSpec schema commonly uses:

```text
proposal
   ├── specs
   └── design
        \ /
        tasks
```

OpenSpec is intentionally fluid: artifact dependencies describe what can be created, not mandatory waterfall phase gates. Custom schemas may define different artifact graphs.

## Configuration

Explicit configuration:

```yaml
specs:
  adapter: openspec
  path: openspec
  pattern: "*.md"
```

`specview init` detects a strong OpenSpec root when `openspec/` exists with markers such as `config.yaml`, `specs/`, or `changes/`.

If both OpenSpec and another strong SDD signature such as `.specify/` are present, initialization fails instead of guessing. The user can then create `.specview.yaml` with an explicit adapter.

## Normalized temporal planes

OpenSpec motivated an explicit distinction in the Specview core:

```text
PlaneKnowledge
PlaneWork
```

### Current specs

Files under:

```text
openspec/specs/<capability>/spec.md
```

map to:

```text
kind        = spec
plane       = knowledge
role        = primary
id          = current:<capability>
status      = done
```

They are indexed as current knowledge but do not become cards on the active work board.

### Active changes

An active change under:

```text
openspec/changes/<change-id>/
```

maps to one work item.

The preferred primary artifact is:

```text
proposal.md -> kind=proposal, plane=work, role=primary
```

Supporting artifacts include:

```text
design.md             -> design
tasks.md              -> task
specs/**/spec.md       -> spec delta
```

All supporting artifacts carry:

```text
plane       = work
role        = supporting
work_item   = <change-id>
belongs_to  = <change-id>
```

Delta specs also point at the current capability they modify:

```text
changes -> current:<capability>
```

## Fluid artifact order

OpenSpec does not require proposal-first execution in all circumstances.

If a change directory contains no `proposal.md` yet, the adapter promotes the first available known artifact to the normalized primary role. This preserves one board work item without imposing a stricter workflow than OpenSpec itself.

## Current v0 status projection

The current three-column dashboard still needs `new`, `in_progress`, or `done`. OpenSpec does not define those as its canonical workflow states, so Specview derives a compatibility projection:

```text
proposal only                         -> new
design or delta specs present         -> in_progress
tasks.md with open tasks              -> in_progress
all tasks.md checkboxes completed     -> done
```

This projection is temporary UI compatibility, not an OpenSpec standard.

Future `TODO -> IN PROGRESS -> ACCEPTANCE -> IN REVIEW -> DONE` lifecycle policy will be independent of the source framework.

## Archive

`openspec/changes/archive/` is historical context, not active work.

The first adapter version excludes archive contents from the active projection. A later knowledge/history view may index archived changes separately without putting them back on the board.

## Watch scope

`OpenSpecAdapter` watches the `openspec/` namespace. It does not poll the application source tree.

This captures:

- current specs;
- active changes;
- archive transitions;
- OpenSpec configuration and custom schema changes.

## UI independence

The UI consumes normalized artifacts. It does not contain OpenSpec path rules.

The existing board shows only artifacts satisfying:

```text
plane = work
role  = primary
```

Therefore current source-of-truth specs remain searchable/indexable inputs without being confused with active work.
