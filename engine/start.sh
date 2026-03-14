#!/usr/bin/env bash

set -euo pipefail

if ! command -v psql >/dev/null 2>&1; then
  echo "psql is required on PATH" >&2
  exit 1
fi

if ! command -v createdb >/dev/null 2>&1; then
  echo "createdb is required on PATH" >&2
  exit 1
fi

db_host="${ARGYLL_POSTGRES_HOST:-127.0.0.1}"
db_port="${ARGYLL_POSTGRES_PORT:-5432}"
db_name="${ARGYLL_POSTGRES_DB:-argyll}"
db_user="${ARGYLL_POSTGRES_USER:-$USER}"
db_sslmode="${ARGYLL_POSTGRES_SSLMODE:-disable}"
db_password="${ARGYLL_POSTGRES_PASSWORD:-}"
db_max_conns="${ARGYLL_POSTGRES_MAX_CONNS:-96}"

base_url="postgres://${db_user}@${db_host}:${db_port}"
admin_url="${ARGYLL_POSTGRES_ADMIN_URL:-${base_url}/postgres?sslmode=${db_sslmode}}"
db_url="${ARGYLL_POSTGRES_URL:-${base_url}/${db_name}?sslmode=${db_sslmode}}"

if [[ -n "${db_password}" ]]; then
  export PGPASSWORD="${db_password}"
fi

db_exists="$(
  psql "${admin_url}" -Atqc \
    "SELECT 1 FROM pg_database WHERE datname = '${db_name}'"
)"

if [[ "${db_exists}" != "1" ]]; then
  createdb -h "${db_host}" -p "${db_port}" -U "${db_user}" "${db_name}"
fi

export CATALOG_POSTGRES_URL="${CATALOG_POSTGRES_URL:-${db_url}}"
export PARTITION_POSTGRES_URL="${PARTITION_POSTGRES_URL:-${db_url}}"
export FLOW_POSTGRES_URL="${FLOW_POSTGRES_URL:-${db_url}}"
export CATALOG_POSTGRES_PREFIX="${CATALOG_POSTGRES_PREFIX:-argyll}"
export PARTITION_POSTGRES_PREFIX="${PARTITION_POSTGRES_PREFIX:-argyll}"
export FLOW_POSTGRES_PREFIX="${FLOW_POSTGRES_PREFIX:-argyll}"
export CATALOG_POSTGRES_MAX_CONNS="${CATALOG_POSTGRES_MAX_CONNS:-${db_max_conns}}"
export PARTITION_POSTGRES_MAX_CONNS="${PARTITION_POSTGRES_MAX_CONNS:-${db_max_conns}}"
export FLOW_POSTGRES_MAX_CONNS="${FLOW_POSTGRES_MAX_CONNS:-${db_max_conns}}"
export API_PORT="${API_PORT:-8080}"
export API_HOST="${API_HOST:-0.0.0.0}"
export WEBHOOK_BASE_URL="${WEBHOOK_BASE_URL:-http://localhost:${API_PORT}}"
export DEV_MODE="${DEV_MODE:-true}"
export LOG_LEVEL="${LOG_LEVEL:-info}"

go run cmd/argyll/main.go
