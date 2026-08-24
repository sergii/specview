# ADR-016 - Host Runtime Configuration Boundary

## Status

Accepted.

## Context

The v0.0.1 repository configuration contract accepts a legacy `server` section:

```yaml
server:
  host: 127.0.0.1
  port: 7331
```

Those values never became authoritative for the Host dashboard. `specview serve` owns its listener independently, while Host identity, federation peers, and federation transport state already live outside repositories.

Keeping `server` required in repository validation therefore creates the wrong ownership boundary and forces `specview init` to generate fields that do not control the runtime.

ADR-013 classified this as explicit post-v0.0.1 compatibility debt.

## Decision

Use the backwards-compatible deprecation-reader path rather than introducing repository config v2.

Repository `.specview.yaml` v1 remains readable with an optional legacy `server` section, but:

- repository validation no longer requires `server.host` or `server.port`;
- when the legacy section is present, its existing syntax and value validation remain supported;
- `specview init` stops generating `server` fields;
- the Host dashboard does not read repository `server` values;
- current Host UI listener settings are resolved from Host-level runtime configuration;
- the default UI listener remains `127.0.0.1:7331`;
- `SPECVIEW_UI_HOST` may select only a loopback host;
- `SPECVIEW_UI_PORT` may select a valid TCP port;
- non-loopback UI binding is rejected until a separate authenticated remote-UI design exists.

The federation snapshot server remains independently loopback-bound on its existing port and is not changed by this ADR.

## Compatibility

The frozen v1 repository configuration fixtures remain valid. Existing repositories containing `server` continue to load unchanged.

New repository configuration generated after H27 omits `server`, while retaining `version: 1` because no repository-owned semantic field changed incompatibly.

Repository `server` values are compatibility data only; they never override Host runtime environment settings.

## Security

The Host web UI is currently unauthenticated and intended for local access. H27 must not turn cleanup of repository configuration debt into accidental LAN/public exposure.

Host runtime configuration therefore accepts loopback hosts only. Remote access should continue through an authenticated/private tunnel or a future explicit remote-UI security slice.

## Consequences

- repository Intent configuration owns only repository concerns;
- new configs stop teaching a misleading networking model;
- v0.0.1 configs remain readable;
- Host runtime settings have an explicit independent boundary;
- no repository config version bump is required;
- non-loopback UI serving remains an explicit future decision rather than an incidental environment variable.
