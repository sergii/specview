---
specview:
  status: done
---

# H28 - Federation Read Surface

## Goal

Make the existing deterministic multi-Host federation projection directly readable by humans and MCP clients without introducing new federation authority or synchronization behavior.

## Acceptance

- [x] Web exposes a read-only federation page from the existing `federationruntime.Projection`.
- [x] The page lists the local Host and configured peers with source/freshness attribution.
- [x] `unreachable` peers with a cached snapshot remain visible with their repository facts.
- [x] `never_retrieved` peers remain visible without invented repository facts.
- [x] Correlated repository groups preserve source instance/Host attribution.
- [x] Web projection failures degrade explicitly instead of rendering invented facts.
- [x] MCP adds one no-argument read-only tool: `get_federation_status`.
- [x] MCP returns the same projection contract used by `specview federation status`.
- [x] Existing MCP tools and argument contracts remain unchanged.
- [x] The MCP tool contract is frozen by a language-neutral additive fixture.
- [x] No peer credential values are exposed by Web or MCP.
- [x] HostSnapshot v1 and H20 correlation semantics remain unchanged.
- [x] Built-binary MCP smoke covers the federation tool.
- [x] Chromium semantic E2E covers the federation page.
- [x] Full CI, race tests, coverage and release cross-build pass.

## Verification

Functional head `9de14eee106997e5a3acc096f3d83c2ea2872534` passed CI #1134 (`32679246811`) with the complete matrix:

- formatting / modules / vet / race tests: PASS;
- total production statement coverage: **64.6%**;
- `internal/mcpserver`: **75.7%**;
- `internal/web`: **40.8%**;
- build: PASS;
- built-binary MCP stdio smoke, including `get_federation_status`: PASS;
- federation CLI / HTTP / peer / runtime binary smokes: PASS;
- Chromium semantic E2E, including `/federation`: PASS;
- release archive cross-build: PASS.

The final diff audit against H27 `main` confirms that H28 does not modify `federationruntime`, peer polling or credentials, HostSnapshot v1, H20 repository-correlation semantics, or federation persistence. H28 is an additive read-surface slice over the existing projection.

## Non-goals

- peer discovery;
- push federation;
- remote execution or writes;
- a shared multi-Host database;
- changing polling cadence;
- changing repository correlation;
- adding another federation transport.
