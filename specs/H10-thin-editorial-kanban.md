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
- compact technical context rather than a large decorative project hero.
- no heavy shadows or elevated cards.
- typography carries hierarchy more than decoration.
- numbered structures such as `01`, `02`, `03`.
- restrained workflow color used as small square markers, not large badges.
- full project filesystem context rendered as a quiet path breadcrumb.
- one thin application shell around the whole board.

## Project context

The board identifies the observed project by its resolved filesystem path rather than by a large project-name heading.

Example:

```text
PROJECT / /Users/example/repos/specview
```

Reasons:

- agent-first development can create many short-lived or similarly named projects.
- a project name alone is not enough to distinguish local, devbox, monorepo, sibling, and ephemeral workspaces.
- the resolved path is the most concrete identity of the filesystem Specview is actually observing.
- the path should remain complete and horizontally scrollable on narrow screens rather than silently changing its meaning through truncation.

The previous `Source / specs` block is removed from the board because `specs.path` is redundant once full project context is visible. The board may still use the configured specs path internally.

## View modes

Specview supports three presentations of the same live workflow state.

### Classic

- preserves the card-based kanban presentation.
- cards show title, path, and relative modification age.
- intended for small and medium projects where visual separation is useful.

### Dense

- removes individual card boxes and renders specifications as compact text rows inside the bordered workflow columns.
- each row shows a stable display ID, specification title, and relative modification age.
- rows use hairline separators and minimal vertical space.
- intended for projects with tens or hundreds of specifications.

### Flow

- removes card borders, row separators, and workflow-column boxes.
- renders the workflow as a near-typographic stream: display ID, title, and relative age with almost no surrounding chrome.
- keeps New, In progress, and Done as semantic groups without turning them into table-like containers.
- uses whitespace, typography, square status markers, and subtle hover treatment instead of boxes.
- is the preferred visual surface for future rapidly changing live agent activity because it minimizes visual noise around changing state.

The header contains a `Classic / Dense / Flow` switch. The selected mode is local presentation state and is persisted in browser `localStorage`, so SSE refreshes do not reset it.

## Workflow markers

Workflow state uses small square markers:

```text
New          neutral graphite
In progress  amber
Done         green
Error        red
```

`New` is deliberately neutral rather than blue because it represents an unstarted workflow state, not an informational notification.

The top-bar Live indicator remains a circle because it represents a continuous connection signal rather than workflow state. The geometry distinction is intentional: **squares are states, the circle is a signal**.

## Stable display IDs

Dense and Flow views need short references that remain stable as a specification moves between workflow states.

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
- 1px structural borders where the selected view uses structure.
- square edges for the application shell, columns, cards, rows, and detail elements.
- system sans typography with monospaced filesystem paths and display IDs.
- no external font or runtime asset dependency.

## Layout balance

- the top bar and project path establish context without consuming hero-scale vertical space.
- the workflow body should begin quickly and remain easy to scan.
- Classic spends more space on individual specification cards.
- Dense materially increases information density while keeping structural lines.
- Flow spends the least visual chrome and relies primarily on typography and whitespace.

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
- Flow mode keeps borderless typographic rows inside each vertical status group.
- no horizontal swipe is required to reach another workflow state.

Page gutters, project breadcrumb spacing, and workflow spacing scale fluidly with the viewport. Long paths and titles must not force a column or row wider than its available track.

## Atoms

- `Brand`: Specview wordmark and home link.
- `ProjectBreadcrumb`: full resolved filesystem path of the observed project.
- `ViewSwitch`: Classic / Dense / Flow presentation selector.
- `ColumnIndex`: two-digit structural index.
- `StatusSquare`: restrained workflow state marker.
- `ColumnTitle`: human-readable workflow state.
- `Count`: compact item count.
- `SpecID`: stable short specification reference.
- `CardTitle`: specification title.
- `Path`: monospaced specification path.
- `Age`: relative modification age.
- `Hairline`: 1px neutral divider or border.
- `LiveIndicator`: small green circle plus Live label.

## Molecules

- `TopBar = Brand + ViewContext + ViewSwitch + LiveIndicator`.
- `ProjectContext = ProjectBreadcrumb`.
- `ColumnHeader = ColumnIndex + StatusSquare + ColumnTitle + Count`.
- `ClassicSpec = CardTitle + Path + Age`.
- `DenseSpec = SpecID + CardTitle + Age`.
- `FlowSpec = SpecID + CardTitle + Age` with no surrounding box or row line.
- `MetadataError = SpecificationPresentation + ErrorMessage`.
- `DetailMetadata = Status + Path + Age`.

## Organisms

- `ApplicationShell`: one bordered surface around the complete interface.
- `WorkflowBoard`: three workflow status groups on desktop and tablet.
- `WorkflowList`: the same status groups stacked vertically on phone widths.
- `StatusColumn`: structured workflow container in Classic and Dense; borderless group in Flow.
- `SpecificationDetail`: editorial detail view using the same shell, typography, and border language.

## Acceptance

- board remains read-only.
- workflow remains New, In progress, Done.
- Classic mode preserves the card presentation.
- Dense mode renders stable IDs and compact rows without individual card boxes.
- Flow mode renders specifications without card, row, or column borders.
- selected mode survives reloads and SSE-driven page refreshes.
- specifications remain clickable to the detail page in all modes.
- live SSE refresh remains unchanged.
- workflow markers are square and New is neutral graphite.
- Live remains a circular connection signal.
- the board shows the complete resolved project filesystem path instead of a large project-name hero.
- the redundant Source / specs block is not shown.
- all three columns fit without clipping on ordinary desktop and laptop viewport widths.
- tablet widths preserve kanban and may use horizontal overflow.
- phone widths use a vertical grouped list rather than horizontal kanban navigation.
- no add-card, add-column, drag-and-drop, or task-management controls.
- white background is used throughout the application.
- corners are square, not rounded.
- metadata errors remain visible.
- dashboard and detail page share one visual system.
