# Spec-driven repository layouts

## Purpose

Specview should normalize existing repo-native spec-driven-development conventions instead of requiring one proprietary directory layout.

There is currently no single formal SDD filesystem standard. The useful common denominator is semantic, not positional: projects persist intent, design/planning artifacts, executable tasks, and project-wide rules in version-controlled files.

Specview has a native convention, but its tool identity and its project artifacts are separate:

```text
repo/
├── .specview.yaml
├── .specview/        # reserved runtime namespace
└── specs/            # durable project specs
```

A top-level `specs/` directory is not an industry-wide standard and is not a sufficient Specview signature by itself.

## Compared conventions

| Convention | Primary paths | Main unit | Persistent project context | Characteristic model |
| --- | --- | --- | --- | --- |
| Specview native | configured root, default `specs/` | project-defined spec | `.specview.yaml` | lightweight repo-native specs |
| GitHub Spec Kit | `specs/<feature>/` | feature | `.specify/memory/constitution.md` | spec -> plan -> tasks -> implement |
| OpenSpec | `openspec/specs/`, `openspec/changes/<change>/` | current capability + change | `openspec/config.yaml`, current specs, optional schemas | current truth + delta change |
| Kiro | `.kiro/specs/<feature>/` | feature | `.kiro/steering/`, optionally `AGENTS.md` | requirements -> design -> tasks |
| BMAD | `_bmad-output/...` | PRD / epic / story / spec | project context | broader SDLC workflow |

## Specview native

The native adapter is `SpecviewAdapter`.

Default artifact structure:

```text
specs/
  ATS-001.md
  ATS-002.md
```

Optional namespaced front matter remains readable as ordinary Markdown without Specview:

```yaml
---
specview:
  status: in_progress
---
```

`SpecviewAdapter` owns the semantics only when selected by configuration or another unambiguous native signal. An arbitrary `specs/` directory is not automatically Specview-native merely because it contains Markdown.

`.specview/` is reserved for Specview-owned runtime material such as a rebuildable SQLite projection, indexes, sockets, or transient state. Durable intent remains in normal repository files such as `specs/`.

## GitHub Spec Kit

Typical feature structure:

```text
.specify/
  memory/
    constitution.md

specs/
  001-candidate-management/
    spec.md
    plan.md
    research.md
    data-model.md
    quickstart.md
    contracts/
    tasks.md
```

Semantic mapping:

- `constitution.md` -> project policy;
- `spec.md` -> intent/specification;
- `plan.md` -> implementation plan;
- `research.md` -> short-lived research artifact;
- `data-model.md` and `contracts/` -> design/contract artifacts;
- `tasks.md` -> execution decomposition.

Although Spec Kit also uses a top-level `specs/` directory, its feature-directory semantics are different from the flat/lightweight Specview-native convention. Auto-detection must use `.specify/`, not the shared `specs/` folder name.

## OpenSpec

OpenSpec makes the strongest temporal distinction among the compared conventions:

```text
openspec/
  specs/                     # current source of truth
    auth/
      spec.md

  changes/                   # proposed changes in flight
    add-two-factor-auth/
      proposal.md
      design.md
      tasks.md
      .openspec.yaml         # optional change metadata
      specs/
        auth/
          spec.md            # delta against current auth spec

    archive/                 # completed historical changes
```

Current OpenSpec documentation explicitly describes `openspec/specs/` as the source of truth for how the system currently behaves and `openspec/changes/` as proposed modifications that remain separate until archive/merge.

Its default spec-driven artifact dependency graph is conceptually:

```text
            proposal
            /      \
         specs    design
            \      /
              tasks
                |
            implement
```

This is not a mandatory waterfall. OpenSpec describes dependencies as enablers rather than phase gates and supports custom schemas where teams define different artifact graphs.

The distinction maps directly to normalized Specview planes:

```text
PlaneKnowledge
  current specs
  durable/historical truth

PlaneWork
  active change artifacts
```

The active change itself is represented by one primary artifact, normally `proposal.md`. Design, tasks, and delta specs are supporting artifacts for the same `WorkItemID`. A delta spec also relates back to the current capability it changes.

