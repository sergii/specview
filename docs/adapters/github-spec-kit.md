# GitHubSpecKitAdapter

`GitHubSpecKitAdapter` reads repositories that use GitHub Spec Kit's feature-centric Spec-Driven Development layout and projects them into Specview's normalized artifact model.

Canonical adapter name:

```text
github-spec-kit
```

## Detection and configuration

A repository initialized with GitHub Spec Kit normally contains:

```text
.specify/
specs/
```

When `specview init` sees `.specify/`, it writes:

```yaml
specs:
  adapter: github-spec-kit
  path: specs
  pattern: "*.md"
```

An existing `.specview.yaml` remains authoritative. Specview does not silently replace an explicitly configured adapter.

## Feature layout

The adapter expects the Spec Kit feature layout:

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

Each directory containing `spec.md` becomes one `ArtifactSpec` and therefore one card in the current dashboard.

Additional files are indexed as related normalized artifacts but are not rendered as additional dashboard cards in the v0 UI.

## Artifact mapping

| Spec Kit artifact | Specview kind |
| --- | --- |
| `.specify/memory/constitution.md` | `policy` |
| `spec.md` | `spec` |
| `plan.md` | `plan` |
| `research.md` | `research` |
| `data-model.md` | `design` |
| `quickstart.md` | `checklist` |
| `tasks.md` | `task` |
| `contracts/**` | `contract` |

Secondary feature artifacts receive a `belongs_to` relation to their feature ID.

## Status projection for the current v0 dashboard

GitHub Spec Kit does not use Specview's `specview.status` front matter. The adapter therefore derives the current three-state dashboard projection from Spec Kit artifacts:

```text
spec.md only
    -> new

plan.md exists
or tasks.md contains incomplete tasks
    -> in_progress

tasks.md exists and all Markdown task checkboxes are complete
    -> done
```

This is intentionally a compatibility projection for the current three-column UI. It is not a claim that GitHub Spec Kit itself defines these statuses.

As Specview moves to the richer lifecycle:

```text
TODO -> IN PROGRESS -> ACCEPTANCE -> IN REVIEW -> DONE
```

status derivation will move into policy/evidence rather than overloading the adapter with workflow semantics.

## Live observation

The adapter watches only its authoritative artifact roots:

```text
specs/
.specify/memory/
```

It does not scan or poll the entire source repository merely to notice specification changes.

## Current limitations

The first implementation intentionally does not yet:

- extract individual requirements from `spec.md` as separate entities;
- extract individual tasks from `tasks.md` as separate entities;
- interpret Spec Kit roadmap/spec-of-specs status tables;
- run Spec Kit commands;
- call an LLM;
- mutate Spec Kit files;
- derive PR, CI, review, or runtime state.

Those belong to later Intent, Execution, and Evidence slices.
