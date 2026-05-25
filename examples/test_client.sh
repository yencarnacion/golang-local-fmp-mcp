#!/usr/bin/env bash
# Smoke test against a running golang-local-fmp-mcp server.
# Usage: examples/test_client.sh [URL]
set -euo pipefail
URL="${1:-http://127.0.0.1:8086/mcp}"

call() {
  local id="$1"
  local payload="$2"
  local body
  echo "----- id=$id -----"
  body="$(curl -fsS "$URL" -H 'Content-Type: application/json' -d "$payload")"
  if command -v jq >/dev/null 2>&1; then
    jq . <<<"$body" || printf '%s\n' "$body"
  else
    printf '%s\n' "$body"
  fi
  echo
}

call 1 '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
call 2 '{"jsonrpc":"2.0","id":2,"method":"tools/list"}'
call 3 '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"quote","arguments":{"endpoint":"quote","symbol":"AAPL"}}}'
call 4 '{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"search","arguments":{"endpoint":"search-symbol","query":"AAPL"}}}'