OpenSpec delta specs make change semantics first-class through sections such as:

```text
ADDED Requirements
MODIFIED Requirements
REMOVED Requirements
```

This is valuable to future Specview intent-drift and convergence analysis because the adapter can distinguish current behavior from explicitly proposed behavioral deltas without diffing arbitrary prose.

## Kiro

Typical structure:

```text
.kiro/
  steering/
    product.md
    tech.md
    structure.md

  specs/
    candidate-feedback/
      requirements.md
      design.md
      tasks.md
```

Semantic mapping:

- steering -> project policy/context;
- requirements -> intent;
- design -> design;
- tasks -> execution decomposition.

## BMAD

BMAD is broader than a spec file convention and can produce artifacts such as:

```text
_bmad-output/
  planning-artifacts/
    PRD.md
    architecture.md
    epics/

  implementation-artifacts/
    sprint-status.yaml

  project-context.md
```

Specview should treat BMAD as an adapter over its artifacts rather than adopting the whole methodology as a core assumption.

## Normalized semantic roles

Across frameworks, the stable abstraction is approximately:

```text
PROJECT POLICY
      |
      v
INTENT / SPEC
      |
      v
DESIGN / PLAN
      |
      v
TASKS
      |
      v
EXECUTION
      |
      v
EVIDENCE
```

Not every project needs every role and not every artifact represents active work.

Normalized artifacts therefore need both semantic kind and temporal/structural classification:

```text
kind
plane = knowledge | work
role  = primary | supporting
work_item_id
relations
```

Recommended normalized artifact kinds:

- `policy`
- `proposal`
- `spec`
- `requirement`
- `example`
- `rfc`
- `decision`
- `design`
- `plan`
- `task`
- `contract`
- `research`
- `checklist`

These are semantic roles, not required filenames.

## Discovery strategy

Adapter selection must distinguish explicit configuration from heuristic discovery:

```text
.specview.yaml + explicit adapter   -> configured adapter
.specview.yaml without adapter      -> SpecviewAdapter compatibility default
.specify/                           -> GitHub Spec Kit adapter
openspec/ with OpenSpec markers     -> OpenSpec adapter
.kiro/specs/                        -> Kiro adapter
_bmad-output/                       -> BMAD adapter
specs/ only                         -> ambiguous
```

Explicit configuration always wins. During `specview init`, multiple strong framework signatures are treated as ambiguous instead of being resolved by arbitrary precedence.

Native example:

```yaml
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
```

OpenSpec example:

```yaml
specs:
  adapter: openspec
  path: openspec
  pattern: "*.md"
```

A future custom Markdown adapter may support arbitrary company-specific roots, for example:

```yaml
artifacts:
  adapter: custom-markdown
  roots:
    - specifications
    - docs/decisions
```

That future adapter is intentionally separate from `SpecviewAdapter`.

## Adapter contract direction

An artifact adapter returns normalized entities and relations, conceptually:

```text
Artifact
- id
- kind
- plane
- role
- work item id
- title
- source path
- lifecycle/declared state when present
- metadata

Relation
- source
- type
- target
```

Example relations:

```text
ATS-003 depends_on ATS-001
RFC-004 relates_to ATS-003
ADR-009 resolves RFC-004
ADR-014 supersedes ADR-009
OpenSpec delta auth changes current:auth
PR-621 implements ATS-003
TEST-183 verifies REQ-017
```

The UI consumes the normalized projection and should not contain framework-specific path logic.

## Initial adapter priority

1. SpecviewAdapter - preserves the current POC after `.specview.yaml` configuration.
2. GitHub Spec Kit - proves feature-centric multi-artifact normalization.
3. OpenSpec - proves current-knowledge vs change-in-flight normalization.
4. Kiro - explicit requirements/design/tasks model.
5. BMAD - broader methodology, useful after the core normalization contract stabilizes.
6. Custom Markdown - company-specific layouts after the normalized contract is proven.

This order is an implementation priority, not a product endorsement ranking.
