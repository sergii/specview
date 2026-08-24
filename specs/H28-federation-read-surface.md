---
specview:
  status: in_progress
---

# H28 - Federation Read Surface

## Goal

Make the existing deterministic multi-Host federation projection directly readable by humans and MCP clients without introducing new federation authority or synchronization behavior.

## Acceptance

- [ ] Web exposes a read-only federation page from the existing `federationruntime.Projection`.
- [ ] The page lists the local Host and configured peers with source/freshness attribution.
- [ ] `unreachable` peers with a cached snapshot remain visible with their repository facts.
- [ ] `never_retrieved` peers remain visible without invented repository facts.
- [ ] Correlated repository groups preserve source instance/Host attribution.
- [ ] Web projection failures degrade explicitly instead of rendering invented facts.
- [ ] MCP adds one no-argument read-only tool: `get_federation_status`.
- [ ] MCP returns the same projection contract used by `specview federation status`.
- [ ] Existing MCP tools and argument contracts remain unchanged.
- [ ] The MCP tool contract is frozen by a language-neutral additive fixture.
- [ ] No peer credential values are exposed by Web or MCP.
- [ ] HostSnapshot v1 and H20 correlation semantics remain unchanged.
- [ ] Built-binary MCP smoke covers the federation tool.
- [ ] Chromium semantic E2E covers the federation page.
- [ ] Full CI, race tests, coverage and release cross-build pass.

## Non-goals

- peer discovery;
- push federation;
- remote execution or writes;
- a shared multi-Host database;
- changing polling cadence;
- changing repository correlation;
- adding another federation transport.
