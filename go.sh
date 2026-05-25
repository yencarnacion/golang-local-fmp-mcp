#!/usr/bin/env bash
# Run the golang-local-fmp-mcp server.
# Stop with Ctrl+C; the Go process handles SIGINT/SIGTERM and shuts down cleanly.
set -euo pipefail

cd "$(dirname "$0")"

if [ ! -f .env ]; then
  echo "No .env file found. Copying .env.example -> .env"
  echo "Edit .env and set FMP_API_KEY before re-running."
  cp .env.example .env
  exit 1
fi

# Forward signals (SIGINT from Ctrl+C, SIGTERM) to the Go binary so it can shut down gracefully.
exec go run -buildvcs=false ./cmd/golang-local-fmp-mcp "$@"
