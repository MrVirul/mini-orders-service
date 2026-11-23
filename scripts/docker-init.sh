#!/usr/bin/env bash
#
# Runs inside the MySQL container from /docker-entrypoint-initdb.d on the
# first initialization of the data volume. Applies every migration, seed and
# database user, exactly like scripts/db-setup.sh does from the host.
#
# The host-side equivalent for an already-running database is:
#   make db-setup
#
set -euo pipefail

INIT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

TENANT_DBS=(
    "tenant_pizza_palace"
    "tenant_sushi_house"
    "tenant_burger_barn"
)

mysql_root() {
    local database="${1:-}"
    shift || true
    MYSQL_PWD="${MYSQL_ROOT_PASSWORD:?MYSQL_ROOT_PASSWORD must be set}" \
        mysql --protocol=socket -u root ${database:+"$database"} "$@"
}

run_sql() {
    local file="$1"
    local database="${2:-}"
    echo "[init] applying ${file#"$INIT_DIR"/}${database:+ to $database}"
    mysql_root "$database" < "$file"
}

echo "[init] creating databases"
run_sql "$INIT_DIR/migrations/setup/001_databases.sql"

echo "[init] applying master database migration"
run_sql "$INIT_DIR/migrations/master/001_init.sql"

for db in "${TENANT_DBS[@]}"; do
    run_sql "$INIT_DIR/migrations/tenant/001_init.sql" "$db"

    seed="$INIT_DIR/seeds/$db.sql"
    if [[ -f "$seed" ]]; then
        run_sql "$seed" "$db"
    fi
done

echo "[init] creating database users and grants"
run_sql "$INIT_DIR/migrations/setup/002_users.sql"

echo "[init] database initialization complete"
