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
- compact project filesystem context rendered as a quiet path breadcrumb.
- one thin application shell around the whole board.

## Project context

The canonical identity of the observed project remains its full resolved filesystem path, but the board renders only the final two components: parent folder plus current project folder.

Example:

```text
sergii/specview
```

The complete resolved path remains available as hover/title metadata, for example:

```text
/Users/serhii/repos/sergii/specview
```

Reasons:

- agent-first development can create many short-lived or similarly named projects.
- a project name alone is not enough to distinguish personal, company, devbox, monorepo, sibling, and ephemeral workspaces.
- the parent/current pair usually provides the useful namespace context without exposing long machine-specific prefixes in the primary reading path.
- the full resolved path remains the concrete internal identity and is never discarded.

The previous `Source / specs` block is removed because `specs.path` is redundant once useful project context is visible. The board may still use the configured specs path internally.

## View modes

Specview supports five presentations of the same live workflow state.

### Classic

- preserves the bordered card-based kanban presentation.
- cards show title, path, and relative modification age.
- intended for small and medium projects where visual separation is useful.

### Classic Light

- preserves Classic card content, spacing, title, path, age, and activity presentation.
- removes the border around each individual specification card.
- keeps the workflow column shell and header structure.
- exists as a low-chrome A/B variant of Classic rather than a separate data model.

### Dense

- removes individual card boxes and renders specifications as compact text rows inside the bordered workflow columns.
- each row shows a stable display ID, specification title, and relative modification age.
- rows use hairline separators and minimal vertical space.
- intended for projects with tens or hundreds of specifications.

### Dense Detail

- preserves Dense row geometry and structural separators.
- adds the Markdown filename/path as a quiet second line under the specification title.
- keeps relative age and activity on the trailing edge.
- is intended for repositories where filenames carry useful technical context.

### Flow

- removes card borders, row separators, and workflow-column boxes.
- renders the workflow as a near-typographic stream: display ID, title, and relative age with almost no surrounding chrome.
- keeps New, In progress, and Done as semantic groups without turning them into table-like containers.
- uses whitespace, typography, square status markers, and subtle hover treatment instead of boxes.
- is the preferred visual surface for rapidly changing live agent activity because it minimizes visual noise around changing state.

The header contains a `Classic / Classic Light / Dense / Dense Detail / Flow` switch. The selected mode is local presentation state and is persisted in browser `localStorage`, so SSE refreshes do not reset it.

The internal persisted keys may remain `classic`, `classic2`, `dense`, `dense2`, and `flow`; user-facing names are semantic and do not require resetting an existing browser preference.

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

Dense, Dense Detail, and Flow views need short references that remain stable as a specification moves between workflow states.

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
- the `Specview` brand text uses the same system sans family, medium weight, and restrained tracking as body titles rather than behaving like a separate display logo font.
- board and detail page use the same brand typography.
- no external font or runtime asset dependency.

## Layout balance

- the top bar and compact project path establish context without consuming hero-scale vertical space.
- the workflow body should begin quickly and remain easy to scan.
- Classic spends more space on bordered individual specification cards.
- Classic Light preserves Classic information while reducing card chrome.
- Dense materially increases information density while keeping structural lines.
- Dense Detail keeps Dense geometry but adds filename context.
- Flow spends the least visual chrome and relies primarily on typography and whitespace.
- Dense, Dense Detail, and Flow share one workflow ID axis: the right edge of `02` aligns with the right edge of `H05`, `H08`, and other specification IDs.
- specification titles begin on the same vertical axis as the left edge of the workflow status square.
- workflow counts stay visually attached to the workflow label, for example `In progress 5`, instead of being pushed to the far right edge of the column header.

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
- Classic and Classic Light preserve their card information treatment.
- Dense and Dense Detail preserve compact row geometry.
- Flow keeps borderless typographic rows.
- no horizontal swipe is required to reach another workflow state.

Page gutters, project context spacing, and workflow spacing scale fluidly with the viewport. Long titles must not force a column or row wider than its available track.

## Atoms

- `Brand`: Specview product text and home link using the body sans language.
- `ProjectContext`: compact `parent/current` path in the top bar; full resolved path on hover.
- `ViewSwitch`: Classic / Classic Light / Dense / Dense Detail / Flow presentation selector.
- `ColumnIndex`: two-digit structural index sharing a right edge with specification IDs in row-based modes.
- `StatusSquare`: restrained workflow state marker.
- `ColumnTitle`: human-readable workflow state.
- `Count`: compact item count visually grouped with the workflow title.
- `SpecID`: stable short specification reference.
- `CardTitle`: specification title.
- `Path`: monospaced specification path.
- `Age`: relative modification age.
- `Hairline`: 1px neutral divider or border.
- `LiveIndicator`: small green circle plus Live label.

## Molecules

- `TopBar = Brand + ViewContext + ViewSwitch + ProjectContext + LiveIndicator`.
- `ColumnHeader = ColumnIndex + StatusSquare + ColumnTitle + Count`.
- `ClassicSpec = CardTitle + Path + Age` inside a bordered card.
- `ClassicLightSpec = CardTitle + Path + Age` without the individual card border.
- `DenseSpec = SpecID + CardTitle + Age`.
- `DenseDetailSpec = SpecID + CardTitle + Path + Age`.
- `FlowSpec = SpecID + CardTitle + Age` with no surrounding box or row line.
- `MetadataError = SpecificationPresentation + ErrorMessage`.
- `DetailMetadata = Status + Path + Age`.

## Organisms

- `ApplicationShell`: one bordered surface around the complete interface.
- `WorkflowBoard`: three workflow status groups on desktop and tablet.
- `WorkflowList`: the same status groups stacked vertically on phone widths.
- `StatusColumn`: structured workflow container in Classic, Classic Light, Dense, and Dense Detail; borderless group in Flow.
- `SpecificationDetail`: editorial detail view using the same shell, typography, and border language.

## Acceptance

- board remains read-only.
- workflow remains New, In progress, Done.
- Classic mode preserves the bordered card presentation.
- Classic Light preserves Classic information without individual card borders.
- Dense mode renders stable IDs and compact rows without individual card boxes.
- Dense Detail adds the Markdown filename/path beneath the title while preserving Dense geometry.
- Flow mode renders specifications without card, row, or column borders.
- selected mode survives reloads and SSE-driven page refreshes.
- specifications remain clickable to the detail page in all modes.
- live SSE refresh remains available.
- workflow markers are square and New is neutral graphite.
- Live remains a circular connection signal.
- the board shows compact `parent/current` project identity rather than a large project-name hero or a long absolute path.
- hovering the compact project identity exposes the full resolved filesystem path.
- the redundant Source / specs block is not shown.
- Specview brand text uses the same sans language on board and detail pages.
- in Dense, Dense Detail, and Flow the right edge of workflow indices aligns with the right edge of specification IDs.
- in Dense, Dense Detail, and Flow specification titles align with the left edge of the workflow status square.
- workflow counts render adjacent to their workflow titles rather than at the far right edge.
- all three columns fit without clipping on ordinary desktop and laptop viewport widths.
- tablet widths preserve kanban and may use horizontal overflow.
- phone widths use a vertical grouped list rather than horizontal kanban navigation.
- no add-card, add-column, drag-and-drop, or task-management controls.
- white background is used throughout the application.
- corners are square, not rounded.
- metadata errors remain visible.
- dashboard and detail page share one visual system.
