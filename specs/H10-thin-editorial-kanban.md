---
specview:
  status: in_progress
---

# Thin editorial kanban design

Redesign the Specview board as a modern, thin, quiet kanban surface while preserving its read-only observation model.

## Design direction

Use the interaction geometry of a modern kanban board and the visual discipline of the TEPLOTEC reference at `https://brisk-riddle-3s3n.here.now/`.

The intended character is **thin industrial editorial kanban**:

- pure white background.
- thin neutral-gray borders.
- square 90-degree corners.
- high-density kanban columns and cards.
- much more space in the header than in the board body.
- no heavy shadows or elevated cards.
- compact cards with restrained metadata.
- typography carries hierarchy more than decoration.
- numbered structures such as `01`, `02`, `03`.
- restrained status color used as small dots, not large badges.
- large project identity paired with quiet technical metadata.
- one thin application shell around the whole board.

## Foundations

- white theme for the POC dashboard.
- white page and white inner surfaces.
- 1px structural borders.
- square edges for the application shell, columns, cards, and detail elements.
- system sans typography with monospaced filesystem paths.
- no external font or runtime asset dependency.

## Layout balance

- the header must feel spacious and editorial.
- the kanban board must feel denser and more compact.
- whitespace should be invested primarily in the top hero and in large headings.
- the board area should scan quickly and efficiently.

## Responsive behavior

- desktop widths at 980px and above render all three workflow columns fluidly within the available viewport width.
- desktop layout must not force a fixed minimum board width that clips the Done column on ordinary laptop screens.
- medium widths below 980px preserve kanban geometry using horizontal overflow with approximately two columns visible.
- mobile widths below 720px preserve one large readable column at a time with horizontal swipe or scroll.
- project title, page gutters, hero spacing, and eyebrow rule scale fluidly with the viewport.
- long paths and titles must not force a column wider than its responsive grid track.
- the top bar may reflow on small screens while keeping project status and view context readable.

## Atoms

- `Brand`: Specview wordmark.
- `Eyebrow`: small uppercase contextual label with a thin horizontal rule.
- `DisplayHeading`: project or specification title.
- `ColumnIndex`: two-digit structural index.
- `StatusDot`: restrained state color marker.
- `ColumnTitle`: human-readable workflow state.
- `Count`: compact item count.
- `CardTitle`: specification title.
- `Path`: monospaced specification path.
- `Age`: relative modification age.
- `Hairline`: 1px neutral divider or border.
- `LiveIndicator`: small green dot plus Live label.

## Molecules

- `TopBar = Brand + ViewLabel + LiveIndicator`.
- `ProjectHeader = Eyebrow + DisplayHeading + SourcePath`.
- `ColumnHeader = ColumnIndex + StatusDot + ColumnTitle + Count`.
- `SpecCard = CardTitle + Path + Age`.
- `MetadataErrorCard = SpecCard + ErrorMessage`.
- `DetailMetadata = Status + Path + Age`.

## Organisms

- `ApplicationShell`: one bordered surface around the complete interface.
- `KanbanBoard`: horizontally arranged workflow columns.
- `StatusColumn`: thin bordered container with compact header and dense card stack.
- `SpecificationDetail`: editorial detail view using the same shell, typography and border language.

## Acceptance

- board remains read-only.
- workflow remains New, In progress, Done.
- cards remain clickable to specification detail.
- live SSE refresh remains unchanged.
- all three columns fit without clipping on ordinary desktop and laptop viewport widths.
- board supports horizontal overflow instead of crushed columns on medium and narrow screens.
- mobile preserves readable one-column-at-a-time kanban navigation.
- no add-card, add-column, drag-and-drop or task-management controls.
- white background is used throughout the application.
- corners are square, not rounded.
- header is spacious.
- board body is visually denser.
- metadata errors remain visible.
- dashboard and detail page share one visual system.
