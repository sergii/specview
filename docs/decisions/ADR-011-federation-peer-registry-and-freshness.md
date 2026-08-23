# ADR-011: Persist federation peers as Host state and keep freshness outside HostSnapshot

- Status: Accepted
- Date: 2026-08-23

## Context

H20 introduced immutable `HostSnapshot` v1 documents and conservative local aggregation. H21 added a localhost-only HTTP source and a strict pull client. A user can now manually move or pull a snapshot from a DevBox to a laptop, but Specview still has no durable notion of a known peer.

The next step must not turn federation into a distributed database or overwrite source facts with transport metadata.

Two timestamps have different authority:

```text
snapshot.observed_at  = when the source Host observed its facts
retrieved_at          = when this Host successfully received that snapshot
```

They must remain distinct.

## Decision

### Peers are Host-level state

Peer configuration belongs beside the local Host catalog, not inside any repository.

A versioned peer registry stores entries such as:

```text
Peer
  name
  url
  expected_host_id
  stale_after_seconds
  credential reference
```

`expected_host_id` is required for a persisted peer. A durable peer is therefore identity-pinned rather than merely hostname-pinned.

### Secrets are never persisted in the peer registry

The first credential reference type is `env_headers`.

It maps an HTTP header name to the name of an environment variable:

```json
{
  "type": "env_headers",
  "headers": {
    "CF-Access-Client-Id": "SPECVIEW_DEVBOX_CF_CLIENT_ID",
    "CF-Access-Client-Secret": "SPECVIEW_DEVBOX_CF_CLIENT_SECRET"
  }
}
```

The registry stores only environment variable names. Secret values are resolved immediately before a request, are not serialized into Host state, and must not be included in errors or logs.

A peer that needs no application credentials, such as the H21 Tailscale Serve path, omits the credential reference.

### Last valid snapshot survives failed retrieval

Each peer may have a local cached observation:

```text
RemoteObservation
  peer name
  retrieved_at
  HostSnapshot v1
```

A failed later request does not delete or modify that last valid snapshot.

Attempt state is separate transport metadata:

```text
last_attempt_at
last_success_at
last_error
```

This lets the projection say `unreachable` while still retaining the last known source facts.

### Freshness is derived, never written into HostSnapshot

For H22 the peer projection exposes:

```text
fresh
stale
unreachable
never_retrieved
```

`fresh` requires a valid cached snapshot whose source `observed_at` is within the peer's `stale_after` threshold and whose latest retrieval attempt succeeded.

`stale` means the latest retrieval attempt succeeded but the source observation is older than the threshold.

`unreachable` means the latest retrieval attempt failed. If a cached snapshot exists, it remains available as last-known data and its source age is still visible.

`never_retrieved` means no valid snapshot has ever been cached.

The local clock never rewrites `snapshot.observed_at`.

### H22 remains explicit/manual first

H22 adds a registry and explicit refresh operations. It does not add a background polling daemon, peer discovery, push synchronization, or distributed consensus.

This keeps retrieval policy separate from the immutable snapshot and correlation contracts.

### Persistence is language-neutral and replaceable

Peer registry and remote observation formats are versioned JSON contracts with language-neutral fixtures. They participate in the Go-to-Rust conformance gate from ADR-007.

Internal Go package structure, timers, and CLI implementation are not contracts.

## Consequences

- laptop and DevBox peers can be named once and reused;
- a hostname change cannot silently substitute another Host when an expected Host ID is configured;
- Cloudflare Access-style credentials can be referenced without writing secret values to disk;
- Tailscale peers need no credential provider;
- temporary network failure does not erase last-known remote facts;
- freshness can be projected consistently without changing `HostSnapshot` v1;
- a future polling runtime can reuse the same registry and observation contracts.

## Non-goals

- background polling;
- automatic peer discovery;
- push federation;
- remote mutation or execution;
- durable global Repository IDs;
- shared database;
- storing raw credential values;
- UI for peer management;
- changing H19 correlation or H20 HostSnapshot semantics.
