#!/usr/bin/env bash
set -euo pipefail

bin=${1:-./specview}
root=$(mktemp -d)
server_pid=""
observer_pid=""
cleanup() {
  if [ -n "$observer_pid" ]; then
    kill "$observer_pid" 2>/dev/null || true
    wait "$observer_pid" 2>/dev/null || true
  fi
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$root"
}
trap cleanup EXIT

export XDG_STATE_HOME="$root/state"
mkdir -p "$XDG_STATE_HOME"

"$bin" federation snapshot > "$root/local.json"
host_id=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["host_id"])' "$root/local.json")

"$bin" federation serve > "$root/federation-server.log" 2>&1 &
server_pid=$!
for _ in $(seq 1 50); do
  if curl -fsS http://127.0.0.1:7332/v1/federation/snapshot > /dev/null; then
    break
  fi
  sleep 0.1
done
curl -fsS http://127.0.0.1:7332/v1/federation/snapshot > /dev/null

"$bin" federation peer add cached \
  --url http://127.0.0.1:7332 \
  --host "$host_id" \
  --stale-after 5m
"$bin" federation peer refresh cached > "$root/cached-ok.json"

"$bin" federation peer add never \
  --url http://127.0.0.1:7332 \
  --host host:33333333-3333-4333-9333-333333333333 \
  --stale-after 5m

kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=""

set +e
"$bin" federation peer refresh cached > "$root/cached-failed.json" 2> "$root/cached-failed.err"
refresh_status=$?
set -e
if [ "$refresh_status" -eq 0 ]; then
  echo "expected cached peer refresh to fail after federation server shutdown" >&2
  exit 1
fi

"$bin" federation status > "$root/status.json"
python3 - "$root/status.json" "$host_id" <<'PY'
import json, sys
status = json.load(open(sys.argv[1]))
host_id = sys.argv[2]

assert status["schema_version"] == 1, status
assert status["generated_at"], status
assert status["federation"]["schema_version"] == 1, status
assert status["federation"]["generated_at"] == status["generated_at"], status

hosts = status["hosts"]
assert len(hosts) == 3, hosts
local = hosts[0]
assert local["source"] == "local", local
assert local["host_id"] == host_id, local
assert local["has_snapshot"] is True, local

by_peer = {row.get("peer"): row for row in hosts if row["source"] == "peer"}
cached = by_peer["cached"]
assert cached["host_id"] == host_id, cached
assert cached["freshness"] == "unreachable", cached
assert cached["has_snapshot"] is True, cached
assert cached["observed_at"], cached
assert cached["retrieved_at"], cached
assert cached["last_error"], cached

never = by_peer["never"]
assert never["freshness"] == "never_retrieved", never
assert never["has_snapshot"] is False, never
assert "observed_at" not in never, never

# Local and cached refer to the same Host in this binary fixture. H20 de-duplicates
# the Host by identity instead of inventing a second source Host.
assert len(status["federation"]["hosts"]) == 1, status["federation"]
assert status["federation"]["hosts"][0]["host_id"] == host_id, status["federation"]
PY

# The real host observer must compose the federation polling runtime and shut down
# cleanly with the same process context.
"$bin" serve > "$root/observer.log" 2>&1 &
observer_pid=$!
for _ in $(seq 1 50); do
  if curl -fsS http://127.0.0.1:7331/ > /dev/null; then
    break
  fi
  sleep 0.1
done
curl -fsS http://127.0.0.1:7331/ > /dev/null
kill "$observer_pid"
wait "$observer_pid"
observer_pid=""
