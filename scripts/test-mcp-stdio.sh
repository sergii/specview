#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./specview}
state_home=$(mktemp -d)
trap 'rm -rf "$state_home"' EXIT

repo_root="$state_home/fixture/specview"
mkdir -p "$repo_root/specs" "$repo_root/.specview/evidence" "$state_home/specview"

cat > "$repo_root/.specview.yaml" <<'YAML'
version: 2
project:
  name: MCP Smoke
  root: .
specs:
  adapter: specview
  path: specs
  pattern: "*.md"
acceptance:
  required:
    - unit-tests
YAML

cat > "$repo_root/specs/H18.md" <<'MARKDOWN'
---
specview:
  status: in_progress
---

# H18 MCP Smoke

Exercise the production MCP binary boundary.
MARKDOWN

printf '.specview/\n' > "$repo_root/.gitignore"
git -C "$repo_root" init -q
git -C "$repo_root" config user.email specview@example.test
git -C "$repo_root" config user.name 'Specview Test'
git -C "$repo_root" add .
git -C "$repo_root" commit -qm fixture
head=$(git -C "$repo_root" rev-parse HEAD)

REPO_ROOT="$repo_root" GIT_HEAD="$head" python3 - <<'PY'
import json
import os
from pathlib import Path

root = Path(os.environ["REPO_ROOT"])
head = os.environ["GIT_HEAD"]
record = {
    "version": 1,
    "id": "H18-tests",
    "work_item_id": "H18",
    "revision": f"git:{head}",
    "check": "unit-tests",
    "kind": "test",
    "provider": "binary-smoke",
    "result": "passed",
    "finished_at": "2026-08-23T18:00:00Z",
    "observed_at": "2026-08-23T18:00:00Z",
    "summary": "binary smoke passed",
}
(root / ".specview" / "evidence" / "tests.json").write_text(
    json.dumps(record, indent=2) + "\n", encoding="utf-8"
)
PY

REPO_ROOT="$repo_root" STATE_HOME="$state_home" python3 - <<'PY'
import json
import os
from pathlib import Path

root = os.environ["REPO_ROOT"]
state_home = Path(os.environ["STATE_HOME"])
catalog = {
    "version": 1,
    "repositories": [{
        "id": "repo-mcp-smoke",
        "name": "fixture/specview",
        "root": root,
        "first_seen_at": "2026-08-23T18:00:00Z",
        "last_seen_at": "2026-08-23T18:00:00Z",
        "convention": {
            "adapter": "specview",
            "label": "Specview",
            "path": "specs",
            "recognized": True,
            "supported": True,
        },
        "sessions": [],
    }],
}
(state_home / "specview" / "catalog.json").write_text(
    json.dumps(catalog, indent=2) + "\n", encoding="utf-8"
)
PY

responses=$(
  printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"binary-smoke","version":"1.0.0"}}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_repositories","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"list_work_items","arguments":{"repository_id":"repo-mcp-smoke"}}}' \
    '{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"get_work_item","arguments":{"repository_id":"repo-mcp-smoke","work_item_id":"H18"}}}' \
    '{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"get_evidence","arguments":{"repository_id":"repo-mcp-smoke","work_item_id":"H18"}}}' \
    '{"jsonrpc":"2.0","id":7,"method":"tools/call","params":{"name":"get_acceptance","arguments":{"repository_id":"repo-mcp-smoke","work_item_id":"H18"}}}' \
    '{"jsonrpc":"2.0","id":8,"method":"tools/call","params":{"name":"get_repository_control_plane","arguments":{"repository_id":"repo-mcp-smoke"}}}' \
    '{"jsonrpc":"2.0","id":9,"method":"tools/call","params":{"name":"get_host_control_plane","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":10,"method":"tools/call","params":{"name":"get_federation_status","arguments":{}}}' \
    '{"jsonrpc":"2.0","id":11,"method":"tools/call","params":{"name":"get_execution_history","arguments":{}}}' \
  | XDG_STATE_HOME="$state_home" "$binary" mcp
)

MCP_RESPONSES="$responses" GIT_HEAD="$head" python3 - <<'PY'
import json
import os

lines = [line for line in os.environ["MCP_RESPONSES"].splitlines() if line.strip()]
if len(lines) != 11:
    raise SystemExit(f"expected 11 MCP responses, got {len(lines)}: {lines!r}")

initialize, tools, repositories, work_items, work_item, evidence, acceptance, control_plane, host_control_plane, federation, history = [json.loads(line) for line in lines]
head = os.environ["GIT_HEAD"]
revision = f"git:{head}"

