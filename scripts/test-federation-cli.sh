#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./specview}
repo_root=$(pwd)
state_home=$(mktemp -d)
trap 'rm -rf "$state_home"' EXIT

fixture_root="$state_home/fixture/specview"
mkdir -p "$fixture_root/specs" "$state_home/specview"

cat > "$fixture_root/.specview.yaml" <<'YAML'
version: 1
project:
  id: specview:sergii/specview
  name: Specview
  root: .
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
server:
  host: 127.0.0.1
  port: 7331
YAML

cat > "$fixture_root/specs/H20.md" <<'MARKDOWN'
---
specview:
  status: in_progress
---

# H20 Federation CLI Smoke
MARKDOWN

git -C "$fixture_root" init -q
git -C "$fixture_root" config user.email specview@example.test
git -C "$fixture_root" config user.name 'Specview Test'
git -C "$fixture_root" remote add origin https://github.com/sergii/specview.git
git -C "$fixture_root" add .
git -C "$fixture_root" commit -qm fixture

FIXTURE_ROOT="$fixture_root" STATE_HOME="$state_home" python3 - <<'PY'
import json
import os
from pathlib import Path

root = os.environ["FIXTURE_ROOT"]
state_home = Path(os.environ["STATE_HOME"])
catalog = {
    "version": 1,
    "repositories": [
        {
            "id": "repo-federation-smoke",
            "name": "fixture/specview",
            "root": root,
            "first_seen_at": "2026-08-23T20:00:00Z",
            "last_seen_at": "2026-08-23T20:00:00Z",
            "convention": {
                "adapter": "specview",
                "label": "Specview",
                "path": "specs",
                "recognized": True,
                "supported": True,
            },
            "sessions": [],
        }
    ],
}
(state_home / "specview" / "catalog.json").write_text(
    json.dumps(catalog, indent=2) + "\n",
    encoding="utf-8",
)
PY

snapshot_one="$state_home/snapshot-one.json"
snapshot_two="$state_home/snapshot-two.json"
XDG_STATE_HOME="$state_home" "$binary" federation snapshot > "$snapshot_one"
XDG_STATE_HOME="$state_home" "$binary" federation snapshot > "$snapshot_two"

SNAPSHOT_ONE="$snapshot_one" SNAPSHOT_TWO="$snapshot_two" FIXTURE_ROOT="$fixture_root" python3 - <<'PY'
import json
import os
from pathlib import Path

first = json.loads(Path(os.environ["SNAPSHOT_ONE"]).read_text(encoding="utf-8"))
second = json.loads(Path(os.environ["SNAPSHOT_TWO"]).read_text(encoding="utf-8"))
root = os.environ["FIXTURE_ROOT"]

if first.get("schema_version") != 1:
    raise SystemExit(f"unexpected snapshot schema: {first!r}")
if not first.get("host_id", "").startswith("host:"):
    raise SystemExit(f"missing Host identity: {first!r}")
if first.get("host_id") != second.get("host_id"):
    raise SystemExit("Host identity changed between snapshot processes")
instances = first.get("repository_instances", [])
if len(instances) != 1:
    raise SystemExit(f"expected one RepositoryInstance: {instances!r}")
instance = instances[0]
if instance.get("source_repository_id") != "repo-federation-smoke":
    raise SystemExit(f"unexpected source repository: {instance!r}")
if instance.get("root") != root:
    raise SystemExit(f"unexpected local root: {instance!r}")
fingerprint = instance.get("fingerprint", {})
if fingerprint.get("explicit_id") != "specview:sergii/specview":
    raise SystemExit(f"missing explicit project identity: {fingerprint!r}")
if fingerprint.get("git_remote") != "https://github.com/sergii/specview.git":
    raise SystemExit(f"unexpected Git remote: {fingerprint!r}")
PY

projection="$state_home/projection.json"
"$binary" federation aggregate \
  "$repo_root/testdata/contracts/federation/v1-laptop.json" \
  "$repo_root/testdata/contracts/federation/v1-devbox.json" \
  > "$projection"

PROJECTION="$projection" python3 - <<'PY'
import json
import os
from pathlib import Path

projection = json.loads(Path(os.environ["PROJECTION"]).read_text(encoding="utf-8"))
if projection.get("schema_version") != 1:
    raise SystemExit(f"unexpected projection schema: {projection!r}")
if len(projection.get("hosts", [])) != 2:
    raise SystemExit(f"expected two Hosts: {projection!r}")
repositories = projection.get("repositories", [])
if len(repositories) != 1:
    raise SystemExit(f"expected one correlated Repository group: {repositories!r}")
instances = repositories[0].get("instances", [])
if len(instances) != 2:
    raise SystemExit(f"expected laptop + DevBox instances: {instances!r}")
if sum(len(instance.get("sessions", [])) for instance in instances) != 3:
    raise SystemExit(f"expected three source-attributed sessions: {instances!r}")
roots = {instance.get("root") for instance in instances}
expected_roots = {
    "/Users/sergii/repos/sergii/specview",
    "/home/sergii/repos/sergii/specview",
}
if roots != expected_roots:
    raise SystemExit(f"unexpected federated roots: {roots!r}")
if projection.get("correlation_issues"):
    raise SystemExit(f"canonical matching fixtures must not have correlation issues: {projection!r}")
PY
