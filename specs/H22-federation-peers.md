---
specview:
  status: done
---

# H22 - Federation Peers and Freshness

## Goal

Persist known remote Specview Hosts, retrieve their unchanged H20 `HostSnapshot` v1 documents on demand, preserve the last valid snapshot across failures, and project freshness without turning federation into a sync engine.

## Peer registry

The registry is Host-level state stored beside the local catalog.

Peer v1 fields:

- stable local peer name;
- HTTPS or loopback HTTP URL accepted by H21;
- required expected Host ID;
- stale threshold in seconds;
- optional credential reference.

The first credential reference type is `env_headers`, where persisted values are environment variable names, never secret values.

## Remote observation

A successful refresh stores:

```text
peer
retrieved_at
HostSnapshot v1
```

A later failed refresh records attempt metadata but keeps the previous valid observation intact.

## Freshness projection

Statuses:

- `fresh` - latest attempt succeeded and source `observed_at` is within `stale_after`;
- `stale` - latest attempt succeeded but source observation is older than `stale_after`;
- `unreachable` - latest attempt failed; last valid snapshot remains available when one exists;
- `never_retrieved` - no valid snapshot has ever been stored.

`observed_at` remains source truth. `retrieved_at` is local transport metadata.

## Initial CLI

```text
specview federation peer add <name> --url <url> --host <host:id> [--stale-after <duration>]
specview federation peer list
specview federation peer refresh <name>
specview federation peer show <name>
specview federation peer remove <name>
```

Credential references are configured without secret values. A Tailscale peer needs no credential reference.

## Acceptance criteria

- [x] peer registry v1 is versioned and strict;
- [x] peer state is Host-level, not repository-level;
- [x] persisted peer requires expected Host ID;
- [x] H21 URL security validation is reused;
- [x] `env_headers` persists only environment variable names;
- [x] missing referenced environment variables fail before sending the request;
- [x] credential values never appear in persisted files or error text;
- [x] successful refresh stores retrieved_at plus unchanged HostSnapshot v1;
- [x] failed refresh preserves last valid snapshot;
- [x] fresh/stale/unreachable/never_retrieved projection is deterministic;
- [x] source observed_at is never rewritten;
- [x] language-neutral peer and remote-observation fixtures are tested;
- [x] add/list/show/refresh/remove CLI flow is covered by a built-binary smoke test;
- [x] H18 MCP, H19 identity, H20 federation, and H21 transport contracts remain compatible;
- [x] full gofmt/module/vet/race/coverage/binary/browser/release CI passes.

## Verification baseline

Functional head `e723e7a21778fbcb019e3b980213ffe24e442b76` passed CI #956 with:

- gofmt and module hygiene;
- go vet;
- race tests;
- 63.3% total production statement coverage;
- 68.4% `internal/federationpeers` coverage;
- build;
- MCP binary stdio smoke;
- federation file/CLI binary smoke;
- federation HTTP binary smoke;
- federation peer lifecycle binary smoke;
- Chromium semantic E2E;
- release archives.

## Out of scope

- background polling daemon;
- automatic peer discovery;
- UI peer management;
- push synchronization;
- remote execution or writes;
- shared database;
- changing HostSnapshot v1;
- changing repository correlation semantics.

## Next

After explicit peer state is proven, a later slice can add periodic polling and multi-host UI projections using the same peer and observation contracts.
