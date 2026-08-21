# Spec-driven repository layouts

## Purpose

Specview should normalize existing repo-native spec-driven-development conventions instead of requiring one proprietary directory layout.

There is currently no single formal SDD filesystem standard. The useful common denominator is semantic, not positional: projects persist intent, design/planning artifacts, executable tasks, and project-wide rules in version-controlled files.

## Compared conventions

| Convention | Primary paths | Main unit | Persistent project context | Characteristic model |
| --- | --- | --- | --- | --- |
| GitHub Spec Kit | `specs/<feature>/` | feature | `.specify/memory/constitution.md` | spec -> plan -> tasks -> implement |
| OpenSpec | `openspec/specs/`, `openspec/changes/<change>/` | current capability + change | `openspec/config.yaml` and current specs | current truth + change delta |
| Kiro | `.kiro/specs/<feature>/` | feature | `.kiro/steering/`, optionally `AGENTS.md` | requirements -> design -> tasks |
| BMAD | `_bmad-output/...` | PRD / epic / story / spec | project context | broader SDLC workflow |
| Generic repo specs | configurable, default `specs/` | project-defined | project-defined | minimal Markdown convention |

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

Specview should support zero-config detection where conventions are unambiguous:

```text
.specify/          -> GitHub Spec Kit
openspec/          -> OpenSpec
.kiro/specs/       -> Kiro
_bmad-output/      -> BMAD
specs/             -> Generic Markdown fallback
specifications/    -> Generic Markdown fallback
```

Explicit configuration must override discovery.

Example:

```yaml
artifacts:
  adapter: generic
  roots:
    - specifications
    - docs/decisions
```

## Generic Markdown fallback

The first-party minimal convention remains intentionally small:

```text
specs/
  ATS-001.md
  ATS-002.md
```

A file may declare normalized metadata through front matter, while ordinary Markdown remains readable without Specview.

Example:

```yaml
---
specview:
  id: ATS-001
  kind: spec
  status: new
---
```

Specview must not require projects to migrate from an established SDD framework merely to use the dashboard.

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

1. Generic Markdown - preserves the current Specview POC and zero-config `specs/` workflow.
2. GitHub Spec Kit - feature-centric and increasingly common in agentic development.
3. OpenSpec - valuable current-truth vs change-in-flight model.
4. Kiro - explicit requirements/design/tasks model.
5. BMAD - broader methodology, useful after the core normalization contract stabilizes.

This order is an implementation priority, not a product endorsement ranking.
