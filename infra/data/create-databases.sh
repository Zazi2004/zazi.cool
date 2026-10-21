#!/bin/sh
# Creates (or updates) one database and one user per app. Idempotent: run by the
# create-databases Job after each new app or password change.
set -eu

until pg_isready -q; do
  echo "waiting for PostgreSQL…"
  sleep 2
done

# create_app_db <name> <password>: the user owns its database; nobody else can connect.
create_app_db() {
  psql -v ON_ERROR_STOP=1 -v app="$1" -v pw="$2" <<'SQL'
SELECT format('CREATE ROLE %I LOGIN', :'app')
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = :'app') \gexec
ALTER ROLE :"app" WITH LOGIN PASSWORD :'pw';
SELECT format('CREATE DATABASE %I OWNER %I', :'app', :'app')
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = :'app') \gexec
REVOKE ALL ON DATABASE :"app" FROM PUBLIC;
GRANT CONNECT, TEMPORARY ON DATABASE :"app" TO :"app";
SQL
  echo "database $1 ready"
}

create_app_db undercover "$UNDERCOVER_DB_PASSWORD"
create_app_db pastebin "$PASTEBIN_DB_PASSWORD"
create_app_db linktree "$LINKTREE_DB_PASSWORD"
