# Federation private ingress

Specview H21 keeps its federation source server on loopback only:

```text
127.0.0.1:7332
GET /v1/federation/snapshot
```

Do not expose that listener by changing its bind address. Put a private authenticated ingress in front of it instead.

## Tailscale Serve

Tailscale Serve is the supported H21 path because it can proxy a localhost HTTP service and terminate HTTPS inside the tailnet without requiring application-level credentials from `specview federation pull`.

On the source Host, for example the DevBox:

```bash
specview federation serve
```

In another shell on that Host:

```bash
tailscale serve --bg 7332
tailscale serve status
```

Tailscale will report the HTTPS hostname available inside the tailnet. Keep normal tailnet ACL/grant policy around which identities may reach this Host.

Record the source Host identity once:

```bash
specview federation snapshot | jq -r .host_id
```

On the consuming Host, for example the laptop:

```bash
specview federation pull \
  --expect-host host:550e8400-e29b-41d4-a716-446655440000 \
  https://devbox.example.ts.net/v1/federation/snapshot \
  > devbox.json
```

Then combine it with the local snapshot:

```bash
specview federation snapshot > laptop.json
specview federation aggregate laptop.json devbox.json
```

The returned DevBox snapshot is still an immutable H20 `HostSnapshot` v1 document. Tailscale changes only how the bytes travel between Hosts.

## Other reverse proxies

A different authenticated TLS reverse proxy may also publish `http://127.0.0.1:7332`, provided the client can satisfy that proxy's authentication requirements.

H21 does not yet have a peer credential configuration or arbitrary request-header support. In particular, Cloudflare Access service-token authentication requires client credentials in request headers, so it is not a turnkey `specview federation pull` path in H21.

Peer configuration, credential providers, polling, persistence, and freshness policy belong to the next federation slice. They must not change `HostSnapshot` v1 or H19 repository correlation semantics.
