# assignment-01

Multi-tenant order service (Go intern assignment).

## Local database

The local environment is one MySQL 8.0 server with four databases:

| Database               | Purpose                                   |
| ---------------------- | ----------------------------------------- |
| `mini_master`          | Master DB — the tenants registry          |
| `tenant_pizza_palace`  | Tenant 1                                  |
| `tenant_sushi_house`   | Tenant 2                                  |
| `tenant_burger_barn`   | Tenant 3                                  |

Each tenant database has the exact same schema (`menu_items`, `orders`,
`order_items`) but its own seed data, so cross-tenant leaks are easy to spot.

### Setup

```bash
make db-up      # start MySQL in Docker
make db-setup   # apply migrations, seeds, database users and grants
make db-verify  # check schemas, seed rows and tenant isolation
```

`make db-setup` and `make db-verify` are safe to re-run. To start from a clean
volume, use `make db-reset`.

### Layout

```
migrations/setup/001_databases.sql   # creates the four databases
migrations/setup/002_users.sql       # tenant users + read-only master user
migrations/master/001_init.sql       # mini_master.tenants + registry seed
migrations/tenant/001_init.sql       # identical schema, applied to every tenant DB
seeds/tenant_<code>.sql              # per-tenant seed data
scripts/db-setup.sh                  # applies everything above
scripts/db-verify.sh                 # checks data and user isolation
```

The tenant migration deliberately has no `USE` statement: the setup script
selects the target database for each tenant, so all three schemas come from the
same file.

### Database users

All passwords below are throw-away local Docker credentials. Real credentials
belong in `.env` files, which are git-ignored and never logged.

| User                  | Access                                   |
| --------------------- | ---------------------------------------- |
| `pizza_palace_user`   | `tenant_pizza_palace` only               |
| `sushi_house_user`    | `tenant_sushi_house` only                |
| `burger_barn_user`    | `tenant_burger_barn` only                |
| `master_reader`       | `SELECT` on `mini_master` only           |
