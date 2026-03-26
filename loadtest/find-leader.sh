#!/usr/bin/env bash

set -euo pipefail

ports=(8081 8082 8083)

for port in "${ports[@]}"; do
	if response="$(curl -sS -D - "http://localhost:${port}/health" -o /dev/null 2>/dev/null)"; then
		if grep -qi '^X-Argyll-Raft-State: leader' <<<"$response"; then
			printf 'http://localhost:%s\n' "$port"
			exit 0
		fi
	fi
done

echo "no leader found on localhost:8081-8083" >&2
exit 1
