---
specview:
  status: in_progress
---

# Thin editorial kanban design

Redesign the Specview board as a modern, thin, quiet kanban surface while preserving its read-only observation model.

## Design direction

Use the interaction geometry of a modern kanban board and the visual discipline of the TEPLOTEC reference at `https://brisk-riddle-3s3n.here.now/`.

The intended character is **thin industrial editorial kanban**:

- thin neutral-gray borders.
- light neutral surfaces.
- no heavy shadows or elevated cards.
- compact cards with generous internal breathing room.
- typography carries hierarchy more than decoration.
- numbered structures such as `01`, `02`, `03`.
- restrained status color used as small dots, not large badges.
- large project identity paired with quiet technical metadata.
- one thin application shell around the whole board.

## Foundations

- light theme for the POC dashboard.
- warm neutral page and surface colors.
- 1px structural borders.
- small radii for columns and cards; slightly larger radius for the outer shell.
- system sans typography with monospaced filesystem paths.
- no external font or runtime asset dependency.

## Atoms

- `Brand`: Specview wordmark.
- `Eyebrow`: small uppercase contextual label.
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
- `StatusColumn`: thin bordered container with sticky conceptual header and card stack.
- `SpecificationDetail`: editorial detail view using the same shell, typography and border language.

## Acceptance

- board remains read-only.
- workflow remains New, In progress, Done.
- cards remain clickable to specification detail.
- live SSE refresh remains unchanged.
- board supports horizontal overflow on narrow screens.
- no add-card, add-column, drag-and-drop or task-management controls.
- no heavy shadows, gradients, glassmorphism or thick panels.
- metadata errors remain visible.
- dashboard and detail page share one visual system.
