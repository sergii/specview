---
specview:
  status: in_progress
---

# H19 - Host Identity and Federation Contract

## Goal

Define the stable identity semantics required for multi-host Specview before implementing any federation transport.

H19 must let Specview distinguish:

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

The two instances may be active concurrently and must remain separately observable while correlating to one logical Repository only when evidence is sufficient.

## Host identity

Each Specview installation gets one persistent opaque ID:

```text
host:<uuid>
```

Requirements:

- generated once with cryptographically secure randomness;
- persisted independently from `catalog.json`;
- stable across restarts;
- stable when hostname changes;
- versioned language-neutral JSON representation;
- invalid/corrupt identity files fail explicitly rather than being silently replaced.

Hostname remains a mutable runtime label.

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

The value is opaque to Specview. It is a strong user-controlled identity hint and must remain backward compatible with existing version-1 configs that omit it.

## Repository correlation

Normalize and compare:

- explicit project ID;
- repository name;
- Git remote;
- forge provider + forge repository.

Correlation result:

```text
match
ambiguous
distinct
conflict
```

No caller may treat `ambiguous` or `conflict` as `match`.

## Contract fixtures

Add language-neutral fixtures for:

- host identity v1;
- repository correlation cases;
- config with explicit project ID.

The fixtures are part of the future Go/Rust parity gate.

## Acceptance criteria

- [ ] separate versioned Host identity persistence exists;
- [ ] Host identity survives reopen and hostname changes;
- [ ] corrupt/unsupported Host identity fails explicitly;
- [ ] RepositoryInstance ID is deterministic from Host ID + canonical root;
- [ ] `project.id` is supported as an optional backward-compatible config field;
- [ ] repository name normalization is deterministic;
- [ ] common Git SSH/HTTPS remote forms normalize consistently;
- [ ] correlation returns `match`, `ambiguous`, `distinct`, or `conflict` according to ADR-008;
- [ ] contradictory evidence never silently merges repositories;
- [ ] language-neutral Host/config/correlation fixtures are tested;
- [ ] existing catalog v1 contract remains unchanged;
- [ ] gofmt, module, vet, race, coverage, browser, and release gates pass.

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
