---
specview:
  status: in_progress
---

# H27 - Host Runtime Configuration Boundary

## Goal

Close the final compatibility-debt item recorded by ADR-013 by removing Host networking ownership from newly generated repository configuration while preserving v0.0.1 readers.

```text
repository config                Host runtime
---------------                  ------------
project / specs / acceptance     UI listener
legacy server (read only)   ->   explicit Host env
```

## Required semantics

- `.specview.yaml` remains version 1;
- existing v1 `server.host` / `server.port` remain readable;
- legacy server values remain validated when supplied;
- `server` is optional and never required for repository validity;
- `specview init` never generates `server`;
- repository server values never control the Host dashboard listener;
- Host UI defaults to `127.0.0.1:7331`;
- `SPECVIEW_UI_HOST` is Host-scoped and accepts loopback hosts only;
- `SPECVIEW_UI_PORT` is Host-scoped and accepts ports 1-65535 only;
- invalid Host runtime environment fails before the listener starts;
- IPv4, localhost, and IPv6 loopback addresses are rendered/listened correctly;
- federation HTTP serving remains unchanged on its existing loopback boundary.

## Compatibility

- frozen config v1 contract fixtures remain unchanged and green;
- existing config files containing `server` continue to load;
- configs without `server` load successfully;
- no repository config v2 is introduced;
- no MCP, HostSnapshot, federation, execution, evidence, or acceptance contract changes.

## Acceptance

- [ ] ADR-016 accepted.
- [ ] repository `server` validation is conditional on presence.
- [ ] new init output omits `server` and reloads successfully.
- [ ] existing v1 server fixture still loads unchanged.
- [ ] malformed/partial legacy server sections still fail closed.
- [ ] Host runtime settings are resolved outside repository config.
- [ ] default Host UI address remains `127.0.0.1:7331`.
- [ ] loopback Host override works without repository config.
- [ ] non-loopback Host override is rejected.
- [ ] invalid port override is rejected.
- [ ] repository legacy server values cannot override Host env/default settings.
- [ ] web listener uses correct IPv6 host:port formatting.
- [ ] README current configuration example omits legacy server fields and documents Host env settings.
- [ ] full race/coverage/build/binary/browser/release CI passes.

## Definition of done

H27 is done when repository configuration no longer owns or generates Host networking, v0.0.1 configs remain readable, Host UI configuration has an explicit loopback-safe runtime boundary, and every existing public contract remains green.
