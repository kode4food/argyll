#!/usr/bin/env bash

set -euo pipefail

DEFAULT_API_HOST="0.0.0.0"
DEFAULT_LOG_LEVEL="info"
DEFAULT_API_PORT="8080"
DEFAULT_NODE_2_API_PORT="8081"
DEFAULT_NODE_3_API_PORT="8082"
DEFAULT_NODE_1_RAFT_ADDR="127.0.0.1:9701"
DEFAULT_NODE_2_RAFT_ADDR="127.0.0.1:9702"
DEFAULT_NODE_3_RAFT_ADDR="127.0.0.1:9703"
DEFAULT_NODE_1_FORWARD_ADDR="127.0.0.1:9801"
DEFAULT_NODE_2_FORWARD_ADDR="127.0.0.1:9802"
DEFAULT_NODE_3_FORWARD_ADDR="127.0.0.1:9803"
DEFAULT_CLUSTER_STARTUP_DELAY="2"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

API_HOST="${API_HOST:-$DEFAULT_API_HOST}"
LOG_LEVEL="${LOG_LEVEL:-$DEFAULT_LOG_LEVEL}"
NODE_1_API_PORT="${ARGYLL_NODE_1_API_PORT:-$DEFAULT_API_PORT}"
NODE_2_API_PORT="${ARGYLL_NODE_2_API_PORT:-$DEFAULT_NODE_2_API_PORT}"
NODE_3_API_PORT="${ARGYLL_NODE_3_API_PORT:-$DEFAULT_NODE_3_API_PORT}"
NODE_1_RAFT_ADDR="${ARGYLL_NODE_1_RAFT_ADDR:-$DEFAULT_NODE_1_RAFT_ADDR}"
NODE_2_RAFT_ADDR="${ARGYLL_NODE_2_RAFT_ADDR:-$DEFAULT_NODE_2_RAFT_ADDR}"
NODE_3_RAFT_ADDR="${ARGYLL_NODE_3_RAFT_ADDR:-$DEFAULT_NODE_3_RAFT_ADDR}"
NODE_1_FORWARD_ADDR="${ARGYLL_NODE_1_FORWARD_ADDR:-$DEFAULT_NODE_1_FORWARD_ADDR}"
NODE_2_FORWARD_ADDR="${ARGYLL_NODE_2_FORWARD_ADDR:-$DEFAULT_NODE_2_FORWARD_ADDR}"
NODE_3_FORWARD_ADDR="${ARGYLL_NODE_3_FORWARD_ADDR:-$DEFAULT_NODE_3_FORWARD_ADDR}"
CLUSTER_STARTUP_DELAY="${ARGYLL_CLUSTER_STARTUP_DELAY:-$DEFAULT_CLUSTER_STARTUP_DELAY}"

NODE_IDS=("node-1" "node-2" "node-3")
NODE_API_PORTS=("$NODE_1_API_PORT" "$NODE_2_API_PORT" "$NODE_3_API_PORT")
NODE_RAFT_ADDRS=("$NODE_1_RAFT_ADDR" "$NODE_2_RAFT_ADDR" "$NODE_3_RAFT_ADDR")
NODE_FORWARD_ADDRS=(
	"$NODE_1_FORWARD_ADDR"
	"$NODE_2_FORWARD_ADDR"
	"$NODE_3_FORWARD_ADDR"
)

cluster_pids=()
cluster_nodes=()
cluster_temp_dir=""

