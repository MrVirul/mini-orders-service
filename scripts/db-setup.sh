#!/usr/bin/env bash
#
# Applies every database migration and seed file to the local MySQL container.
# Safe to re-run: databases, tables, users and seed rows are all applied
# idempotently.
#
# Usage:
#   ./scripts/db-setup.sh
#
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CONTAINER="${MYSQL_CONTAINER:-mini-orders-mysql}"
ROOT_USER="${MYSQL_ROOT_USER:-root}"
ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root_local_pw}"

MASTER_DB="mini_master"
TENANT_DBS=(
    "tenant_pizza_palace"
    "tenant_sushi_house"
    "tenant_burger_barn"
)

run_sql() {
    local file="$1"
    local database="${2:-}"
    docker exec -e MYSQL_PWD="$ROOT_PASSWORD" -i "$CONTAINER" \
        mysql -u "$ROOT_USER" ${database:+"$database"} < "$file"
}

log() {
    printf '==> %s\n' "$*"
}

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
    echo "Container '$CONTAINER' not found. Start it with: docker compose up -d" >&2
    exit 1
fi

log "Waiting for MySQL in $CONTAINER to accept connections"
for _ in $(seq 1 60); do
    if docker exec -e MYSQL_PWD="$ROOT_PASSWORD" "$CONTAINER" mysqladmin ping -h 127.0.0.1 -u "$ROOT_USER" --silent >/dev/null 2>&1; then
        break
    fi
    sleep 1
done
docker exec -e MYSQL_PWD="$ROOT_PASSWORD" "$CONTAINER" mysqladmin ping -h 127.0.0.1 -u "$ROOT_USER" --silent >/dev/null 2>&1 || {
    echo "MySQL in '$CONTAINER' did not become ready in time." >&2
    exit 1
}

log "Creating databases"
run_sql "$ROOT_DIR/migrations/setup/001_databases.sql"

log "Applying master database migration"
run_sql "$ROOT_DIR/migrations/master/001_init.sql"

for db in "${TENANT_DBS[@]}"; do
    log "Applying tenant schema to $db"
    run_sql "$ROOT_DIR/migrations/tenant/001_init.sql" "$db"

    seed="$ROOT_DIR/seeds/$db.sql"
    if [[ -f "$seed" ]]; then
        log "Seeding $db"
        run_sql "$seed" "$db"
    fi
done

log "Creating database users and grants"
run_sql "$ROOT_DIR/migrations/setup/002_users.sql"

log "Database setup complete"
