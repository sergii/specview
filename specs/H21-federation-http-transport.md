---
specview:
  status: done
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

H21 intentionally has no option to bind this server to a non-loopback address. Tests freeze both the loopback default address and rejection of a positional bind-address argument.

## Pull client

`specview federation pull <url>`:

- accepts remote HTTPS;
- accepts cleartext HTTP only for loopback hosts;
- rejects remote HTTP before sending a request;
- rejects redirects;
- uses a bounded timeout;
- limits response size;
- requires a successful HTTP response;
- requires JSON content type;
- strict-decodes HostSnapshot v1;
- prints validated snapshot JSON to stdout.

Optional peer pinning:

```text
specview federation pull --expect-host host:<uuid> <url>
```

A different returned Host ID must fail. The pull CLI grammar is covered by table-driven tests for missing, duplicate, unknown, and extra arguments.

## Private ingress

H21 does not configure the private network itself.

The supported ready-now deployment path is:

- Tailscale Serve -> localhost federation server.

Tailscale can terminate HTTPS inside the tailnet and proxy the loopback HTTP listener without requiring extra application headers from the Specview client.

Other authenticated TLS reverse proxies can be used only when their authentication mechanism is already satisfied outside `specview federation pull`.

Cloudflare Tunnel can route a public hostname to the localhost listener, but Cloudflare Access service-token authentication requires client request headers. H21 deliberately does not yet provide peer credential or arbitrary request-header configuration, so Cloudflare Access service-token support is not treated as turnkey in this slice.

See `docs/federation-private-ingress.md` for the executable Tailscale flow.

## Acceptance criteria

- [x] localhost-only federation server exists;
- [x] GET snapshot endpoint returns valid HostSnapshot v1;
- [x] endpoint emits no-store JSON responses;
- [x] unsupported methods are rejected;
- [x] snapshot builder errors do not return partial snapshot JSON;
- [x] pull client accepts remote HTTPS;
- [x] pull client allows HTTP only for loopback development/test URLs;
- [x] remote cleartext HTTP is rejected before request;
- [x] redirects are rejected;
- [x] response size is bounded;
- [x] strict H20 snapshot validation is reused;
- [x] expected Host ID pinning works;
- [x] pull CLI grammar edge cases are tested;
- [x] loopback-only listener address is an executable safety contract;
- [x] built-binary localhost serve/pull flow is tested;
- [x] H20 aggregation accepts a pulled snapshot unchanged;
- [x] Tailscale Serve deployment path is documented;
- [x] H18 MCP, H19 identity, and H20 federation fixtures remain unchanged;
- [x] final exact-head gofmt, module, vet, race, coverage, MCP binary, federation binary, browser, and release gates pass.

## Coverage baseline

Completed H21 functional head:

- total production statement coverage: 65.5%;
- `internal/federationhttp`: 76.9%;
- `internal/federation`: 81.8%;
- `internal/identity`: 82.3%;
- `internal/controlplane`: 76.4%;
- `internal/mcpserver`: 75.2%.

## Out of scope

- peer discovery;
- automatic polling;
- persistence of remote snapshots;
- UI for remote Hosts;
- application-level user login;
- peer credential storage;
- Cloudflare Access service-token headers;
- automatic Tailscale/Cloudflare setup;
- push sync;
- remote writes/execution;
- WebSockets;
- A2A;
- embedded LLM reasoning.

## Next

After H21, Specview can add explicit peer configuration, credential providers, and freshness-aware polling while continuing to aggregate immutable HostSnapshot v1 documents locally.
