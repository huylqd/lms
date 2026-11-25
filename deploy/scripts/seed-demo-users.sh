#!/usr/bin/env bash
# Creates the demo users listed in docs/ARCHITECTURE.md in the Keycloak realm "lms".
# Idempotent: existing users are left alone. Run from the directory that holds compose.yml.
# Required env: KC_BOOTSTRAP_ADMIN_PASSWORD, DEMO_USER_PASSWORD. Optional: COMPOSE (docker compose command).
set -euo pipefail

: "${KC_BOOTSTRAP_ADMIN_PASSWORD:?required}"
: "${DEMO_USER_PASSWORD:?required}"
COMPOSE="${COMPOSE:-docker compose}"
# Git Bash on Windows would rewrite /opt/... container paths into Windows paths.
export MSYS_NO_PATHCONV=1

kcadm() {
  $COMPOSE exec -T keycloak /opt/keycloak/bin/kcadm.sh "$@"
}

kcadm config credentials --server http://localhost:8080/idp --realm master \
  --user kcadmin --password "$KC_BOOTSTRAP_ADMIN_PASSWORD" >/dev/null

# username|role|first name|last name
users=(
  "admin01|admin|Quản trị|Hệ thống"
  "bgh01|school_manager|Ban|Giám hiệu"
  "qlk01|faculty_manager|Quản lý|Khoa QLKD"
  "gv01|lecturer|Nguyễn Văn|An"
  "gv02|lecturer|Trần Thị|Bình"
  "cv01|advisor|Lê Minh|Châu"
)
for i in $(seq -w 1 10); do
  users+=("sv$i|student|Sinh viên|$i")
done

for entry in "${users[@]}"; do
  IFS='|' read -r username role first last <<<"$entry"
  if kcadm get users -r lms -q "username=$username" -q exact=true --fields id 2>/dev/null | grep -q '"id"'; then
    echo "exists  $username"
    continue
  fi
  kcadm create users -r lms -s "username=$username" -s enabled=true \
    -s "firstName=$first" -s "lastName=$last" -s "email=$username@trungvuong.test" -s emailVerified=true >/dev/null
  kcadm set-password -r lms --username "$username" --new-password "$DEMO_USER_PASSWORD" >/dev/null
  kcadm add-roles -r lms --uusername "$username" --rolename "$role" >/dev/null
  echo "created $username ($role)"
done
