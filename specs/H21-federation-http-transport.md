---
specview:
  status: in_progress
---

# H21 - Federation HTTP Pull Transport

## Goal

Carry the unchanged H20 `HostSnapshot` v1 contract between two Specview Hosts over a small read-only HTTP boundary without weakening local-first security or source-host authority.

## Target flow

```text
DevBox
  specview federation serve
  127.0.0.1:7332
       ↑
 private HTTPS ingress
       ↑
Laptop
  specview federation pull https://devbox/.../v1/federation/snapshot
       ↓
HostSnapshot v1
       ↓
H20 aggregator
```

## Source server

`specview federation serve`:

- listens only on `127.0.0.1:7332`;
- exposes `GET /v1/federation/snapshot`;
- builds a fresh H20 HostSnapshot for each successful request;
- emits `application/json` and `Cache-Control: no-store`;
- never exposes mutation endpoints;
- rejects unsupported methods;
- fails without returning partial snapshot JSON when local collection fails.

H21 intentionally has no option to bind this server to a non-loopback address.

## Pull client

`specview federation pull <url>`:

- accepts remote HTTPS;
- accepts cleartext HTTP only for loopback hosts;
- rejects remote HTTP before sending a request;
- rejects redirects;
- uses a bounded timeout;
- limits response size;
- requires a successful HTTP response;
- strict-decodes HostSnapshot v1;
- prints canonical validated snapshot JSON to stdout.

Optional peer pinning:

```text
specview federation pull --expect-host host:<uuid> <url>
```

A different returned Host ID must fail.

## Private ingress

H21 does not configure the private network itself.

Supported deployment patterns include:

- Tailscale Serve -> localhost federation server;
- Cloudflare Tunnel + Access -> localhost federation server;
- another authenticated TLS reverse proxy -> localhost federation server.

The transport core remains vendor-neutral.

## Acceptance criteria

- [ ] localhost-only federation server exists;
- [ ] GET snapshot endpoint returns valid HostSnapshot v1;
- [ ] endpoint emits no-store JSON responses;
- [ ] unsupported methods are rejected;
- [ ] snapshot builder errors do not return partial snapshot JSON;
- [ ] pull client accepts remote HTTPS;
- [ ] pull client allows HTTP only for loopback development/test URLs;
- [ ] remote cleartext HTTP is rejected before request;
- [ ] redirects are rejected;
- [ ] response size is bounded;
- [ ] strict H20 snapshot validation is reused;
- [ ] expected Host ID pinning works;
- [ ] built-binary localhost serve/pull flow is tested;
- [ ] H20 aggregation accepts a pulled snapshot unchanged;
- [ ] H18 MCP, H19 identity, and H20 federation fixtures remain unchanged;
- [ ] gofmt, module, vet, race, coverage, MCP binary, federation binary, browser, and release gates pass.

## Out of scope

- peer discovery;
- automatic polling;
- persistence of remote snapshots;
- UI for remote Hosts;
- application-level user login;
- automatic Tailscale/Cloudflare setup;
- push sync;
- remote writes/execution;
- WebSockets;
- A2A;
- embedded LLM reasoning.

## Next

After H21, Specview can add explicit peer configuration and freshness-aware polling while continuing to aggregate immutable HostSnapshot v1 documents locally.
