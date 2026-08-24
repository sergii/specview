---
specview:
  status: in_progress
---

# H29 - Federation Dashboard Integration

## Goal

Make the H28 federation read surface discoverable from the normal Host dashboard and keep it current through the existing material-change SSE channel without introducing a second federation runtime or browser polling loop.

## Acceptance

- [ ] The Host dashboard exposes a visible `Federation` navigation link.
- [ ] The federation page has an explicit route back to the Host dashboard.
- [ ] `/fragments/federation` renders the same deterministic `federationruntime.Projection` body as `/federation`.
- [ ] Federation fragments are `no-store`.
- [ ] Fragment projection failures return an explicit non-200 response without invented repository facts.
- [ ] The federation page subscribes to the existing Host SSE endpoint.
- [ ] A Hub `changed` event refreshes the federation fragment without a full-page reload.
- [ ] No browser federation polling loop is introduced.
- [ ] H23 peer polling/freshness/cached-outage semantics remain unchanged.
- [ ] HostSnapshot v1, correlation and MCP contracts remain unchanged.
- [ ] Unit tests cover fragment success/failure behavior.
- [ ] Chromium semantic E2E covers dashboard navigation and live federation refresh.
- [ ] Existing H28 federation semantic E2E remains green.
- [ ] Full CI, race, coverage, binary smokes and release cross-build pass.

## Non-goals

- new federation transport or discovery;
- push federation;
- remote execution or writes;
- shared multi-Host persistence;
- polling/correlation changes;
- Host HTTP listener lifecycle refactor.
