#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

export API_HOST="${API_HOST:-0.0.0.0}"
export API_PORT="${API_PORT:-8080}"
export WEBHOOK_BASE_URL="${WEBHOOK_BASE_URL:-http://localhost:$API_PORT}"
export DEV_MODE="${DEV_MODE:-true}"
export LOG_LEVEL="${LOG_LEVEL:-info}"
export POSTGRES_URL="${POSTGRES_URL:-postgres://localhost:5432/postgres?sslmode=disable}"
export POSTGRES_PREFIX="${POSTGRES_PREFIX:-argyll}"

echo "Argyll starting:"
echo "  api: http://localhost:$API_PORT"
echo "  postgres schema: $POSTGRES_PREFIX"

exec go run ./cmd/argyll
