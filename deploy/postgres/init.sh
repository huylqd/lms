#!/bin/sh
# Runs once, on the first start of an empty data volume.
set -eu

psql -v ON_ERROR_STOP=1 --username postgres \
  -v lms_pw="$LMS_DB_PASSWORD" -v kc_pw="$KEYCLOAK_DB_PASSWORD" <<'SQL'
CREATE ROLE lms LOGIN PASSWORD :'lms_pw';
CREATE DATABASE lms OWNER lms;
CREATE ROLE keycloak LOGIN PASSWORD :'kc_pw';
CREATE DATABASE keycloak OWNER keycloak;
REVOKE ALL ON DATABASE lms FROM PUBLIC;
REVOKE ALL ON DATABASE keycloak FROM PUBLIC;
SQL
