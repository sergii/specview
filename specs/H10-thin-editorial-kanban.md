---
specview:
  status: in_progress
---

# Thin editorial kanban design

Bring the Specview board much closer to the TEPLOTEC visual language while staying a read-only specification kanban.

## Direction

- white / near-white background.
- as much free and air space as practical.
- thin gray structural lines.
- very light surfaces.
- strong oversized editorial headings.
- quiet technical metadata.
- thin, restrained specification cards.
- kanban preserved as the primary interaction model.

## Foundations

### Background and surface

- near-white page background.
- near-white inner surfaces.
- avoid dark mode as the default presentation for this design.
- avoid heavy card fills, gradients, glass, or strong elevation.

### Borders

- use thin 1px gray rules.
- one outer application shell.
- one border around each kanban column.
- one border around each specification card.
- use subtle separators rather than shadows.

### Typography

- oversized display heading for the project title and specification title.
- small uppercase eyebrow labels with generous tracking.
- calm secondary metadata.
- monospaced paths and source identifiers.

### Spacing

- generous top and side padding.
- large quiet hero area above the kanban.
- enough empty space that the page still feels airy with only a few cards.
- more whitespace than a typical SaaS dashboard.

## Kanban mapping

### Board header

- top bar with `Specview`, a centered context label, and a compact live indicator.
- a large project hero below it.
- project hero includes an eyebrow, a large project name, and a quiet source block.

### Columns

- three default columns remain: New, In progress, Done.
- column headers use numbered structure: `01`, `02`, `03`.
- each column keeps a small colored status dot and a quiet count.

### Cards

- cards stay thin and compact.
- title first.
- monospaced path below.
- relative age aligned to the right.
- no add-card or drag-and-drop affordances in v0.0.1.

### Detail page

- match the same white, airy editorial system.
- large specification title.
- thin rule separating heading metadata from specification body.
- body content presented with minimal chrome and clear reading rhythm.

## Acceptance criteria

- the board still reads as a kanban board.
- the page is much closer to the provided TEPLOTEC screenshots than to a default shadcn demo board.
- the board uses a white / near-white background.
- empty space is intentionally preserved.
- all structural lines are thin and gray.
- cards remain clearly scannable and clickable.
- detail view visually belongs to the same design system.