if initialize.get("result", {}).get("protocolVersion") != "2025-11-25":
    raise SystemExit(f"unexpected MCP protocol version: {initialize!r}")

names = [tool.get("name") for tool in tools.get("result", {}).get("tools", [])]
expected = [
    "list_repositories",
    "get_repository",
    "get_host_control_plane",
    "get_repository_control_plane",
    "list_active_sessions",
    "get_execution_history",
    "list_worktrees",
    "list_work_items",
    "get_work_item",
    "get_evidence",
    "get_acceptance",
    "get_federation_status",
    "get_federation_host",
]
if names != expected:
    raise SystemExit(f"unexpected MCP tools: {names!r}")
for tool in tools.get("result", {}).get("tools", []):
    annotations = tool.get("annotations", {})
    if annotations.get("readOnlyHint") is not True or annotations.get("destructiveHint") is not False:
        raise SystemExit(f"tool is not strictly read-only: {tool!r}")

def structured(response, schema_version=1):
    result = response.get("result", {})
    if result.get("isError"):
        raise SystemExit(f"MCP tool failed: {response!r}")
    value = result.get("structuredContent")
    if not isinstance(value, dict) or value.get("schema_version") != schema_version:
        raise SystemExit(f"unexpected structured content: {response!r}")
    return value

repositories_value = structured(repositories)
if not any(item.get("id") == "repo-mcp-smoke" for item in repositories_value.get("repositories", [])):
    raise SystemExit(f"fixture repository missing: {repositories_value!r}")

items = structured(work_items).get("work_items", [])
if len(items) != 1 or items[0].get("work_item_id") != "H18":
    raise SystemExit(f"unexpected WorkItem discovery: {items!r}")
if structured(work_item).get("work_item", {}).get("work_item_id") != "H18":
    raise SystemExit(f"unexpected WorkItem detail: {work_item!r}")
records = structured(evidence).get("records", [])
if len(records) != 1 or records[0].get("revision") != revision or records[0].get("provider") != "binary-smoke":
    raise SystemExit(f"unexpected Evidence: {records!r}")
acceptance_value = structured(acceptance)
if acceptance_value.get("revision", {}).get("revision") != revision or acceptance_value.get("decision", {}).get("state") != "accepted":
    raise SystemExit(f"unexpected Acceptance: {acceptance_value!r}")

control_plane_value = structured(control_plane)
if control_plane_value.get("repository_id") != "repo-mcp-smoke":
    raise SystemExit(f"unexpected control-plane repository: {control_plane_value!r}")
intent = control_plane_value.get("intent", {})
if intent.get("total") != 1 or intent.get("in_progress") != 1 or intent.get("invalid") != 0:
    raise SystemExit(f"unexpected control-plane Intent: {intent!r}")
execution = control_plane_value.get("execution", {})
if execution.get("active") != 0 or execution.get("latest") is not None:
    raise SystemExit(f"unexpected control-plane Execution: {execution!r}")
repo_evidence = control_plane_value.get("evidence", {})
latest_record = repo_evidence.get("latest", {}).get("record", {})
if repo_evidence.get("total") != 1 or repo_evidence.get("passed") != 1 or latest_record.get("revision") != revision or latest_record.get("provider") != "binary-smoke":
    raise SystemExit(f"unexpected control-plane Evidence: {repo_evidence!r}")
repo_acceptance = control_plane_value.get("acceptance", {})
if repo_acceptance.get("configured") is not True or repo_acceptance.get("accepted") != 1 or repo_acceptance.get("revision", {}).get("revision") != revision:
    raise SystemExit(f"unexpected control-plane Acceptance: {repo_acceptance!r}")

host_control_plane_value = structured(host_control_plane)
if host_control_plane_value.get("host") == "":
    raise SystemExit(f"Host identity missing from Host control plane: {host_control_plane_value!r}")
host_intent = host_control_plane_value.get("intent", {})
if host_intent.get("managed_repositories") != 1 or host_intent.get("work_items") != 1 or host_intent.get("in_progress") != 1:
    raise SystemExit(f"unexpected Host control-plane Intent: {host_intent!r}")
host_execution = host_control_plane_value.get("execution", {})
if host_execution.get("active_sessions") != 0 or host_execution.get("active_repositories") != 0 or host_execution.get("has_latest") is not False:
    raise SystemExit(f"unexpected Host control-plane Execution: {host_execution!r}")
host_evidence = host_control_plane_value.get("evidence", {})
host_latest_record = host_evidence.get("latest", {}).get("entry", {}).get("record", {})
if host_evidence.get("total") != 1 or host_evidence.get("passed") != 1 or host_latest_record.get("revision") != revision:
    raise SystemExit(f"unexpected Host control-plane Evidence: {host_evidence!r}")
