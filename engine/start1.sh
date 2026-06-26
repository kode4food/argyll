#!/usr/bin/env bash

set -euo pipefail

DEFAULT_API_HOST="0.0.0.0"
DEFAULT_LOG_LEVEL="info"
DEFAULT_API_PORT="8080"
DEFAULT_NODE_RAFT_ADDR="127.0.0.1:9701"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

API_HOST="${API_HOST:-$DEFAULT_API_HOST}"
LOG_LEVEL="${LOG_LEVEL:-$DEFAULT_LOG_LEVEL}"
NODE_API_PORT="${ARGYLL_NODE_1_API_PORT:-$DEFAULT_API_PORT}"
NODE_RAFT_ADDR="${ARGYLL_NODE_1_RAFT_ADDR:-$DEFAULT_NODE_RAFT_ADDR}"
NODE_ID="node-1"

node_temp_dir=""

cleanup() {
	if [[ -n "$node_temp_dir" && -d "$node_temp_dir" ]]; then
		rm -rf "$node_temp_dir"
	fi
}

handle_sigint() {
	cleanup
	exit 130
}

handle_sigterm() {
	cleanup
	exit 143
}

trap cleanup EXIT
trap handle_sigint INT
trap handle_sigterm TERM

node_temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/argyll-node.XXXXXX")"
node_bin="$node_temp_dir/argyll"

go build -o "$node_bin" ./cmd/argyll

if [[ -n "${ARGYLL_RAFT_DATA_DIR:-}" ]]; then
	mkdir -p "$ARGYLL_RAFT_DATA_DIR"
else
	ARGYLL_RAFT_DATA_DIR="$node_temp_dir/data"
	mkdir -p "$ARGYLL_RAFT_DATA_DIR"
fi

data_dir="$ARGYLL_RAFT_DATA_DIR/$NODE_ID"
mkdir -p "$data_dir"

RAFT_SERVERS="$NODE_ID=$NODE_RAFT_ADDR"

echo "Argyll single-node cluster starting:"
echo "  $NODE_ID api: http://localhost:$NODE_API_PORT raft: $NODE_RAFT_ADDR"
echo "  data dir: $ARGYLL_RAFT_DATA_DIR"

API_HOST="$API_HOST" \
API_PORT="$NODE_API_PORT" \
WEBHOOK_BASE_URL="http://localhost:$NODE_API_PORT" \
DEV_MODE=true \
LOG_LEVEL="$LOG_LEVEL" \
RAFT_NODE_ID="$NODE_ID" \
RAFT_ADDRESS="$NODE_RAFT_ADDR" \
RAFT_DATA_DIR="$data_dir" \
RAFT_SERVERS="$RAFT_SERVERS" \
	"$node_bin"
