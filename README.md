# assignment-01

Multi-tenant order service (Go intern assignment).

## Local database

The local environment runs three MySQL 8.0 containers. Each tenant has its own
server, so a tenant database cannot be reached from another tenant's container.

Provisioning is entirely declarative: the SQL files are mounted into
`/docker-entrypoint-initdb.d` and MySQL runs them on the first start of an empty
volume. There are no custom scripts. `MYSQL_DATABASE` creates each database and
`MYSQL_USER` / `MYSQL_PASSWORD` create each tenant user with access only to its
own database.

| Container                  | Host port | Database              | Purpose                      |
| -------------------------- | --------- | --------------------- | ---------------------------- |
| `mini-orders-mysql-master` | 3306      | `mini_master`         | Master DB — tenants registry |
| `mini-orders-mysql-pizza`  | 3307      | `tenant_pizza_palace` | Tenant 1 — Pizza Palace      |
| `mini-orders-mysql-sushi`  | 3308      | `tenant_sushi_house`  | Tenant 2 — Sushi House       |

The master `tenants` table maps `x_tenant_code` to the tenant database's
`host`, `port`, `database_name`, `username` and `password`. Both tenant
databases use the identical schema (`menu_items`, `orders`, `order_items`) but
have different seed data, so cross-tenant leaks are easy to spot.

### Setup

```bash
cp .env.example .env   # first time only
make db-up             # start all three containers; empty volumes are provisioned automatically
make db-reset          # wipe the volumes and reprovision after any SQL change
```

Init SQL only runs on an empty volume. To re-apply migrations or seeds, use
`make db-reset` (this destroys local data). If 3306 is already used by a local
MySQL server, set `MYSQL_MASTER_PORT` in `.env` to a free port.

### Layout

```
migrations/master/001_init.sql   # mini_master.tenants table
migrations/master/002_user.sql   # read-only master_reader user
migrations/tenant/001_init.sql   # identical schema, applied to every tenant DB
seeds/master_tenants.sql         # tenants registry rows
seeds/tenant_<code>.sql          # per-tenant seed data
```

The tenant migration deliberately has no `USE` statement: MySQL applies it to
the database named by `MYSQL_DATABASE`, so the same file works for every tenant.

### Database users

All passwords below are throw-away local Docker credentials. Real credentials
belong in `.env` files, which are git-ignored and never logged.

| User                | Access                         |
| ------------------- | ------------------------------ |
| `pizza_palace_user` | `tenant_pizza_palace` only     |
| `sushi_house_user`  | `tenant_sushi_house` only      |
| `master_reader`     | `SELECT` on `mini_master` only |

Each tenant user exists only inside its own container, so even a leaked
credential cannot reach another tenant.

Tenant credentials are loaded from `.env` by `docker-compose.yaml`. The master
registry seed (`seeds/master_tenants.sql`) is plain SQL and cannot read `.env`,
so if you change a tenant credential or port, update that file too and run
`make db-reset`.

### Manual verification

```bash
# Registry contents
docker compose exec -T mysql-master \
  sh -c 'MYSQL_PWD="$MYSQL_ROOT_PASSWORD" mysql -uroot -e "SELECT x_tenant_code, host, port, database_name FROM mini_master.tenants"'

# Pizza tenant seed data as its own user
docker compose exec -T mysql-tenant-pizza \
  sh -c 'MYSQL_PWD=pizza_local_pw mysql -upizza_palace_user -D tenant_pizza_palace -e "SELECT COUNT(*) AS orders FROM orders"'

# Sushi tenant seed data as its own user
docker compose exec -T mysql-tenant-sushi \
  sh -c 'MYSQL_PWD=sushi_local_pw mysql -usushi_house_user -D tenant_sushi_house -e "SELECT COUNT(*) AS orders FROM orders"'

# The master user is read-only
docker compose exec -T mysql-master \
  sh -c 'MYSQL_PWD=master_local_pw mysql -umaster_reader -e "UPDATE mini_master.tenants SET site_name = \"x\""'
```
