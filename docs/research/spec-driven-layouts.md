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
| OpenSpec | `openspec/specs/`, `openspec/changes/<change>/` | current capability + change | `openspec/config.yaml` and current specs | current truth + change delta |
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

OpenSpec makes an important temporal distinction:

```text
openspec/
  specs/                     # current agreed system behavior
    interviews/
      spec.md

  changes/                   # changes in flight
    add-feedback/
      proposal.md
      design.md
      tasks.md
      specs/
        feedback/
          spec.md
```

This maps naturally to two Specview concepts:

```text
CURRENT KNOWLEDGE
vs
WORK IN FLIGHT
```

A completed change may update the current spec and then move to an archive. Specview should preserve this distinction in its normalized model rather than flattening every Markdown file into one work-item list.

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

Not every project needs every role.

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
openspec/                           -> OpenSpec adapter
.kiro/specs/                        -> Kiro adapter
_bmad-output/                       -> BMAD adapter
specs/ only                         -> ambiguous
```

Explicit configuration always wins.

Native example:

```yaml
specs:
  adapter: specview
  path: specs
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

An artifact adapter should return normalized entities and relations, conceptually:

```text
Artifact
- id
- kind
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
PR-621 implements ATS-003
TEST-183 verifies REQ-017
```

The UI consumes the normalized projection and should not contain framework-specific path logic.

## Initial adapter priority

1. SpecviewAdapter - preserves the current POC after explicit `.specview.yaml` configuration.
2. GitHub Spec Kit - feature-centric and increasingly common in agentic development.
3. OpenSpec - valuable current-truth vs change-in-flight model.
4. Kiro - explicit requirements/design/tasks model.
5. BMAD - broader methodology, useful after the core normalization contract stabilizes.
6. Custom Markdown - company-specific layouts after the normalized contract is proven.

This order is an implementation priority, not a product endorsement ranking.
