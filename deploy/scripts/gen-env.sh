#!/usr/bin/env bash
# Creates app.env with random secrets. Never overwrites an existing file.
# Usage: deploy/scripts/gen-env.sh <output-path> [public-url]
set -euo pipefail

out="${1:?usage: gen-env.sh <output-path> [public-url]}"
public_url="${2:-http://lms.trungvuong.test:8880}"

if [ -e "$out" ]; then
  echo "$out already exists, not overwriting." >&2
  exit 1
fi

rand() { openssl rand -hex "${1:-24}"; }

umask 077
cat > "$out" <<EOF
# Generated $(date -u +%Y-%m-%dT%H:%M:%SZ). Keep this file secret (chmod 600). Never commit it.
PUBLIC_URL=$public_url
HTTP_PORT=8880
SITE_ADDRESS=:80

POSTGRES_PASSWORD=$(rand)
LMS_DB_PASSWORD=$(rand)
KEYCLOAK_DB_PASSWORD=$(rand)
KC_BOOTSTRAP_ADMIN_PASSWORD=$(rand 16)
LMS_WEB_CLIENT_SECRET=$(rand)
SESSION_SECRET=$(rand 32)
AI_SERVICE_TOKEN=$(rand)
AI_PROVIDER=mock

# Demo data + demo Keycloak users. Set to false before real users go live.
SEED_DEMO=true
DEMO_USER_PASSWORD=Demo-$(rand 6)
EOF
chmod 600 "$out"
echo "Wrote $out"
