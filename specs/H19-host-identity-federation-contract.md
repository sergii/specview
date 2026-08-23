---
specview:
  status: done
---

# H19 - Host Identity and Federation Contract

## Goal

Define the stable identity semantics required for multi-host Specview before implementing any federation transport.

H19 lets Specview distinguish:

```text
Host
Repository
RepositoryInstance
```

without using hostname, filesystem path, GitHub, or any one forge provider as universal identity.

## Target topology

```text
Repository: sergii/specview
  RepositoryInstance
    Host: laptop
    Root: /Users/sergii/repos/sergii/specview
    Worktrees: ...
    Sessions: ...

  RepositoryInstance
    Host: devbox
    Root: /home/sergii/repos/sergii/specview
    Worktrees: ...
    Sessions: ...
```

The two instances may be active concurrently and remain separately observable while correlating to one logical Repository only when evidence is sufficient.

## Host identity

Each Specview installation gets one persistent opaque ID:

```text
host:<uuid>
```

The identity is generated with cryptographically secure randomness and persisted in versioned `host.json` state next to, but independently from, `catalog.json`. It contains no hostname, so changing the OS hostname does not change Host identity.

Invalid, corrupt, unknown-field, or unsupported-version identity state fails explicitly rather than being silently replaced.

## RepositoryInstance identity

A host-local RepositoryInstance ID is deterministic for:

```text
Host ID + canonical repository root
```

It is not the logical Repository ID and is not intended to survive moving the checkout to another path.

## Explicit Repository identity

Config supports optional:

```yaml
project:
  id: specview:sergii/specview
```

The value is a strong user-controlled identity hint and remains backward compatible with existing version-1 configs that omit it.

## Repository correlation

Correlation normalizes and compares:

- explicit project ID;
- repository name;
- Git remote;
- forge provider + forge repository.

Possible results are:

```text
match
ambiguous
distinct
conflict
```

No caller may treat `ambiguous` or `conflict` as `match`.

Repository-name-only equality remains ambiguous. Matching normalized Git or forge identity can corroborate matching names. Contradictory strong evidence remains distinct, or conflict when it contradicts a shared explicit project ID.

## Federation authority

Host-local facts stay authoritative at their source Host. Federation may cache, correlate, and project those facts, but it must preserve source Host and observation time rather than presenting cached remote state as local observation.

Logical Repository grouping is derived and recomputable. Source RepositoryInstance facts remain intact if correlation changes later.

A disconnected remote Host is stale/unavailable, not evidence of zero activity.

## Contract fixtures

Language-neutral fixtures now cover:

- Host identity v1;
- config v1 with explicit project ID;
- RepositoryInstance deterministic identity;
- repository-name and Git-remote normalization;
- repository correlation outcomes and contradictions.

These fixtures are part of the future Go/Rust parity gate.

## Acceptance criteria

- [x] separate versioned Host identity persistence exists;
- [x] Host identity survives reopen and is independent of hostname;
- [x] corrupt/unsupported Host identity fails explicitly;
- [x] RepositoryInstance ID is deterministic from Host ID + canonical root;
- [x] `project.id` is supported as an optional backward-compatible config field;
- [x] repository name normalization is deterministic;
- [x] common Git SSH/HTTPS remote forms normalize consistently;
- [x] correlation returns `match`, `ambiguous`, `distinct`, or `conflict` according to ADR-008;
- [x] contradictory evidence never silently merges repositories;
- [x] language-neutral Host/config/correlation fixtures are tested;
- [x] existing catalog v1 contract remains unchanged;
- [x] gofmt, module, vet, race, coverage, browser, MCP binary, and release gates pass.

## Verification

Functional H19 head passed:

```text
production statement coverage: 65.5%
internal/identity:             84.2%
internal/config:               80.7%
internal/controlplane:         76.4%
internal/mcpserver:            75.2%
```

The real MCP binary creates `host.json`, reopens the same state in a second process, and verifies that the persisted Host identity does not change.

## Out of scope

- network federation transport;
- HTTP/WebSocket synchronization;
- service discovery;
- remote writes;
- conflict-resolution UI;
- multiple forge mirrors;
- A2A;
- embedded LLM reasoning.

## Next

H20 will use this contract to build a federation POC between two Specview hosts while keeping local facts authoritative at their source Host.