cleanup() {
	for pid in "${cluster_pids[@]:-}"; do
		kill "$pid" 2>/dev/null || true
	done

	if ((${#cluster_pids[@]} > 0)); then
		wait "${cluster_pids[@]}" 2>/dev/null || true
	fi

	if [[ -n "$cluster_temp_dir" && -d "$cluster_temp_dir" ]]; then
		rm -rf "$cluster_temp_dir"
	fi
}

trap cleanup EXIT

run_node() {
	local id="$1"
	local api_port="$2"
	local raft_addr="$3"
	local forward_addr="$4"
	local mode="${5:-background}"
	local data_dir="$ARGYLL_RAFT_DATA_DIR/$id"
	local log_dir="$ARGYLL_RAFT_DATA_DIR/logs"
	local log_file="$log_dir/$id.log"

	mkdir -p "$data_dir" "$log_dir"

	if [[ "$mode" == "background" ]]; then
		API_HOST="$API_HOST" \
		API_PORT="$api_port" \
		WEBHOOK_BASE_URL="http://localhost:$api_port" \
		DEV_MODE=true \
		LOG_LEVEL="$LOG_LEVEL" \
		RAFT_NODE_ID="$id" \
		RAFT_BIND_ADDRESS="$raft_addr" \
		RAFT_ADVERTISE_ADDRESS="$raft_addr" \
		RAFT_FORWARD_BIND_ADDRESS="$forward_addr" \
		RAFT_FORWARD_ADVERTISE_ADDRESS="$forward_addr" \
		RAFT_DATA_DIR="$data_dir" \
		RAFT_SERVERS="$RAFT_SERVERS" \
			go run ./cmd/argyll/main.go \
			>"$log_file" 2>&1 &

		cluster_pids+=("$!")
		cluster_nodes+=("$id")
		return
	fi

	API_HOST="$API_HOST" \
	API_PORT="$api_port" \
	WEBHOOK_BASE_URL="http://localhost:$api_port" \
	DEV_MODE=true \
	LOG_LEVEL="$LOG_LEVEL" \
	RAFT_NODE_ID="$id" \
	RAFT_BIND_ADDRESS="$raft_addr" \
	RAFT_ADVERTISE_ADDRESS="$raft_addr" \
	RAFT_FORWARD_BIND_ADDRESS="$forward_addr" \
	RAFT_FORWARD_ADVERTISE_ADDRESS="$forward_addr" \
	RAFT_DATA_DIR="$data_dir" \
	RAFT_SERVERS="$RAFT_SERVERS" \
		go run ./cmd/argyll/main.go
}

assert_cluster_started() {
	local idx
	for idx in "${!cluster_pids[@]}"; do
		if ! kill -0 "${cluster_pids[$idx]}" 2>/dev/null; then
			echo \
				"Argyll node ${cluster_nodes[$idx]} failed to start; see $ARGYLL_RAFT_DATA_DIR/logs/${cluster_nodes[$idx]}.log" \
				>&2
			return 1
		fi
	done
}

if [[ -n "${ARGYLL_RAFT_DATA_DIR:-}" ]]; then
	mkdir -p "$ARGYLL_RAFT_DATA_DIR"
else
	cluster_temp_dir="$(mktemp -d "${TMPDIR:-/tmp}/argyll-cluster.XXXXXX")"
	ARGYLL_RAFT_DATA_DIR="$cluster_temp_dir"
fi

raft_servers_parts=()
for i in "${!NODE_IDS[@]}"; do
	raft_servers_parts+=(
		"${NODE_IDS[$i]}=${NODE_RAFT_ADDRS[$i]}|${NODE_FORWARD_ADDRS[$i]}"
	)
done
RAFT_SERVERS="$(IFS=,; printf '%s' "${raft_servers_parts[*]}")"

for i in 1 2; do
	run_node \
		"${NODE_IDS[$i]}" \
		"${NODE_API_PORTS[$i]}" \
		"${NODE_RAFT_ADDRS[$i]}" \
		"${NODE_FORWARD_ADDRS[$i]}"
done

sleep "$CLUSTER_STARTUP_DELAY"
assert_cluster_started

echo "Argyll Raft cluster starting:"
for i in "${!NODE_IDS[@]}"; do
	echo \
		"  ${NODE_IDS[$i]} api: http://localhost:${NODE_API_PORTS[$i]} raft: ${NODE_RAFT_ADDRS[$i]} forward: ${NODE_FORWARD_ADDRS[$i]}"
done
echo "  data dir: $ARGYLL_RAFT_DATA_DIR"
echo "  follower logs: $ARGYLL_RAFT_DATA_DIR/logs"

run_node \
	"${NODE_IDS[0]}" \
	"${NODE_API_PORTS[0]}" \
	"${NODE_RAFT_ADDRS[0]}" \
	"${NODE_FORWARD_ADDRS[0]}" \
	foreground
