---
specview:
  status: done
---

# H30 - Shared Host Navigation Shell

## Goal

Make the existing Host, execution-history, and federation Web projections directly discoverable from one shared read-only navigation shell without adding product authority or changing route semantics.

## Acceptance

- [x] The normal Host page exposes direct navigation to Host, History, and Federation views.
- [x] The same navigation is rendered from one shared Web template instead of being copied into individual pages.
- [x] `/history` and `/federation` remain their existing read-only routes.
- [x] The current Host view is marked with accessible `aria-current="page"` semantics.
- [x] Repository and specification detail pages keep Host as the active top-level view.
- [x] Theme controls and existing live-status behavior continue to work unchanged.
- [x] Navigation remains compact on narrow screens and does not introduce a SPA/router dependency.
- [x] Go Web semantics cover the shared navigation contract.
- [x] Chromium E2E navigates Host -> History -> Federation -> Host and verifies active-view semantics.
- [x] No Host catalog, SQLite, federation, execution-history, MCP, Evidence, Acceptance, repository config, or network contract changes.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional head `1f06ff3e5e71ca3c997eea49840807dbb6add087` passed CI #1175 (`32732028720`) completely:

- formatting/modules/vet/race ✅
- total production coverage: **64.7%** ✅
- `internal/web`: **41.7%**
- build ✅
- MCP and federation binary smokes ✅
- Chromium semantic E2E for shared Host navigation ✅
- Linux/macOS amd64/arm64 release archive cross-build ✅

## Non-goals

- a new frontend framework or client-side router;
- write actions;
- redesigning Host, History, or Federation content;
- new Host views;
- changing federation or execution-history semantics;
- breadcrumbs for arbitrary future nested routes.
