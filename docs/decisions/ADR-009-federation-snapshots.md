# ADR-009: Federate immutable source-host snapshots before adding network transport

- Status: Accepted
- Date: 2026-08-23

## Context

H19 gave every Specview Host a stable identity and defined conservative Repository correlation. The next product requirement is to show one logical repository across multiple machines, for example one active session on a laptop and two active sessions on a DevBox.

The dangerous shortcut would be to start with a shared database or a bidirectional synchronization protocol. That would mix transport, identity, authority, freshness, and merge semantics before the data model is proven.

ADR-008 already states that each Host remains authoritative for facts it directly observes. H20 therefore needs a transport-neutral representation of those facts before choosing HTTP, WebSocket, Tailscale, Cloudflare, or another transport.

## Decision

### HostSnapshot is the first federation wire contract

Each Host can produce a versioned, immutable snapshot:

```text
HostSnapshot v1
  source Host identity
  source hostname label
  observed_at
  RepositoryInstance[]
```

A snapshot is a report from one Host at one observation time. It is not a command and does not grant authority to mutate the source Host.

### RepositoryInstance facts stay source-attributed

Each snapshot instance contains enough normalized facts for the first multi-host view:

```text
RepositoryInstance
  instance_id
  source repository id
  display name
  local root
  repository fingerprint
  active agents
  execution sessions
  Git worktrees
```

The fingerprint uses the H19 contract:

```text
explicit project id
normalized repository name
normalized Git remote
forge provider + repository identity
```

Local filesystem paths remain visible as source-host facts. They are never used as global Repository identity.

### Snapshot construction reuses the control plane

The snapshot builder consumes the normalized local control-plane read model rather than reimplementing process discovery, Git inspection, or forge inspection.

```text
local adapters
     ↓
controlplane.Reader
     ↓
HostSnapshot builder
     ↓
versioned snapshot JSON
```

The builder may additionally read repository configuration only for identity hints such as `project.id` that are not part of the existing MCP v1 contract.

### Aggregation is derived and read-only

A federation aggregator accepts snapshots and produces a projection of logical repositories with their source RepositoryInstances.

It never mutates input snapshots.

For the POC:

1. if multiple snapshots from the same Host are supplied, only the newest snapshot from that Host participates in the current projection;
2. RepositoryInstances are correlated with ADR-008;
3. an instance can join an existing logical group only when at least one member is a `match` and no member comparison is `distinct` or `conflict`;
4. `ambiguous` alone never causes a merge;
5. conflicts remain separately visible rather than being silently collapsed;
6. no durable global Repository ID is introduced in H20.

Logical grouping is therefore a projection that can be recomputed when new snapshots arrive.

### Freshness is explicit

Every source Host snapshot carries `observed_at`.

The aggregator preserves this timestamp per Host and per RepositoryInstance projection. A later live transport can use it to derive freshness/staleness without changing snapshot semantics.

No snapshot disappearance is interpreted as zero activity. The last known snapshot remains historical evidence until explicitly superseded or expired by a projection policy.

### Transport is intentionally deferred

H20 can prove federation using JSON files and in-process aggregation. A later slice may carry the exact same HostSnapshot contract over HTTP or another transport.

Transport authentication, encryption, discovery, retries, and connectivity are separate concerns and must not redefine Repository correlation or source authority.

## Consequences

- laptop and DevBox behavior can be tested deterministically without networking;
- the federation contract is language-neutral and suitable for future Rust parity;
- a shared database is not required for the first federation proof;
- source Host authority is preserved;
- stale remote state cannot masquerade as current local state;
- the UI can eventually show one logical repository with multiple source-host instances and session counts;
- HTTP/Tailscale/Cloudflare transport can be added later without changing the core snapshot format.

## Non-goals

- bidirectional remote mutation;
- distributed consensus;
- globally durable logical Repository IDs;
- peer discovery;
- authentication or OAuth;
- conflict-resolution UI;
- remote execution;
- A2A;
- embedded LLM reasoning.
