#!/usr/bin/env bash
set -euo pipefail

bin=${1:-./specview}
root=$(mktemp -d)
server_pid=""
cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$root"
}
trap cleanup EXIT

export XDG_STATE_HOME="$root/state"
mkdir -p "$XDG_STATE_HOME"

"$bin" federation snapshot > "$root/source.json"
host_id=$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["host_id"])' "$root/source.json")

"$bin" federation serve > "$root/server.log" 2>&1 &
server_pid=$!
for _ in $(seq 1 50); do
  if curl -fsS http://127.0.0.1:7332/v1/federation/snapshot > /dev/null; then
    break
  fi
  sleep 0.1
done
curl -fsS http://127.0.0.1:7332/v1/federation/snapshot > /dev/null

export SPECVIEW_PEER_TEST_TOKEN='peer-secret-must-not-persist'
"$bin" federation peer add local-devbox \
  --url http://127.0.0.1:7332 \
  --host "$host_id" \
  --stale-after 5m \
  --header-env Authorization=SPECVIEW_PEER_TEST_TOKEN

"$bin" federation peer list > "$root/list-before.json"
python3 - "$root/list-before.json" "$host_id" <<'PY'
import json, sys
rows = json.load(open(sys.argv[1]))
assert len(rows) == 1, rows
row = rows[0]
assert row["name"] == "local-devbox", row
assert row["expected_host_id"] == sys.argv[2], row
assert row["status"] == "never_retrieved", row
assert row["credential_refs"]["Authorization"] == "SPECVIEW_PEER_TEST_TOKEN", row
PY

"$bin" federation peer refresh local-devbox > "$root/refresh-ok.json"
python3 - "$root/refresh-ok.json" "$host_id" <<'PY'
import json, sys
row = json.load(open(sys.argv[1]))
assert row["status"] == "fresh", row
assert row["snapshot"]["host_id"] == sys.argv[2], row
assert row["retrieved_at"], row
PY

if grep -R -F "$SPECVIEW_PEER_TEST_TOKEN" "$XDG_STATE_HOME" >/dev/null 2>&1; then
  echo "credential secret leaked into Specview state" >&2
  exit 1
fi

kill "$server_pid"
wait "$server_pid" 2>/dev/null || true
server_pid=""

set +e
"$bin" federation peer refresh local-devbox > "$root/refresh-failed.json" 2> "$root/refresh-failed.err"
refresh_status=$?
set -e
if [ "$refresh_status" -eq 0 ]; then
  echo "expected peer refresh to fail after server shutdown" >&2
  exit 1
fi
python3 - "$root/refresh-failed.json" "$host_id" <<'PY'
import json, sys
row = json.load(open(sys.argv[1]))
assert row["status"] == "unreachable", row
assert row["snapshot"]["host_id"] == sys.argv[2], row
assert row["retrieved_at"], row
assert row["last_error"], row
PY

"$bin" federation peer show local-devbox > "$root/show.json"
python3 - "$root/show.json" "$host_id" <<'PY'
import json, sys
row = json.load(open(sys.argv[1]))
assert row["status"] == "unreachable", row
assert row["snapshot"]["host_id"] == sys.argv[2], row
PY

"$bin" federation peer remove local-devbox
"$bin" federation peer list > "$root/list-after.json"
python3 - "$root/list-after.json" <<'PY'
import json, sys
assert json.load(open(sys.argv[1])) == []
PY
