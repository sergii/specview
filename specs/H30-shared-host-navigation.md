---
specview:
  status: in_progress
---

# H30 - Shared Host Navigation Shell

## Goal

Make the existing Host, execution-history, and federation Web projections directly discoverable from one shared read-only navigation shell without adding product authority or changing route semantics.

## Acceptance

- [ ] The normal Host page exposes direct navigation to Host, History, and Federation views.
- [ ] The same navigation is rendered from one shared Web template instead of being copied into individual pages.
- [ ] `/history` and `/federation` remain their existing read-only routes.
- [ ] The current Host view is marked with accessible `aria-current="page"` semantics.
- [ ] Repository and specification detail pages keep Host as the active top-level view.
- [ ] Theme controls and existing live-status behavior continue to work unchanged.
- [ ] Navigation remains compact on narrow screens and does not introduce a SPA/router dependency.
- [ ] Go Web semantics cover the shared navigation contract.
- [ ] Chromium E2E navigates Host -> History -> Federation -> Host and verifies active-view semantics.
- [ ] No Host catalog, SQLite, federation, execution-history, MCP, Evidence, Acceptance, repository config, or network contract changes.
- [ ] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Non-goals

- a new frontend framework or client-side router;
- write actions;
- redesigning Host, History, or Federation content;
- new Host views;
- changing federation or execution-history semantics;
- breadcrumbs for arbitrary future nested routes.