host_acceptance = host_control_plane_value.get("acceptance", {})
if host_acceptance.get("configured_repositories") != 1 or host_acceptance.get("accepted") != 1 or host_acceptance.get("blocked") != 0:
    raise SystemExit(f"unexpected Host control-plane Acceptance: {host_acceptance!r}")
if host_control_plane_value.get("attention") != []:
    raise SystemExit(f"unexpected Host control-plane attention: {host_control_plane_value!r}")

federation_value = structured(federation, schema_version=2)
hosts = federation_value.get("hosts", [])
if len(hosts) != 1 or hosts[0].get("source") != "local" or hosts[0].get("has_snapshot") is not True:
    raise SystemExit(f"unexpected federation hosts: {hosts!r}")
if hosts[0].get("control_plane") != host_control_plane_value:
    raise SystemExit(f"federation local Host control plane diverged from get_host_control_plane: federation={hosts[0]!r} host={host_control_plane_value!r}")
groups = federation_value.get("federation", {}).get("repositories", [])
if not any(group.get("name") == "fixture/specview" for group in groups):
    raise SystemExit(f"fixture repository missing from federation projection: {groups!r}")
if federation_value.get("federation", {}).get("schema_version") != 1:
    raise SystemExit(f"nested H20 repository projection changed: {federation_value!r}")

history_value = structured(history)
if history_value.get("entries") != []:
    raise SystemExit(f"unexpected execution history: {history_value!r}")
PY

host_id=$(MCP_RESPONSES="$responses" python3 - <<'PY'
import json
import os

lines = [json.loads(line) for line in os.environ["MCP_RESPONSES"].splitlines() if line.strip()]
federation = lines[9].get("result", {}).get("structuredContent", {})
hosts = federation.get("hosts", [])
if len(hosts) != 1 or not hosts[0].get("host_id"):
    raise SystemExit(f"cannot discover exact local Host ID: {hosts!r}")
print(hosts[0]["host_id"])
PY
)

host_response=$(
  printf '{"jsonrpc":"2.0","id":12,"method":"tools/call","params":{"name":"get_federation_host","arguments":{"host_id":"%s"}}}\n' "$host_id" \
  | XDG_STATE_HOME="$state_home" "$binary" mcp
)

MCP_RESPONSES="$responses" MCP_HOST_RESPONSE="$host_response" REPO_ROOT="$repo_root" HOST_ID="$host_id" python3 - <<'PY'
import json
import os

responses = [json.loads(line) for line in os.environ["MCP_RESPONSES"].splitlines() if line.strip()]
selected_response = json.loads(os.environ["MCP_HOST_RESPONSE"])
expected_host = responses[9]["result"]["structuredContent"]["hosts"][0]
result = selected_response.get("result", {})
if result.get("isError"):
    raise SystemExit(f"get_federation_host failed: {selected_response!r}")
selected = result.get("structuredContent", {})
if selected.get("schema_version") != 1:
    raise SystemExit(f"unexpected federation Host result schema: {selected!r}")
host = selected.get("host", {})
if host.get("host_id") != os.environ["HOST_ID"] or host.get("source") != "local" or host.get("has_snapshot") is not True:
    raise SystemExit(f"unexpected exact federation Host: {host!r}")
if not host.get("observed_at"):
    raise SystemExit(f"selected local Host lost observation time: {host!r}")
for key in ("source", "host_id", "hostname", "has_snapshot", "control_plane"):
    if host.get(key) != expected_host.get(key):
        raise SystemExit(f"get_federation_host diverged on {key}: selected={host!r} expected={expected_host!r}")
repositories = selected.get("repositories", [])
if len(repositories) != 1:
    raise SystemExit(f"unexpected selected Host repositories: {repositories!r}")
row = repositories[0]
instance = row.get("instance", {})
if row.get("name") != "fixture/specview" or instance.get("source_repository_id") != "repo-mcp-smoke" or instance.get("root") != os.environ["REPO_ROOT"]:
    raise SystemExit(f"unexpected selected Host repository attribution: {row!r}")
if instance.get("host_id") != os.environ["HOST_ID"]:
    raise SystemExit(f"repository instance belongs to another Host: {instance!r}")
PY

host_file="$state_home/specview/host.json"
if [ ! -f "$host_file" ]; then
  echo "specview mcp did not create persistent host identity" >&2
  exit 1
fi
cp "$host_file" "$state_home/host-before.json"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"binary-smoke-reopen","version":"1.0.0"}}}' \
  | XDG_STATE_HOME="$state_home" "$binary" mcp >/dev/null
cmp "$state_home/host-before.json" "$host_file"
