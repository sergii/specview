---
specview:
  status: in_progress
---

# H20 - Federation Snapshot POC

## Goal

Prove that two independent Specview Hosts can be represented together as one read-only federated projection while preserving source-host authority and separate RepositoryInstance facts.

The target user-visible idea is:

```text
sergii/specview
  laptop
    1 active session
    /Users/sergii/repos/sergii/specview

  devbox
    2 active sessions
    /home/sergii/repos/sergii/specview
```

The logical grouping is derived. Laptop and DevBox facts remain separately attributable to their source Hosts.

## First wire contract

H20 introduces language-neutral `HostSnapshot` JSON version 1.

A snapshot contains:

- source Host ID;
- source hostname label;
- observation timestamp;
- RepositoryInstances;
- identity fingerprint for each instance;
- active agents/sessions;
- local Git worktrees.

The snapshot is immutable input to federation. It is not a synchronization command.

## Snapshot builder

Build snapshots from the existing normalized local control plane rather than rediscovering processes or Git state.

Repository config may be read to obtain optional H19 `project.id` identity hints without changing the frozen MCP v1 structured-content contract.

## Aggregation

For the POC:

- newest `observed_at` snapshot wins when several snapshots have the same Host ID;
- equal timestamps with different content fail explicitly instead of depending on input order;
- all source timestamps remain visible;
- ADR-008 repository correlation decides whether instances can group;
- a RepositoryInstance joins a group only if it is `match` against every existing group member;
- `ambiguous`, `distinct`, or `conflict` against any group member blocks the join;
- a candidate that fully matches more than one already-distinct group remains separate and surfaces ambiguity rather than bridging those groups transitively;
- no durable global logical Repository ID is invented yet;
- `group_id` is derived projection state for the exact current member set, not canonical Repository identity.

## Executable POC

The first transport is intentionally file/stdout based:

```text
specview federation snapshot > laptop.json
specview federation snapshot > devbox.json
specview federation aggregate laptop.json devbox.json
```

This makes H20 usable between two real machines through a manual copy or `scp` while keeping the snapshot contract independent from future HTTP/WebSocket/Tailscale/Cloudflare transport.

## Contract fixtures

Add language-neutral fixtures for:

1. laptop HostSnapshot v1;
2. DevBox HostSnapshot v1;
3. expected two-host federated projection;
4. same-name ambiguous repositories that remain separate;
5. contradictory explicit identity that remains a conflict;
6. a transitive bridge case that must not collapse explicitly distinct groups.

These fixtures must be consumable by a future Rust implementation.

## Acceptance criteria

- [ ] HostSnapshot v1 model is versioned and strictly validated;
- [ ] snapshot includes stable source Host ID and `observed_at`;
- [ ] RepositoryInstance IDs use the H19 deterministic contract;
- [ ] snapshot builder reuses `controlplane.Reader` facts;
- [ ] optional `project.id` participates in snapshot fingerprints;
- [ ] sessions stay attached to the correct source RepositoryInstance;
- [ ] Git worktrees stay attached to the correct source RepositoryInstance;
- [ ] multiple snapshots from one Host resolve to the newest snapshot for current projection;
- [ ] same-time conflicting Host snapshots fail explicitly;
- [ ] laptop + DevBox matching instances group into one derived logical repository;
- [ ] the grouped projection still exposes both Host IDs and both local roots;
- [ ] same-name-only repositories remain separate/ambiguous;
- [ ] contradictory evidence never silently merges instances;
- [ ] transitive correlation cannot bridge already-distinct groups;
- [ ] language-neutral snapshot and aggregation fixtures are tested;
- [ ] `specview federation snapshot` emits valid HostSnapshot v1 JSON;
- [ ] `specview federation aggregate` consumes snapshot files and emits the derived projection;
- [ ] existing H18 MCP v1 contract remains unchanged;
- [ ] existing catalog v1 and Host identity v1 contracts remain unchanged;
- [ ] gofmt, module, vet, race, coverage, MCP binary, federation binary, browser, and release gates pass.

## Out of scope

- live HTTP/WebSocket transport;
- discovery;
- authentication;
- remote writes;
- distributed database;
- durable global Repository IDs;
- conflict-resolution UI;
- remote execution;
- A2A;
- embedded LLM reasoning.

## Next

After the snapshot POC is proven, the next federation slice can transport the same contract between laptop and DevBox over an authenticated local/private network channel without changing identity semantics.
