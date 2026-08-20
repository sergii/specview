---
specview:
  status: in_progress
---

# Thin editorial kanban design

Redesign the Specview board as a modern, thin, quiet workflow surface while preserving its read-only observation model.

## Design direction

Use the interaction geometry of a modern kanban board and the visual discipline of the TEPLOTEC reference at `https://brisk-riddle-3s3n.here.now/`.

The intended character is **thin industrial editorial workflow**:

- pure white background.
- thin neutral-gray borders.
- square 90-degree corners.
- much more space in the header than in the board body.
- no heavy shadows or elevated cards.
- typography carries hierarchy more than decoration.
- numbered structures such as `01`, `02`, `03`.
- restrained status color used as small dots, not large badges.
- large project identity paired with quiet technical metadata.
- one thin application shell around the whole board.

## View modes

Specview supports two presentations of the same live workflow state.

### Classic

- preserves the current card-based kanban presentation.
- cards show title, path, and relative modification age.
- intended for small and medium projects where visual separation is useful.

### Dense

- removes individual card boxes and renders specifications as compact text rows.
- each row shows a stable display ID, specification title, and relative modification age.
- rows use hairline separators and minimal vertical space.
- intended for projects with tens or hundreds of specifications.

The header contains a `Classic / Dense` switch. The selected mode is local presentation state and is persisted in browser `localStorage`, so SSE refreshes do not reset it.

## Stable display IDs

Dense view needs short references that remain stable as a specification moves between workflow states.

- if the filename begins with an explicit identifier such as `H07`, `API12`, or `AUTH-03`, use that identifier.
- otherwise derive a deterministic five-character uppercase identifier from the specification path.
- fallback IDs are presentation references, not persistent metadata and not a replacement for an explicit domain identifier.
- sorting and workflow transitions must not change the displayed identifier.

Examples:

```text
H07      Git status observation
AUTH-03  Authentication flow
7K2QF    Retry semantics
```

## Foundations

- white theme for the POC dashboard.
- white page and white inner surfaces.
- 1px structural borders.
- square edges for the application shell, columns, cards, rows, and detail elements.
- system sans typography with monospaced filesystem paths and display IDs.
- no external font or runtime asset dependency.

## Layout balance

- the header must feel spacious and editorial.
- the workflow body must feel denser and faster to scan.
- whitespace should be invested primarily in the top hero and in large headings.
- Dense view deliberately spends less vertical space per specification than Classic view.

## Responsive behavior

### Desktop, 980px and wider

- render all three workflow columns fluidly within the available viewport width.
- do not force a fixed minimum board width that clips the Done column on ordinary laptop screens.

### Tablet / intermediate, 800px through 979px

- preserve kanban geometry.
- allow horizontal overflow with approximately two columns visible when three readable columns do not fit.

### Phone / narrow, below 800px

- stop presenting the workflow as horizontally scrolling kanban columns.
- transform the board into one vertical list grouped by workflow state: New, In progress, Done.
- each status section spans the viewport width.
- Classic mode keeps card styling inside each vertical status group.
- Dense mode keeps compact specification rows inside each vertical status group.
- no horizontal swipe is required to reach another workflow state.

Project title, page gutters, hero spacing, and eyebrow rule scale fluidly with the viewport. Long paths and titles must not force a column or row wider than its available track.

## Atoms

- `Brand`: Specview wordmark and home link.
- `Eyebrow`: small uppercase contextual label with a thin horizontal rule.
- `DisplayHeading`: project or specification title.
- `ViewSwitch`: Classic / Dense presentation selector.
- `ColumnIndex`: two-digit structural index.
- `StatusDot`: restrained state color marker.
- `ColumnTitle`: human-readable workflow state.
- `Count`: compact item count.
- `SpecID`: stable short specification reference.
- `CardTitle`: specification title.
- `Path`: monospaced specification path.
- `Age`: relative modification age.
- `Hairline`: 1px neutral divider or border.
- `LiveIndicator`: small green dot plus Live label.

## Molecules

- `TopBar = Brand + ViewContext + ViewSwitch + LiveIndicator`.
- `ProjectHeader = Eyebrow + DisplayHeading + SourcePath`.
- `ColumnHeader = ColumnIndex + StatusDot + ColumnTitle + Count`.
- `ClassicSpec = CardTitle + Path + Age`.
- `DenseSpec = SpecID + CardTitle + Age`.
- `MetadataError = SpecificationPresentation + ErrorMessage`.
- `DetailMetadata = Status + Path + Age`.

## Organisms

- `ApplicationShell`: one bordered surface around the complete interface.
- `WorkflowBoard`: three workflow status groups on desktop and tablet.
- `WorkflowList`: the same status groups stacked vertically on phone widths.
- `StatusColumn`: thin bordered container with compact header and specification collection.
- `SpecificationDetail`: editorial detail view using the same shell, typography, and border language.

## Acceptance

- board remains read-only.
- workflow remains New, In progress, Done.
- Classic mode preserves the card presentation.
- Dense mode renders stable IDs and compact rows without individual card boxes.
- selected mode survives reloads and SSE-driven page refreshes.
- specifications remain clickable to the detail page in both modes.
- live SSE refresh remains unchanged.
- all three columns fit without clipping on ordinary desktop and laptop viewport widths.
- tablet widths preserve kanban and may use horizontal overflow.
- phone widths use a vertical grouped list rather than horizontal kanban navigation.
- no add-card, add-column, drag-and-drop, or task-management controls.
- white background is used throughout the application.
- corners are square, not rounded.
- header is spacious.
- Dense mode materially increases information density.
- metadata errors remain visible.
- dashboard and detail page share one visual system.
