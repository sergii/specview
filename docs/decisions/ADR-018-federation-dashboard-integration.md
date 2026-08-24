# ADR-018 - Federation dashboard integration

## Status

Accepted.

## Context

H23 established a deterministic local multi-Host projection and broadcasts material peer/runtime changes through the Host process. H28 exposed that projection at `/federation` and through the read-only MCP `get_federation_status` tool.

The H28 Web page is correct but not yet integrated into the everyday Host dashboard: users must know the URL, and the page is request-time only even though the Host already has an SSE change channel driven by both local execution changes and H23 federation material changes.

Adding another browser poller or another federation runtime would duplicate authority and timing semantics.

## Decision

The Host dashboard will make Federation a first-class discoverable read view while continuing to reuse the existing projection and event infrastructure.

- Host pages expose an explicit navigation path to `/federation`.
- `/federation` exposes a read-only HTML fragment endpoint for its projection body.
- the federation page subscribes to the existing Host SSE endpoint;
- an existing Hub broadcast triggers a fragment refresh;
- the browser does not create a federation polling loop;
- `federationruntime.Projection` remains the only projection input;
- H23 peer polling, freshness, cached-outage behavior and correlation remain unchanged;
- failures remain explicit and never synthesize repository facts.

The existing shared Hub may produce refreshes for local Host material changes as well as remote-peer material changes. Extra read refreshes are acceptable because the projection builder is read-only and the page is a local UI; no new persistence or federation network request cadence is introduced by the browser.

## Non-goals

- changing federation polling cadence;
- changing repository correlation;
- changing HostSnapshot or peer contracts;
- browser-side polling;
- push federation;
- remote execution or mutation;
- refactoring the duplicated Host HTTP listener lifecycle introduced by H28.

## Consequences

Federation becomes reachable from the normal dashboard and stays current while open, but all authority remains in the existing Host/federation runtime. The listener-lifecycle duplication remains explicit technical debt for a separate slice if it becomes worth removing.
