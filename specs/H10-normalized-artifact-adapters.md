---
specview:
  status: in_progress
---

# Normalized artifact adapters

Introduce the first architecture boundary that separates repository-specific spec layouts from the Specview domain model.

## Intent

Specview must be able to observe repo-native specification systems without requiring every project to store files in the same directory structure.

The native artifact root defaults to top-level `specs/`, but `specs/` is project content, not a sufficient Specview framework signature by itself. Native semantics are selected by `.specview.yaml` configuration.

The core vocabulary is:

```text
INTENT | EXECUTION | EVIDENCE
```

This slice is limited to the Intent side of that model.

## Scope

Define a normalized artifact model and adapter boundary for repository artifacts.

Initial normalized concepts:

- artifact;
- artifact kind;
- work item/capability/change identity when derivable;
- source path;
- declared metadata;
- relation between artifacts.

Initial artifact kinds should be able to represent:

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

Not every kind needs a dedicated UI in this slice.

## Adapter behavior

The first implementation must support:

1. `SpecviewAdapter` using the existing configured `specs.path`, with `specs/` as the native default artifact root;
2. optional explicit adapter selection through `specs.adapter`;
3. backward compatibility: `.specview.yaml` configs without `specs.adapter` resolve to `specview`;
4. normalized artifacts that contain no framework-specific path assumptions in the UI layer;
5. normalized relations such as `depends_on`, `relates_to`, `resolves`, and `supersedes` can be represented even before every adapter can discover them;
6. a store/projection that depends on the adapter contract rather than directly scanning Markdown files.

Candidate later adapters:

- GitHub Spec Kit;
- OpenSpec;
- Kiro;
- BMAD;
- custom company layouts;
- custom Markdown layouts that are deliberately distinct from the native `SpecviewAdapter` convention.

## Adapter naming

`SpecviewAdapter` is the canonical name for Specview's own native repository convention when selected by `.specview.yaml`:

```text
repo/
├── .specview.yaml
└── specs/
    ├── H01.md
    └── H02.md
```

Do not call this adapter `GenericMarkdownAdapter`. Generic/custom Markdown is a different future concern because arbitrary Markdown directories do not necessarily follow Specview semantics.

A plain `specs/` directory is ambiguous because other SDD frameworks may use the same path.

## Adapter detection

Selection must prefer explicit configuration and strong framework signatures:

```text
.specview.yaml + explicit adapter   -> configured adapter
.specview.yaml without adapter      -> SpecviewAdapter compatibility default
.specify/                           -> GitHub Spec Kit adapter
openspec/                           -> OpenSpec adapter
.kiro/specs/                        -> Kiro adapter
_bmad-output/                       -> BMAD adapter
specs/ only                         -> ambiguous
```

`.specview/` is reserved for future Specview-owned runtime material such as a rebuildable SQLite projection, indexes, sockets, or transient state. Durable intent remains in ordinary repository files such as `specs/`.

## Current truth vs work in flight

The normalized model must be capable of representing the OpenSpec-style distinction between:

```text
CURRENT KNOWLEDGE
```

and:

```text
CHANGE IN FLIGHT
```

The first implementation does not need to render a dedicated UI for this distinction, but it must not choose a schema that makes the distinction impossible later.

## SQLite

A future SQLite database may index normalized artifacts, relations, full-text search, and derived state.

SQLite is explicitly not the canonical source of repository intent. The projection must be rebuildable from authoritative adapters.

SQLite implementation is outside this slice unless needed for a minimal proof of the adapter boundary.

## UI

Do not redesign the dashboard as part of this slice.

The current UI is a projection. The adapter/core boundary should allow future list, board, graph, timeline, evidence, and workspace views without changing input contracts.

## Acceptance

- existing native `specs/*.md` projects continue to render unchanged through `SpecviewAdapter` when Specview configuration selects it;
- `.specview.yaml` may explicitly declare `specs.adapter: specview`;
- old `.specview.yaml` configuration without `specs.adapter` remains valid and defaults to `specview`;
- `specs/` alone is not treated as a definitive Specview framework signature;
- artifact discovery is behind an adapter interface rather than embedded in the store or UI;
- normalized artifacts expose stable semantic kinds independent of directory names;
- normalized relations can be represented even if the current UI does not render all of them;
- unsupported adapter names fail explicitly rather than silently choosing another interpretation;
- no Jira/Linear task-management behavior is introduced;
- no specification editing is introduced;
- no embedded LLM is introduced;
- documentation references ADR-001 as the governing semantic model.

## Implementation progress

Completed in the first H10 increment:

- `SpecviewAdapter` introduced in the Go domain package;
- adapter interface introduced;
- Store now refreshes through an adapter;
- normalized artifact kinds and relations introduced;
- native specs receive normalized `id` and `kind=spec`;
- `.specview.yaml` supports `specs.adapter`;
- missing adapter configuration defaults to `specview`;
- Specview dogfoods explicit `adapter: specview`;
- UI behavior remains unchanged.

Remaining H10 work:

- extract more adapter-neutral model details if needed as additional adapters are implemented;
- implement strong framework auto-detection that does not infer Specview semantics from `specs/` alone;
- implement the first non-Specview adapter, most likely GitHub Spec Kit;
- parse normalized relations from native metadata once the relation contract is finalized.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/research/spec-driven-layouts.md`
- `docs/adapters/specview.md`
