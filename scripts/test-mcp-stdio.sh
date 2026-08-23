#!/usr/bin/env bash
set -euo pipefail

binary=${1:-./specview}
state_home=$(mktemp -d)
trap 'rm -rf "$state_home"' EXIT

responses=$(
  printf '%s\n' \
    '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-11-25"}}' \
    '{"jsonrpc":"2.0","method":"notifications/initialized"}' \
    '{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' \
    '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"list_repositories","arguments":{}}}' \
  | XDG_STATE_HOME="$state_home" "$binary" mcp
)

MCP_RESPONSES="$responses" python3 - <<'PY'
import json
import os

lines = [line for line in os.environ["MCP_RESPONSES"].splitlines() if line.strip()]
if len(lines) != 3:
    raise SystemExit(f"expected 3 MCP responses, got {len(lines)}: {lines!r}")

initialize, tools, repositories = [json.loads(line) for line in lines]

if initialize.get("id") != 1:
    raise SystemExit(f"unexpected initialize response: {initialize!r}")
if initialize.get("result", {}).get("protocolVersion") != "2025-11-25":
    raise SystemExit(f"unexpected MCP protocol version: {initialize!r}")

names = [tool.get("name") for tool in tools.get("result", {}).get("tools", [])]
expected = ["list_repositories", "get_repository", "list_active_sessions", "list_worktrees"]
if names != expected:
    raise SystemExit(f"unexpected MCP tools: {names!r}")

result = repositories.get("result", {})
if result.get("isError"):
    raise SystemExit(f"list_repositories failed: {repositories!r}")
structured = result.get("structuredContent", {})
if structured.get("schema_version") != 1:
    raise SystemExit(f"unexpected control-plane schema: {structured!r}")
if not isinstance(structured.get("repositories"), list):
    raise SystemExit(f"repositories must be a list: {structured!r}")
PY
