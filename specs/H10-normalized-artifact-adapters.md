---
specview:
  status: done
---

# Normalized artifact adapters

Introduce the architecture boundary that separates repository-specific SDD layouts from the Specview domain model.

## Intent

Specview observes repo-native specification systems without requiring every project to store or name artifacts the same way.

The core vocabulary remains:

```text
INTENT | EXECUTION | EVIDENCE
```

This slice establishes the normalized Intent input boundary.

## Native convention

The native Specview artifact root defaults to top-level `specs/`, but `specs/` is project content and not a sufficient Specview signature by itself.

```text
repo/
├── .specview.yaml
└── specs/
    ├── H01.md
    └── H02.md
```

`SpecviewAdapter` is selected by configuration. Old `.specview.yaml` files without `specs.adapter` remain compatible and resolve to `specview`.

## Normalized artifact model

Artifacts expose framework-independent semantics:

```text
Artifact
├── id
├── kind
├── plane
├── role
├── work_item_id
├── path
├── title
├── declared/projected status
└── relations
```

Supported semantic kinds include:

- policy;
- proposal;
- spec;
- requirement;
- example;
- RFC;
- decision/ADR;
- design;
- plan;
- task;
- contract;
- research;
- checklist.

## Knowledge vs Work

A repository may contain durable system knowledge that is not active work.

The normalized model therefore has two temporal planes:

```text
PlaneKnowledge
  current or historical project truth

PlaneWork
  artifacts participating in an active change
```

Artifacts inside a work item also have a structural role:

```text
RolePrimary
  represents the work item in a board/list projection

RoleSupporting
  belongs to the same work item without becoming another card
```

The current dashboard is now derived from:

```text
plane = work
role  = primary
```

rather than from a framework-specific filename or `kind=spec` check.

## Supported adapters

### SpecviewAdapter

Native lightweight convention:

```text
specs/*.md
```

Each native specification is normalized as:

```text
kind        = spec
plane       = work
role        = primary
work_item   = file id
```

### GitHubSpecKitAdapter

Recognizes:

```text
.specify/
  memory/
    constitution.md

specs/<feature>/
  spec.md
  plan.md
  research.md
  data-model.md
  quickstart.md
  contracts/
  tasks.md
```

Projection:

- constitution -> knowledge/policy;
- feature `spec.md` -> work/primary;
- plan, research, data model, quickstart, tasks, contracts -> work/supporting;
- supporting artifacts relate to the feature through `belongs_to`.

The current three-column status is a compatibility projection:

```text
spec.md only                                      -> new
plan.md or incomplete tasks.md                    -> in_progress
all Markdown task checkboxes in tasks.md complete -> done
```

Spec Kit's own specification metadata remains source-framework intent and is not confused with Specview execution state.

A representative fixture lives under:

```text
internal/specs/testdata/github-spec-kit/
```

and is exercised through `NewAdapter -> Store.Refresh -> Artifact[]`.

### OpenSpecAdapter

Recognizes the OpenSpec namespace:

```text
openspec/
├── specs/                 # current source of truth
└── changes/               # changes in flight
    ├── <change>/
    │   ├── proposal.md
    │   ├── design.md
    │   ├── tasks.md
    │   └── specs/**/spec.md
    └── archive/
```

Projection:

```text
openspec/specs/**/spec.md
  -> plane=knowledge

active change primary artifact
  -> plane=work, role=primary

change design/tasks/delta specs
  -> plane=work, role=supporting
```

Delta specs relate to the current capability they modify:

```text
change:delta:auth
  --changes--> current:auth
```

The adapter preserves OpenSpec's fluid workflow. `proposal.md` is preferred as the primary artifact, but if artifacts are created in another order the first available change artifact can represent the work item without imposing an artificial waterfall.

Archived changes are excluded from active work in this slice.

A representative fixture lives under:

```text
internal/specs/testdata/openspec/
```

and validates current knowledge, an active change, delta relations, task-derived status, fluid artifact order, and archive exclusion.

## Adapter detection

Explicit configuration wins.

`specview init` uses only strong signatures:

```text
.specify/                         -> github-spec-kit, path=specs
openspec/ with OpenSpec markers   -> openspec, path=openspec
no strong signature              -> specview, path=specs
```

If multiple strong SDD conventions are present, initialization fails rather than choosing arbitrary precedence.

A plain `specs/` directory remains ambiguous.

## Watch roots

Adapters define their own observation roots.

- SpecviewAdapter watches its configured artifact root.
- GitHubSpecKitAdapter watches `specs/` and `.specify/memory/`.
- OpenSpecAdapter watches the `openspec/` namespace.

The application source tree is not scanned merely to observe specification changes.

## SQLite

A future SQLite database may index normalized artifacts, relations, full-text search, and derived state.

SQLite is not canonical intent storage. The projection must remain rebuildable from authoritative adapters.

## UI

No UI redesign is part of H10.

The current dashboard remains a projection over the normalized model. Future list, board, graph, timeline, evidence, and workspace views can consume the same model without adapter-specific path logic.

## Acceptance

- existing Specview-native repositories remain compatible;
- adapter selection is explicit or strongly detected during init;
- `specs/` alone does not identify a framework;
- Store depends on the adapter interface rather than direct Markdown scanning;
- adapters define their own watch roots;
- normalized artifacts represent semantic kind, knowledge/work plane, primary/supporting role, work-item identity, and relations;
- GitHub Spec Kit supporting artifacts are indexed without becoming duplicate board cards;
- OpenSpec current specs are indexed as knowledge rather than active work;
- OpenSpec active changes become one work item with supporting artifacts;
- OpenSpec archive does not reappear as active work;
- current UI contains no Spec Kit or OpenSpec path logic;
- no Jira/Linear behavior is introduced;
- no specification editing is introduced;
- no embedded LLM is introduced.

## Verification

The GitHub Actions code gate passed with:

```text
gofmt            PASS
go mod verify     PASS
go vet            PASS
go test -race     PASS
go build           PASS
```

## Follow-up slices

H10 intentionally stops at the normalized adapter boundary.

Good follow-ups are independent work:

- native Specview relation metadata and an Intent graph;
- requirement/scenario extraction where useful;
- additional Kiro/BMAD/custom adapters;
- Execution adapters;
- Evidence adapters and acceptance policy.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/research/spec-driven-layouts.md`
- `docs/adapters/specview.md`
- `docs/adapters/github-spec-kit.md`
- `docs/adapters/openspec.md`
