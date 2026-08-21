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

The implementation must support:

1. `SpecviewAdapter` using the existing configured `specs.path`, with `specs/` as the native default artifact root;
2. explicit adapter selection through `specs.adapter`;
3. backward compatibility: `.specview.yaml` configs without `specs.adapter` resolve to `specview`;
4. normalized artifacts that contain no framework-specific path assumptions in the UI layer;
5. normalized relations such as `depends_on`, `relates_to`, `resolves`, `supersedes`, and adapter-specific structural relations;
6. a store/projection that depends on the adapter contract rather than directly scanning Markdown files;
7. adapter-defined watch roots so a framework can observe more than one artifact directory without polling the whole source tree.

Supported adapters in this slice:

- `SpecviewAdapter`;
- `GitHubSpecKitAdapter`.

Candidate later adapters:

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

Selection prefers explicit configuration. `specview init` may use strong framework signatures to create that configuration:

```text
.specview.yaml + explicit adapter   -> configured adapter
.specview.yaml without adapter      -> SpecviewAdapter compatibility default
.specify/ during init               -> write github-spec-kit
specs/ only                         -> do not infer a foreign framework
```

Future adapters may add similarly strong init-time signatures:

```text
openspec/                           -> OpenSpec adapter
.kiro/specs/                        -> Kiro adapter
_bmad-output/                       -> BMAD adapter
```

`.specview/` is reserved for future Specview-owned runtime material such as a rebuildable SQLite projection, indexes, sockets, or transient state. Durable intent remains in ordinary repository files such as `specs/`.

## GitHub Spec Kit projection

`GitHubSpecKitAdapter` recognizes the feature-centric layout:

```text
.specify/
  memory/
    constitution.md

specs/
  001-feature/
    spec.md
    plan.md
    research.md
    data-model.md
    quickstart.md
    contracts/
    tasks.md
```

The feature `spec.md` becomes the dashboard-visible `ArtifactSpec`. Supporting files are indexed as related policy, plan, research, design, checklist, task, and contract artifacts.

For the current three-column UI only, feature status is derived without requiring Specview metadata:

```text
spec.md only                                      -> new
plan.md or incomplete tasks.md                    -> in_progress
all Markdown task checkboxes in tasks.md complete -> done
```

This is a compatibility projection, not a GitHub Spec Kit status standard.

## Representative fixture

The adapter is validated against a repository fixture under:

```text
internal/specs/testdata/github-spec-kit/
```

The fixture follows the current Spec Kit document shapes closely enough to exercise the real adapter boundary rather than isolated parser helpers. It includes:

- `.specify/memory/constitution.md`;
- a feature `spec.md` with feature branch, Draft status, user scenario, acceptance scenarios, and functional requirements;
- `plan.md`;
- `research.md`;
- partially completed `tasks.md` using Spec Kit checkbox syntax;
- `contracts/api.yaml`.

The integration test exercises:

```text
NewAdapter
  -> GitHubSpecKitAdapter
  -> Store.Refresh
  -> normalized Artifact[]
  -> feature status + supporting artifacts + relations
```

Spec Kit's own `**Status**: Draft` field remains source-framework metadata. Specview does not reinterpret it as the execution workflow state; the current compatibility board state is derived separately from plan/tasks progress.

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

The current UI is a projection. It renders only normalized `ArtifactSpec` entries as cards even when the Store contains policy, plan, task, contract, and other artifacts.

The adapter/core boundary should allow future list, board, graph, timeline, evidence, and workspace views without changing input contracts.

## Acceptance

- existing native `specs/*.md` projects continue to render unchanged through `SpecviewAdapter` when Specview configuration selects it;
- `.specview.yaml` may explicitly declare `specs.adapter: specview` or `specs.adapter: github-spec-kit`;
- old `.specview.yaml` configuration without `specs.adapter` remains valid and defaults to `specview`;
- `specview init` detects `.specify/` and creates configuration using `github-spec-kit`;
- `specs/` alone is not treated as a definitive Specview framework signature;
- artifact discovery is behind an adapter interface rather than embedded in the store or UI;
- adapters can define multiple narrow watch roots;
- GitHub Spec Kit supporting artifacts are indexed but do not become extra dashboard cards;
- normalized artifacts expose stable semantic kinds independent of directory names;
- normalized relations can be represented even if the current UI does not render all of them;
- representative Spec Kit repository structure is covered by an end-to-end adapter fixture;
- unsupported adapter names fail explicitly rather than silently choosing another interpretation;
- no Jira/Linear task-management behavior is introduced;
- no specification editing is introduced;
- no embedded LLM is introduced;
- documentation references ADR-001 as the governing semantic model.

## Implementation progress

Completed:

- `SpecviewAdapter` introduced in the Go domain package;
- adapter interface introduced;
- Store now refreshes through an adapter;
- adapter-defined watch roots introduced;
- normalized artifact kinds and relations introduced;
- native specs receive normalized `id` and `kind=spec`;
- `.specview.yaml` supports `specs.adapter`;
- missing adapter configuration defaults to `specview`;
- Specview dogfoods explicit `adapter: specview`;
- `GitHubSpecKitAdapter` implemented;
- `.specify/memory/constitution.md` maps to project policy;
- Spec Kit feature artifacts map to normalized semantic kinds;
- current Spec Kit feature status is deterministically projected from plan/tasks artifacts;
- `specview init` detects `.specify/` and writes the GitHub Spec Kit adapter;
- representative GitHub Spec Kit fixture added and exercised through the production adapter/store boundary;
- current dashboard remains visually spec-oriented while the Store can contain additional artifact kinds.

Remaining H10 work:

- evaluate edge cases from additional real-world Spec Kit repositories as they appear;
- extract individual requirements and tasks only if needed by the normalized relation model;
- implement broader runtime auto-detection only if it can remain unambiguous and backward-compatible;
- parse normalized relations from native Specview metadata once the relation contract is finalized;
- decide whether H10 should close after CI validation or include a second foreign adapter to prove the contract further.

## References

- `docs/decisions/ADR-001-intent-execution-evidence.md`
- `docs/research/spec-driven-layouts.md`
- `docs/adapters/specview.md`
- `docs/adapters/github-spec-kit.md`
