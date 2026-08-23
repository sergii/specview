# ADR-010: Transport HostSnapshot over localhost-first pull HTTP

- Status: Accepted
- Date: 2026-08-23

## Context

H20 proved multi-host federation without networking. The `HostSnapshot` v1 wire contract and conservative aggregation semantics are now independent from transport.

The next useful step is to let one Specview Host fetch another Host's current snapshot without copying JSON files manually. This transport must not redefine Host identity, Repository correlation, source authority, or snapshot contents.

The first network implementation must also avoid accidentally exposing local development state to a LAN or the public internet. Specview already follows a local-first model, and current host UI serving is bound to loopback.

Private connectivity can be provided outside the Specview core. For example, Tailscale Serve can proxy a localhost service into a tailnet over HTTPS while applying tailnet access controls. Cloudflare Tunnel and Access can similarly protect a localhost origin and authenticate machine-to-machine requests. Specview therefore does not need a Tailscale or Cloudflare SDK dependency in its core transport.

## Decision

### Pull, not push

H21 uses read-only pull semantics:

```text
consumer Host
    GET /v1/federation/snapshot
remote source Host
```

The source Host remains authoritative. The consumer does not send repository state, acknowledgements, mutations, or merge decisions back to the source.

### Localhost-only source server

The first Specview federation HTTP server listens only on:

```text
127.0.0.1:7332
```

H21 does not provide a CLI option to bind the server to `0.0.0.0`, a LAN address, or a public interface.

A private ingress such as Tailscale Serve or Cloudflare Tunnel may proxy this localhost endpoint when remote access is desired. Access control and TLS termination belong to that private ingress boundary in H21.

### Stable endpoint

The source server exposes one read-only endpoint:

```text
GET /v1/federation/snapshot
```

The response body is exactly a valid `HostSnapshot` v1 JSON document generated from current local control-plane facts.

Response requirements:

- `Content-Type: application/json`;
- `Cache-Control: no-store`;
- non-GET methods are rejected;
- builder failures return an error status without a partial snapshot.

The endpoint does not expose a mutation API.

### Remote client safety

The H21 pull client accepts:

- `https://` for remote peers;
- `http://localhost`, `http://127.0.0.1`, or `http://[::1]` only for local development and tests.

Remote cleartext HTTP is rejected before a request is sent.

The client:

- has a bounded request timeout;
- disables automatic redirects;
- limits response size;
- requires a successful HTTP status;
- decodes the response with the strict H20 `HostSnapshot` decoder;
- may optionally pin an expected Host ID and reject a different source identity.

These checks are transport safety. They do not alter snapshot or correlation semantics.

### CLI boundary

H21 adds:

```text
specview federation serve
specview federation pull <url>
specview federation pull --expect-host <host:id> <url>
```

`pull` writes the validated `HostSnapshot` JSON to stdout so existing H20 aggregation remains composable:

```text
specview federation snapshot > laptop.json
specview federation pull https://devbox.example.ts.net/v1/federation/snapshot > devbox.json
specview federation aggregate laptop.json devbox.json
```

### Vendor-neutral private ingress

Specview does not automatically configure Tailscale, Cloudflare, DNS, certificates, tunnels, or Access policies.

The localhost-only source server is intentionally compatible with several private ingress choices. Vendor-specific automation can be added later without changing `HostSnapshot` or federation aggregation contracts.

## Consequences

- H20 wire semantics remain unchanged;
- no shared database is introduced;
- remote state remains read-only and source-attributed;
- accidental direct LAN/public bind is unavailable in H21;
- Tailscale and Cloudflare remain deployment choices rather than core dependencies;
- the same HTTP transport can survive a future Go-to-Rust implementation change;
- later peer configuration, polling, caching, freshness, and UI can build on a narrow tested fetch boundary.

## Non-goals

- push synchronization;
- peer discovery;
- background polling;
- remote mutation or execution;
- automatic Tailscale or Cloudflare configuration;
- public unauthenticated serving;
- distributed consensus;
- durable global Repository identity;
- WebSocket streaming;
- A2A;
- embedded LLM reasoning.
