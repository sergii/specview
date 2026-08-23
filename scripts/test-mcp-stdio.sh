#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./specview}
state_home=$(mktemp -d)
trap 'rm -rf "$state_home"' EXIT

repo_root="$state_home/fixture/specview"
mkdir -p "$repo_root/specs" "$repo_root/.specview/evidence" "$state_home/specview"

cat > "$repo_root/.specview.yaml" <<'YAML'
version: 1
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
server:
  host: 127.0.0.1
  port: 7331
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
    json.dumps(record, indent=2) + "\n",
    encoding="utf-8",
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
    "repositories": [
        {
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
        }
    ],
}
(state_home / "specview" / "catalog.json").write_text(
    json.dumps(catalog, indent=2) + "\n",
    encoding="utf-8",
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
  | XDG_STATE_HOME="$state_home" "$binary" mcp
)

MCP_RESPONSES="$responses" GIT_HEAD="$head" python3 - <<'PY'
import json
import os

lines = [line for line in os.environ["MCP_RESPONSES"].splitlines() if line.strip()]
if len(lines) != 7:
    raise SystemExit(f"expected 7 MCP responses, got {len(lines)}: {lines!r}")

initialize, tools, repositories, work_items, work_item, evidence, acceptance = [json.loads(line) for line in lines]
head = os.environ["GIT_HEAD"]
revision = f"git:{head}"

if initialize.get("id") != 1:
    raise SystemExit(f"unexpected initialize response: {initialize!r}")
if initialize.get("result", {}).get("protocolVersion") != "2025-11-25":
    raise SystemExit(f"unexpected MCP protocol version: {initialize!r}")

names = [tool.get("name") for tool in tools.get("result", {}).get("tools", [])]
expected = [
    "list_repositories",
    "get_repository",
    "list_active_sessions",
    "list_worktrees",
    "list_work_items",
    "get_work_item",
    "get_evidence",
    "get_acceptance",
]
if names != expected:
    raise SystemExit(f"unexpected MCP tools: {names!r}")

for tool in tools.get("result", {}).get("tools", []):
    annotations = tool.get("annotations", {})
    if annotations.get("readOnlyHint") is not True:
        raise SystemExit(f"tool is not read-only: {tool!r}")
    if annotations.get("destructiveHint") is not False:
        raise SystemExit(f"tool is destructive: {tool!r}")

def structured(response):
    result = response.get("result", {})
    if result.get("isError"):
        raise SystemExit(f"MCP tool failed: {response!r}")
    value = result.get("structuredContent")
    if not isinstance(value, dict) or value.get("schema_version") != 1:
        raise SystemExit(f"unexpected structured content: {response!r}")
    return value

repositories_value = structured(repositories)
if not any(item.get("id") == "repo-mcp-smoke" for item in repositories_value.get("repositories", [])):
    raise SystemExit(f"fixture repository missing: {repositories_value!r}")

work_items_value = structured(work_items)
items = work_items_value.get("work_items", [])
if len(items) != 1 or items[0].get("work_item_id") != "H18":
    raise SystemExit(f"unexpected WorkItem discovery: {work_items_value!r}")

work_item_value = structured(work_item)
if work_item_value.get("work_item", {}).get("work_item_id") != "H18":
    raise SystemExit(f"unexpected WorkItem: {work_item_value!r}")

evidence_value = structured(evidence)
records = evidence_value.get("records", [])
if len(records) != 1 or records[0].get("revision") != revision or records[0].get("provider") != "binary-smoke":
    raise SystemExit(f"unexpected Evidence: {evidence_value!r}")

acceptance_value = structured(acceptance)
if acceptance_value.get("revision", {}).get("revision") != revision:
    raise SystemExit(f"unexpected Acceptance revision: {acceptance_value!r}")
if acceptance_value.get("decision", {}).get("state") != "accepted":
    raise SystemExit(f"unexpected Acceptance decision: {acceptance_value!r}")
PY

host_file="$state_home/specview/host.json"
if [ ! -f "$host_file" ]; then
  echo "specview mcp did not create persistent host identity" >&2
  exit 1
fi

HOST_FILE="$host_file" python3 - <<'PY'
import json
import os
from pathlib import Path

value = json.loads(Path(os.environ["HOST_FILE"]).read_text(encoding="utf-8"))
if value.get("version") != 1:
    raise SystemExit(f"unexpected host identity version: {value!r}")
if not str(value.get("id", "")).startswith("host:"):
    raise SystemExit(f"unexpected host identity id: {value!r}")
if not value.get("created_at"):
    raise SystemExit(f"host identity created_at missing: {value!r}")
PY

cp "$host_file" "$state_home/host-before.json"
printf '%s\n' \
  '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25","capabilities":{},"clientInfo":{"name":"binary-smoke-reopen","version":"1.0.0"}}}' \
  | XDG_STATE_HOME="$state_home" "$binary" mcp >/dev/null
cmp "$state_home/host-before.json" "$host_file"
