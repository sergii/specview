#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./specview}
repo_root=$(pwd)
state_home=$(mktemp -d)
server_pid=''
cleanup() {
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$state_home"
}
trap cleanup EXIT

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

cat > "$fixture_root/specs/H21.md" <<'MARKDOWN'
---
specview:
  status: in_progress
---

# H21 Federation HTTP Smoke
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
            "id": "repo-federation-http-smoke",
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

local_snapshot="$state_home/local.json"
XDG_STATE_HOME="$state_home" "$binary" federation snapshot > "$local_snapshot"
host_id=$(SNAPSHOT="$local_snapshot" python3 - <<'PY'
import json
import os
from pathlib import Path
print(json.loads(Path(os.environ["SNAPSHOT"]).read_text())["host_id"])
PY
)

server_log="$state_home/server.log"
XDG_STATE_HOME="$state_home" "$binary" federation serve >"$server_log" 2>&1 &
server_pid=$!

endpoint_v1='http://127.0.0.1:7332/v1/federation/snapshot'
endpoint_v2='http://127.0.0.1:7332/v2/federation/snapshot'
ready=false
for _ in $(seq 1 50); do
  if curl --fail --silent --show-error "$endpoint_v2" >/dev/null 2>&1; then
    ready=true
    break
  fi
  if ! kill -0 "$server_pid" 2>/dev/null; then
    cat "$server_log" >&2
    echo 'federation HTTP server exited before becoming ready' >&2
    exit 1
  fi
  sleep 0.1
done
if [ "$ready" != true ]; then
  cat "$server_log" >&2
  echo 'federation HTTP server did not become ready' >&2
  exit 1
fi

served_v1="$state_home/served-v1.json"
served_v2="$state_home/served-v2.json"
curl --fail --silent --show-error "$endpoint_v1" > "$served_v1"
curl --fail --silent --show-error "$endpoint_v2" > "$served_v2"

SERVED_V1="$served_v1" SERVED_V2="$served_v2" HOST_ID="$host_id" python3 - <<'PY'
import json
import os
from pathlib import Path

v1 = json.loads(Path(os.environ["SERVED_V1"]).read_text(encoding="utf-8"))
v2 = json.loads(Path(os.environ["SERVED_V2"]).read_text(encoding="utf-8"))

if v1.get("schema_version") != 1 or "control_plane" in v1:
    raise SystemExit(f"frozen v1 endpoint changed: {v1!r}")
if v2.get("schema_version") != 2 or not v2.get("control_plane"):
    raise SystemExit(f"v2 endpoint missing Host control plane: {v2!r}")
if v1.get("host_id") != os.environ["HOST_ID"] or v2.get("host_id") != os.environ["HOST_ID"]:
    raise SystemExit(f"dual-version Host identity mismatch: v1={v1!r} v2={v2!r}")
if v2["control_plane"].get("host") != v2.get("hostname"):
    raise SystemExit(f"v2 control-plane Host authority mismatch: {v2!r}")
PY

pulled="$state_home/pulled.json"
"$binary" federation pull --expect-host "$host_id" "$endpoint_v1" > "$pulled"

PULLED="$pulled" HOST_ID="$host_id" FIXTURE_ROOT="$fixture_root" python3 - <<'PY'
import json
import os
from pathlib import Path

snapshot = json.loads(Path(os.environ["PULLED"]).read_text(encoding="utf-8"))
if snapshot.get("schema_version") != 2:
    raise SystemExit(f"new client did not prefer v2: {snapshot!r}")
if snapshot.get("host_id") != os.environ["HOST_ID"]:
    raise SystemExit(f"unexpected Host ID: {snapshot!r}")
if not snapshot.get("control_plane"):
    raise SystemExit(f"preferred v2 snapshot missing control plane: {snapshot!r}")
instances = snapshot.get("repository_instances", [])
if len(instances) != 1:
    raise SystemExit(f"expected one RepositoryInstance: {instances!r}")
instance = instances[0]
if instance.get("root") != os.environ["FIXTURE_ROOT"]:
    raise SystemExit(f"unexpected root: {instance!r}")
if instance.get("fingerprint", {}).get("explicit_id") != "specview:sergii/specview":
    raise SystemExit(f"missing explicit identity: {instance!r}")
PY

wrong_host='host:22222222-2222-4222-9222-222222222222'
if "$binary" federation pull --expect-host "$wrong_host" "$endpoint_v1" > /dev/null 2>"$state_home/wrong-host.err"; then
  echo 'expected Host ID pin mismatch to fail' >&2
  exit 1
fi
grep -q 'does not match expected' "$state_home/wrong-host.err"

if "$binary" federation pull 'http://example.com/v1/federation/snapshot' > /dev/null 2>"$state_home/cleartext.err"; then
  echo 'expected remote cleartext HTTP to fail' >&2
  exit 1
fi
grep -q 'must use HTTPS' "$state_home/cleartext.err"

projection="$state_home/projection.json"
"$binary" federation aggregate \
  "$pulled" \
  "$repo_root/testdata/contracts/federation/v1-devbox.json" \
  > "$projection"

PROJECTION="$projection" python3 - <<'PY'
import json
import os
from pathlib import Path

projection = json.loads(Path(os.environ["PROJECTION"]).read_text(encoding="utf-8"))
repositories = projection.get("repositories", [])
if len(repositories) != 1:
    raise SystemExit(f"v2 pulled snapshot did not correlate with v1 DevBox fixture: {repositories!r}")
instances = repositories[0].get("instances", [])
if len(instances) != 2:
    raise SystemExit(f"expected two source instances: {instances!r}")
if projection.get("correlation_issues"):
    raise SystemExit(f"unexpected correlation issues: {projection!r}")
PY
