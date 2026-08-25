---
specview:
  status: done
---

# H41 - Federation Host Detail

## Goal

Add an exact, read-only Host drill-down over the existing H40 federation runtime projection so one local or remote Host can be inspected across source/freshness metadata, Intent, logical Execution, native Evidence, Acceptance, factual attention, and repository instances without introducing a new federation authority or wire model.

## Acceptance

- [x] `GET /federation/host?host=<exact-host-id>` resolves exactly one Host from the current H40 federation runtime projection.
- [x] Missing `host` returns 400; an unknown Host ID returns 404; no fuzzy Host matching or fallback is introduced.
- [x] The handler builds the existing federation projection once and derives the page only from that projection.
- [x] The page preserves Host source attribution, peer name, freshness, snapshot availability, observed/retrieved/attempt/success timestamps, source age, and last retrieval error when present.
- [x] When `control_plane` is available, the page renders the existing four Host facets - Intent, Execution, Evidence, and Acceptance - without recomputing or changing H38/H39 semantics.
- [x] Intent shows managed repositories, WorkItems, new, in-progress, done, invalid, and unavailable counts.
- [x] Execution shows active sessions, active repositories, and the existing latest logical session when available.
- [x] Evidence shows total, passed, failed/error, invalid, affected repositories, unavailable, and the existing latest Evidence context when available.
- [x] Acceptance shows configured/unconfigured repositories, accepted, waiting, blocked, unconfigured, invalid, evaluation-pending, and unavailable repository counts.
- [x] When `control_plane` is unavailable, the page says so explicitly and does not render invented zero/healthy facet values.
- [x] The page renders the existing factual attention list in its existing deterministic order without adding severity, ranking, or a synthetic Host health score.
- [x] Repository instances belonging to the selected Host are listed from the nested H20 repository projection and link to the existing `/federation/repository` snapshot drill-down.
- [x] Attention rows link to an exact repository instance only when the selected Host has exactly one instance with the same `source_repository_id`; ambiguous or missing matches remain plain factual rows.
- [x] Local repository navigation continues to be owned by the existing federation repository detail; H41 does not invent direct remote-to-local links.
- [x] The Federation Host rows link to their exact Host detail page.
- [x] H41 changes no HostSnapshot version, federation runtime schema, H20 repository projection, repository correlation semantics, peer observation persistence, Host control-plane contract, MCP contract, Evidence contract, Acceptance contract, execution-history contract, or repository config contract.
- [x] Go tests cover local/remote source metadata, complete control-plane rendering, unavailable control plane, exact repository/attention mapping, missing Host ID, and unknown Host ID.
- [x] Chromium E2E covers Federation -> cached-unreachable Host detail -> repository snapshot drill-down while preserving the H40 cached control-plane facts.
- [x] Formatting, modules, vet, race, coverage, build, binary smokes, browser E2E, and release cross-build pass.

## Verification

Functional implementation head `92bccc309150a0d44120e56f6f006beb51ab8672` passed CI #1352 (`32819250882`) with formatting, module verification, vet, race tests, coverage gate, build, MCP and federation binary smokes, Chromium semantic tests, release archives, and installation command all green.

Production statement coverage on that head is 64.4% total; `internal/web` is 53.6%. Existing federation layers remain at `internal/federation` 81.7%, `internal/federationhttp` 77.5%, `internal/federationruntime` 81.7%, and `internal/mcpserver` 77.2%.

Go coverage verifies exact Host lookup, one projection build, local and remote source attribution, explicit unavailable control plane, complete four-facet rendering, and conservative attention-to-repository linking. Chromium verifies the cached-unreachable `e2e-devbox` path from Federation to Host detail and then to the existing remote repository snapshot while preserving the H40 source facts.

## Non-goals

- remote writes or execution;
- a federation-wide or per-Host synthetic health score;
- severity ranking or alert policy;
- recomputing remote Host control-plane facts locally;
- changing v1/v2 federation transport;
- adding a new MCP tool in this slice;
- remote execution-history expansion beyond the facts already carried by H40.
