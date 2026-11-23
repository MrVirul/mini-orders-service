#!/usr/bin/env bash
#
# Sanity-checks the local database setup: schemas are identical across tenants,
# seed data is in place, and each tenant user can only reach its own database.
#
# Usage:
#   ./scripts/db-verify.sh
#
set -euo pipefail

CONTAINER="${MYSQL_CONTAINER:-mini-orders-mysql}"
ROOT_USER="${MYSQL_ROOT_USER:-root}"
ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root_local_pw}"

failure=0

sql() {
    local database="$1"
    shift
    docker exec -e MYSQL_PWD="$ROOT_PASSWORD" -i "$CONTAINER" mysql -u "$ROOT_USER" -N -B "$database" -e "$*"
}

as_user() {
    local user="$1"
    local password="$2"
    local database="$3"
    shift 3
    docker exec -e MYSQL_PWD="$password" -i "$CONTAINER" mysql -u "$user" -N -B "$database" -e "$*" 2>/dev/null
}

check() {
    local description="$1"
    local actual="$2"
    local expected="$3"
    if [[ "$actual" == "$expected" ]]; then
        printf 'ok   %s\n' "$description"
    else
        printf 'FAIL %s (expected %s, got %s)\n' "$description" "$expected" "$actual" >&2
        failure=1
    fi
}

check_denied() {
    local description="$1"
    shift
    if "$@" >/dev/null 2>&1; then
        printf 'FAIL %s (access was allowed)\n' "$description" >&2
        failure=1
    else
        printf 'ok   %s\n' "$description"
    fi
}

echo "== master database =="
check "tenants registry is seeded" "$(sql mini_master 'SELECT COUNT(*) FROM tenants')" "3"
check "all tenants are active" "$(sql mini_master 'SELECT COUNT(*) FROM tenants WHERE is_active = 1')" "3"

TENANT_DBS=(
    "tenant_pizza_palace"
    "tenant_sushi_house"
    "tenant_burger_barn"
)

for db in "${TENANT_DBS[@]}"; do
    case "$db" in
        tenant_pizza_palace)
            expected_menu=3
            expected_items=1
            expected_order="TAKEAWAY|CONFIRMED|1798"
            ;;
        tenant_sushi_house)
            expected_menu=3
            expected_items=2
            expected_order="DINE_IN|PREPARING|1000"
            ;;
        tenant_burger_barn)
            expected_menu=3
            expected_items=2
            expected_order="DELIVERY|PENDING|1300"
            ;;
        *)
            echo "Unexpected tenant database: $db" >&2
            exit 1
            ;;
    esac

    echo
    echo "== $db =="
    check "tables exist" \
        "$(sql "$db" "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = '$db' AND table_name IN ('menu_items','orders','order_items')")" \
        "3"
    check "menu_items seeded" "$(sql "$db" 'SELECT COUNT(*) FROM menu_items')" "$expected_menu"
    check "order_items seeded" "$(sql "$db" 'SELECT COUNT(*) FROM order_items')" "$expected_items"
    check "order 1 has the expected tenant signature" \
        "$(sql "$db" 'SELECT CONCAT(order_type, "|", status, "|", total_minor) FROM orders WHERE id = 1')" \
        "$expected_order"
done

echo
echo "== tenant isolation =="
check "pizza user reads own database" \
    "$(as_user pizza_palace_user pizza_local_pw tenant_pizza_palace 'SELECT COUNT(*) FROM orders')" \
    "1"
check_denied "pizza user cannot read sushi orders" \
    as_user pizza_palace_user pizza_local_pw tenant_pizza_palace 'SELECT COUNT(*) FROM tenant_sushi_house.orders'

check "sushi user reads own database" \
    "$(as_user sushi_house_user sushi_local_pw tenant_sushi_house 'SELECT COUNT(*) FROM orders')" \
    "1"
check_denied "sushi user cannot read burger orders" \
    as_user sushi_house_user sushi_local_pw tenant_sushi_house 'SELECT COUNT(*) FROM tenant_burger_barn.orders'

check "burger user reads own database" \
    "$(as_user burger_barn_user burger_local_pw tenant_burger_barn 'SELECT COUNT(*) FROM orders')" \
    "1"
check_denied "burger user cannot read pizza orders" \
    as_user burger_barn_user burger_local_pw tenant_burger_barn 'SELECT COUNT(*) FROM tenant_pizza_palace.orders'

echo
echo "== master reader =="
check "master reader reads registry" \
    "$(as_user master_reader master_local_pw mini_master 'SELECT COUNT(*) FROM tenants')" \
    "3"
check_denied "master reader cannot write the registry" \
    as_user master_reader master_local_pw mini_master "UPDATE tenants SET site_name = 'x'"

if (( failure )); then
    echo
    echo "Verification failed." >&2
    exit 1
fi

echo
echo "All checks passed."
