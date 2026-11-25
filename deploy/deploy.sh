#!/usr/bin/env bash
# Deploys the LMS stack with docker compose, waits for health checks, and rolls back the
# three app images to the previous release if the new one is unhealthy.
#
# Required env: IMAGE_TAG, DEPLOY_DIR
# Optional env: APP_URL (default http://localhost:8880), HEALTH_TIMEOUT (s), KEEP_IMAGES
# Expects $DEPLOY_DIR/app.env to exist (create it once with deploy/scripts/gen-env.sh).
set -euo pipefail

: "${IMAGE_TAG:?IMAGE_TAG is required}"
: "${DEPLOY_DIR:?DEPLOY_DIR is required}"
APP_URL="${APP_URL:-http://localhost:8880}"
HEALTH_TIMEOUT="${HEALTH_TIMEOUT:-240}"
KEEP_IMAGES="${KEEP_IMAGES:-5}"
SERVICES=(core-api ai-service web)

src="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
summary="${GITHUB_STEP_SUMMARY:-/dev/null}"

if [ ! -f "$DEPLOY_DIR/app.env" ]; then
  echo "::error::$DEPLOY_DIR/app.env not found. Create it once on the server: bash deploy/scripts/gen-env.sh $DEPLOY_DIR/app.env"
  exit 1
fi

# Sync deploy files (app.env and images.env are never touched here).
mkdir -p "$DEPLOY_DIR"
cp "$src/compose.yml" "$src/Caddyfile" "$DEPLOY_DIR/"
for dir in keycloak postgres scripts; do
  rm -rf "${DEPLOY_DIR:?}/$dir"
  cp -r "$src/$dir" "$DEPLOY_DIR/$dir"
done
cd "$DEPLOY_DIR"

compose() {
  docker compose --env-file app.env --env-file images.env -p lms "$@"
}

write_images() {
  {
    for svc in "${SERVICES[@]}"; do
      var="$(echo "$svc" | tr 'a-z-' 'A-Z_')_IMAGE"
      echo "$var=lms-$svc:$1"
    done
  } > images.env
}

check() {
  curl -fsS -o /dev/null --max-time 5 "$1"
}

healthy() {
  local deadline=$((SECONDS + HEALTH_TIMEOUT))
  local urls=("$APP_URL/api/v1/readyz" "$APP_URL/api/health" "$APP_URL/idp/realms/lms")
  echo "Waiting up to ${HEALTH_TIMEOUT}s for: ${urls[*]}"
  while :; do
    local ok=1
    for u in "${urls[@]}"; do check "$u" || ok=0; done
    [ "$ok" -eq 1 ] && return 0
    if [ "$SECONDS" -ge "$deadline" ]; then
      for u in "${urls[@]}"; do check "$u" && echo "  ok   $u" || echo "  FAIL $u"; done
      return 1
    fi
    sleep 5
  done
}

previous=""
if [ -f images.env ]; then
  previous="$(sed -n 's/^CORE_API_IMAGE=lms-core-api://p' images.env)"
  cp images.env images.env.prev
fi

echo "Deploying tag $IMAGE_TAG (previous: ${previous:-none})"
write_images "$IMAGE_TAG"
compose up -d --remove-orphans

if healthy; then
  echo "Deployed $IMAGE_TAG"
  echo "### Deployed \`$IMAGE_TAG\`" >> "$summary"
else
  echo "::error::Health check failed for $IMAGE_TAG"
  compose ps || true
  compose logs --tail 80 core-api web keycloak || true
  if [ -n "$previous" ] && [ "$previous" != "$IMAGE_TAG" ]; then
    echo "Rolling back to $previous"
    cp images.env.prev images.env
    compose up -d --remove-orphans
    if healthy; then
      echo "::warning::Rolled back to $previous"
    else
      echo "::error::Rollback to $previous is also unhealthy. Manual action needed."
    fi
  fi
  exit 1
fi

# Demo users in Keycloak (idempotent) when SEED_DEMO=true in app.env.
if grep -qx 'SEED_DEMO=true' app.env; then
  (
    set -a
    # shellcheck disable=SC1091
    . ./app.env
    set +a
    COMPOSE="docker compose --env-file app.env --env-file images.env -p lms" bash scripts/seed-demo-users.sh
  )
fi

# Keep only the newest $KEEP_IMAGES images of each service.
for svc in "${SERVICES[@]}"; do
  docker image ls "lms-$svc" --format '{{.CreatedAt}}|{{.Repository}}:{{.Tag}}' \
    | grep -v ':test$' | sort -r | tail -n +"$((KEEP_IMAGES + 1))" | cut -d'|' -f2 \
    | xargs -r docker image rm >/dev/null 2>&1 || true
done
