#!/bin/sh
# Writes the UI's sign-in settings from the container environment, so one image serves any provider
set -eu

json() {
    printf '"%s"' "$(printf '%s' "$1" | sed 's/\\/\\\\/g; s/"/\\"/g')"
}

cat > /usr/share/nginx/html/env.js <<EOF
globalThis.__ARGYLL_ENV__ = {
  AUTH_PROVIDER: $(json "${AUTH_PROVIDER:-}"),
  AUTH_AUTHORITY: $(json "${AUTH_AUTHORITY:-}"),
  AUTH_CLIENT_ID: $(json "${AUTH_CLIENT_ID:-}"),
};
EOF
