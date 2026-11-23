# Assignment 1 — Build a Multi-Tenant Order Service in Go

**Role:** Backend Intern (Golang) · **Duration:** about 2 weeks · **Reviewer:** your mentor

Welcome to Delivergate. By the end of this assignment you will have built, from scratch, a small
service that works the same way our real services work: **one codebase serving many restaurants
(tenants), each with its own private database**. You will also have used GitHub the way we do — feature
branches, three long-lived branches, and pull requests — to build it one feature at a time.

The business domain is deliberately tiny (menu items and orders). The point is not the orders. The
point is that you understand **how a request finds the right tenant's database, and how we make sure
it can never touch the wrong one.**

---

## 1. What you will learn

1. What multi-tenancy is, and why we use **database-per-tenant** instead of a `tenant_id` column.
2. How a **master database** records which tenants exist and where their databases live.
3. How the **`X-Tenant-Code`** header selects a tenant on every request, and how a Go middleware turns that
   header into a database connection.
4. How to write Go HTTP handlers, repositories and tests that stay tenant-safe.
5. How to work in Git: **feature branches → pull requests → `dev` → `qa` → `prod`**.

---

## 2. Multi-tenancy in five minutes

A **tenant** is one customer of the platform — a restaurant or restaurant chain. Delivergate serves
many tenants from the *same running service*. The risk in any such system is a **data leak**: tenant A
seeing tenant B's orders. Our design makes that structurally hard:

```
                         ┌──────────────────────────────┐
                         │          MASTER DB           │
                         │  tenants                     │
                         │   x_tenant_code → host, port,│
                         │   database_name, user, pass  │
                         └──────────────▲───────────────┘
                                        │ 2. look up the tenant
 Client ── GET /orders ──►  Your Go service
        X-Tenant-Code: pizza_palace     │ 1. read header
                                        │ 3. pick that tenant's connection pool
                  ┌─────────────────────┼─────────────────────┐
                  ▼                     ▼                     ▼
        ┌───────────────────┐ ┌───────────────────┐ ┌───────────────────┐
        │ tenant_pizza_palace│ │ tenant_sushi_house │ │ tenant_burger_barn │
        │ menu_items         │ │ menu_items         │ │ menu_items         │
        │ orders             │ │ orders             │ │ orders             │
        │ order_items        │ │ order_items        │ │ order_items        │
        └───────────────────┘ └───────────────────┘ └───────────────────┘
              4. the query runs ONLY on that tenant's database
```

Key facts to remember:

- **One master database.** It contains the `tenants` table — the registry of every tenant. It holds
  *no business data*.
- **One database per tenant.** Every tenant database has the **identical schema** (same tables, same
  columns). Only the data differs.
- **Tenant resolution happens once, in middleware**, before any handler runs. Handlers and repositories
  receive an already-selected connection. **They never choose a database themselves.**
- Because every tenant DB has its own tables, **auto-increment IDs overlap**: order `1` exists in
  *every* tenant. That is intentional — it makes isolation bugs visible instead of hidden.
- Adding a new tenant means **adding a row to `tenants` and creating a database** — *no code change,
  no redeploy*.

---

## 3. The system you are building

**`mini-orders-service`** — a Go HTTP API.

| Concern | Decision |
|---|---|
| Language | Go (latest stable) |
| HTTP framework | [Fiber v2](https://docs.gofiber.io/) (our POS Go service uses it) |
| Database | MySQL 8, via `database/sql` + `github.com/go-sql-driver/mysql` (no ORM) |
| Local environment | Docker Compose (MySQL) |
| Config | Environment variables |
| Tests | Go `testing`, table-driven; integration tests against the Docker MySQL |

### 3.1 The shape of an order

An order belongs to exactly one tenant, has one or more lines, and money is **always an integer in
minor units** (pence/cents) — never a float.

**Create-order request** — `POST /orders` (note: the client sends *no prices*; the server looks them up):

```json
{
  "customer_name": "Asha Perera",
  "customer_phone": "+447700900123",
  "order_type": "TAKEAWAY",
  "notes": "No onions please",
  "items": [
    { "menu_item_id": 1, "quantity": 2 },
    { "menu_item_id": 3, "quantity": 1 }
  ]
}
```

**Order response** — `201 Created` / `GET /orders/{id}`:

```json
{
  "id": 2,
  "order_number": "ORD-0002",
  "status": "PENDING",
  "order_type": "TAKEAWAY",
  "customer_name": "Asha Perera",
  "notes": "No onions please",
  "currency": "GBP",
  "items": [
    { "menu_item_id": 1, "name": "Margherita", "unit_price_minor": 899, "quantity": 2, "line_total_minor": 1798 },
    { "menu_item_id": 3, "name": "Garlic Bread", "unit_price_minor": 399, "quantity": 1, "line_total_minor": 399 }
  ],
  "total_minor": 2197,
  "created_at": "2026-11-23T12:30:00Z"
}
```

Notes on the shape:

- `currency` is **tenant configuration** (stored in the master DB), not stored on each order.
- `name` and `unit_price_minor` on a line are a **snapshot** taken at order time — if the menu price
  changes tomorrow, yesterday's order must not change.
- `customer_phone` is **personal data (PII)**. It is stored, but it is **never returned in list
  responses and never written to logs.** Treat any customer detail the same way.

**Order statuses and allowed transitions:**

```
PENDING ──► CONFIRMED ──► PREPARING ──► READY ──► COMPLETED
   │            │
   └────────────┴──────► CANCELLED
```

Any other transition is rejected with `409 Conflict`.

---

## 4. Database schemas

You will create **four databases** on one local MySQL server: one master and three tenants.

| Database | Purpose |
|---|---|
| `mini_master` | Master DB — the `tenants` registry |
| `tenant_pizza_palace` | Tenant 1 |
| `tenant_sushi_house` | Tenant 2 |
| `tenant_burger_barn` | Tenant 3 |

### 4.1 Master database — `mini_master`

This mirrors our real master DB: a `tenants` table where each row tells the service *where a tenant's
database lives*, keyed by `x_tenant_code`.

```sql
CREATE DATABASE IF NOT EXISTS mini_master
  CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
USE mini_master;

CREATE TABLE tenants (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  x_tenant_code VARCHAR(64)     NOT NULL,            -- the value clients send in X-Tenant-Code
  site_name     VARCHAR(255)    NOT NULL,            -- display name of the restaurant
  host          VARCHAR(255)    NOT NULL,            -- where the tenant DB lives
  port          VARCHAR(8)      NOT NULL DEFAULT '3306',
  database_name VARCHAR(64)     NOT NULL,
  username      VARCHAR(64)     NOT NULL,
  password      VARCHAR(255)    NOT NULL,
  currency      CHAR(3)         NOT NULL DEFAULT 'GBP',
  is_active     TINYINT(1)      NOT NULL DEFAULT 1,  -- 0 = suspended tenant, must be rejected
  created_at    TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at    TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_tenants_x_tenant_code (x_tenant_code),
  UNIQUE KEY uq_tenants_database_name (database_name)
) ENGINE=InnoDB;

INSERT INTO tenants (x_tenant_code, site_name, host, port, database_name, username, password, currency) VALUES
  ('pizza_palace', 'Pizza Palace', 'localhost', '3306', 'tenant_pizza_palace', 'pizza_palace_user', 'pizza_local_pw', 'GBP'),
  ('sushi_house',  'Sushi House',  'localhost', '3306', 'tenant_sushi_house',  'sushi_house_user',  'sushi_local_pw',  'EUR'),
  ('burger_barn',  'Burger Barn',  'localhost', '3306', 'tenant_burger_barn',  'burger_barn_user',  'burger_local_pw', 'GBP');
```

> The passwords above are throw-away **local Docker** credentials, so plain text is fine *for this
> exercise*. Real credentials are never committed to Git and never logged. Yours go in `.env` files that
> are listed in `.gitignore`.

### 4.2 Tenant schema — identical in all three tenant databases

Write this once as a migration file and apply it to each tenant database. **If your three tenant
databases ever have different schemas, you have made a mistake.**

```sql
-- migrations/tenant/001_init.sql  (run against EACH tenant database)

CREATE TABLE menu_items (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  name         VARCHAR(120)    NOT NULL,
  description  VARCHAR(500)    NULL,
  price_minor  INT UNSIGNED    NOT NULL,             -- pence/cents, never a float
  is_available TINYINT(1)      NOT NULL DEFAULT 1,
  created_at   TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE orders (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_number   VARCHAR(32)     NOT NULL,           -- e.g. ORD-0002, unique within the tenant
  customer_name  VARCHAR(120)    NOT NULL,
  customer_phone VARCHAR(32)     NULL,               -- PII: never log, never list
  order_type     ENUM('DINE_IN','TAKEAWAY','DELIVERY') NOT NULL,
  status         ENUM('PENDING','CONFIRMED','PREPARING','READY','COMPLETED','CANCELLED')
                 NOT NULL DEFAULT 'PENDING',
  total_minor    INT UNSIGNED    NOT NULL,
  notes          VARCHAR(500)    NULL,
  created_at     TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at     TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  PRIMARY KEY (id),
  UNIQUE KEY uq_orders_order_number (order_number),
  KEY idx_orders_status_created (status, created_at)
) ENGINE=InnoDB;

CREATE TABLE order_items (
  id               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  order_id         BIGINT UNSIGNED NOT NULL,
  menu_item_id     BIGINT UNSIGNED NOT NULL,
  item_name        VARCHAR(120)    NOT NULL,         -- snapshot at order time
  unit_price_minor INT UNSIGNED    NOT NULL,         -- snapshot at order time
  quantity         SMALLINT UNSIGNED NOT NULL,
  line_total_minor INT UNSIGNED    NOT NULL,
  PRIMARY KEY (id),
  KEY idx_order_items_order_id (order_id),
  CONSTRAINT fk_order_items_order     FOREIGN KEY (order_id)     REFERENCES orders (id) ON DELETE CASCADE,
  CONSTRAINT fk_order_items_menu_item FOREIGN KEY (menu_item_id) REFERENCES menu_items (id),
  CONSTRAINT chk_order_items_quantity CHECK (quantity > 0)
) ENGINE=InnoDB;
```

### 4.3 Seed data — different per tenant, on purpose

Every tenant gets an order with **`id = 1`** but a *different* menu and order. If your service ever
returns the wrong restaurant's order `1`, you will see it immediately.

```sql
-- seeds/tenant_pizza_palace.sql   (run against tenant_pizza_palace)
INSERT INTO menu_items (id, name, description, price_minor) VALUES
  (1, 'Margherita',   'Tomato, mozzarella, basil', 899),
  (2, 'Pepperoni',    'Pepperoni, mozzarella',     1049),
  (3, 'Garlic Bread', NULL,                        399);
INSERT INTO orders (id, order_number, customer_name, order_type, status, total_minor) VALUES
  (1, 'ORD-0001', 'Seed Customer', 'TAKEAWAY', 'CONFIRMED', 1798);
INSERT INTO order_items (order_id, menu_item_id, item_name, unit_price_minor, quantity, line_total_minor) VALUES
  (1, 1, 'Margherita', 899, 2, 1798);

-- seeds/tenant_sushi_house.sql    (run against tenant_sushi_house)
INSERT INTO menu_items (id, name, description, price_minor) VALUES
  (1, 'Salmon Nigiri',  '2 pieces',          650),
  (2, 'California Roll','8 pieces',          780),
  (3, 'Miso Soup',      NULL,                350);
INSERT INTO orders (id, order_number, customer_name, order_type, status, total_minor) VALUES
  (1, 'ORD-0001', 'Seed Customer', 'DINE_IN', 'PREPARING', 1000);
INSERT INTO order_items (order_id, menu_item_id, item_name, unit_price_minor, quantity, line_total_minor) VALUES
  (1, 1, 'Salmon Nigiri', 650, 1, 650),
  (1, 3, 'Miso Soup',     350, 1, 350);

-- seeds/tenant_burger_barn.sql    (run against tenant_burger_barn)
INSERT INTO menu_items (id, name, description, price_minor) VALUES
  (1, 'Classic Burger',   'Beef, lettuce, tomato', 950),
  (2, 'Cheeseburger',     'Beef, cheddar',         1050),
  (3, 'Fries',            NULL,                    350);
INSERT INTO orders (id, order_number, customer_name, order_type, status, total_minor) VALUES
  (1, 'ORD-0001', 'Seed Customer', 'DELIVERY', 'PENDING', 1300);
INSERT INTO order_items (order_id, menu_item_id, item_name, unit_price_minor, quantity, line_total_minor) VALUES
  (1, 1, 'Classic Burger', 950, 1, 950),
  (1, 3, 'Fries',          350, 1, 350);
```

### 4.4 Database users (isolation at the database level too)

Give each tenant its own MySQL user that can reach **only its own database**. Even a bug in your Go
code then cannot read another tenant's tables through that connection.

```sql
CREATE USER 'pizza_palace_user'@'%' IDENTIFIED BY 'pizza_local_pw';
CREATE USER 'sushi_house_user'@'%'  IDENTIFIED BY 'sushi_local_pw';
CREATE USER 'burger_barn_user'@'%'  IDENTIFIED BY 'burger_local_pw';

GRANT ALL PRIVILEGES ON tenant_pizza_palace.* TO 'pizza_palace_user'@'%';
GRANT ALL PRIVILEGES ON tenant_sushi_house.*  TO 'sushi_house_user'@'%';
GRANT ALL PRIVILEGES ON tenant_burger_barn.*  TO 'burger_barn_user'@'%';

-- the service's master connection: read-only on the registry
CREATE USER 'master_reader'@'%' IDENTIFIED BY 'master_local_pw';
GRANT SELECT ON mini_master.* TO 'master_reader'@'%';
FLUSH PRIVILEGES;
```

---

## 5. The `X-Tenant-Code` contract

This is the single most important rule of the assignment.

> **Every request that touches tenant data must carry an `X-Tenant-Code` header. The service looks the
> code up in the master DB `tenants` table and uses *that tenant's* database — and nothing else.**

| Situation | Response |
|---|---|
| Header missing or empty | `400 Bad Request` — `{"error":{"code":"tenant_header_missing","message":"Missing X-Tenant-Code header"}}` |
| Code not in `tenants`, or `is_active = 0` | `401 Unauthorized` — `{"error":{"code":"tenant_unknown","message":"Unknown tenant"}}` |
| Code resolves, tenant DB unreachable | `503 Service Unavailable` — and log it (without credentials) |
| Code resolves | request continues with that tenant's connection attached to the request |

Rules that are **not negotiable**:

1. **The tenant comes from the header only** — never from the request body, query string, or URL. A
   client must not be able to say "order 5 of *some other* tenant."
2. **Resolve in middleware, once per request.** Handlers read the resolved tenant from the request
   context. A handler or repository must never open its own database connection or call `USE <db>`.
3. **One connection pool per tenant, created lazily and cached** (e.g. a `map[string]*sql.DB` guarded
   by a mutex). Never open a new pool per request — that exhausts MySQL connections within minutes.
4. **No `tenant_id` column anywhere.** Isolation comes from the separate database, not from remembering
   to add `WHERE tenant_id = ?`.
5. **`GET /health` is the only route that works without the header.**
6. **Never log passwords or customer PII.** Do log `tenant_code` on every request log line.

Go-specific traps (these have bitten real services):

- Fiber's `c.Get("X-Tenant-Code")` returns a string backed by a buffer that Fiber **reuses after the
  handler returns**. If you keep it anywhere that outlives the request (a goroutine, a cache key, a
  log), copy it first — `utils.CopyString(...)` from `github.com/gofiber/fiber/v2/utils`. Our POS
  service does exactly this in its tenant middleware.
- Store the resolved tenant on the request (`c.Locals("tenant", tenant)`) as a small struct —
  `{Code, SiteName, Currency, DB *sql.DB}` — and write one helper, `TenantFrom(c)`, that handlers use.
- Pass `c.Context()` into every `QueryContext` / `ExecContext`, so a cancelled request cancels its query.

### 5.1 API surface

| Method | Path | Needs header | What it does |
|---|---|---|---|
| GET | `/health` | no | Liveness + master DB ping |
| GET | `/tenant` | yes | Returns the resolved tenant (`code`, `site_name`, `currency`) |
| GET | `/menu-items` | yes | List available menu items |
| POST | `/orders` | yes | Create an order |
| GET | `/orders/{id}` | yes | One order with its lines |
| GET | `/orders` | yes | List orders — `?status=`, `?limit=` (default 20, max 100), `?offset=` |
| PATCH | `/orders/{id}/status` | yes | Move an order along the state machine |

Every error uses the envelope `{"error":{"code":"...","message":"..."}}`.

---

## 6. How you will work: Git and GitHub

You will build this service **one feature at a time**. Each feature is its own branch, its own pull
request, and its own review. This is exactly how we ship at Delivergate.

### 6.1 The three long-lived branches

Your repo has three permanent branches. **Nobody pushes directly to any of them — not you, not your
mentor.** Code reaches them only through a reviewed pull request.

| Branch | Meaning | Who merges into it |
|---|---|---|
| `dev` | Integration branch. Everything under development lands here first. | Feature PRs, after review |
| `qa` | Release candidate. What is being tested. | A promotion PR from `dev` |
| `prod` | What is live. Always stable. | A promotion PR from `qa` |

```
 feature branches (short-lived, one per feature)
   project-skeleton/DEV-1/you ─┐
   tenant-middleware/DEV-3/you ┼──PR──►  dev  ──PR──►  qa  ──PR──►  prod
   menu-items/DEV-5/you ───────┘      (integrate)   (test)        (live)
```

Code only ever moves **to the right**. You never merge `prod` back into a feature branch, and you
never fix something directly on `qa` or `prod`.

### 6.2 Feature branches

Every piece of work happens on a branch cut **from the latest `dev`**, named:

```
{feature_name}/{task_id}/{github_username}
```

for example `tenant-middleware/DEV-1203/janedoe`. Your mentor gives you the `task_id` for each
feature. Names are lower-case with hyphens.

### 6.3 The loop you repeat for every feature

```bash
# 1. Start from an up-to-date dev
git checkout dev
git pull origin dev

# 2. Create your feature branch
git checkout -b tenant-middleware/DEV-1203/janedoe

# 3. Work in small commits (see 6.4)
git add -p
git commit -m "feat(tenant): resolve tenant from X-Tenant-Code header"

# 4. Before opening the PR, bring your branch up to date with dev
git fetch origin
git rebase origin/dev        # resolve conflicts if any, then: git rebase --continue

# 5. Run the checks locally — they must pass
gofmt -l .                   # must print nothing
go vet ./...
go test ./...

# 6. Push your branch (first push sets upstream)
git push -u origin tenant-middleware/DEV-1203/janedoe

# 7. Open a pull request into dev (see 6.5), request review, respond to comments
#    with new commits on the same branch — the PR updates automatically.

# 8. After it is merged: clean up and start the next feature from step 1
git checkout dev && git pull origin dev
git branch -d tenant-middleware/DEV-1203/janedoe
```

Hard rules:

- **Never** `git push` to `dev`, `qa` or `prod`.
- **Never** force-push to a shared branch. Force-pushing your own *feature* branch after a rebase is
  fine (`git push --force-with-lease`), because only you are using it.
- **One feature = one branch = one PR.** Do not stack unrelated changes in one PR.
- Keep PRs small — under roughly 400 changed lines. If it is bigger, the feature should have been split.

### 6.4 Commit messages

Use a short imperative subject, with a type prefix:

```
feat(orders): create order with server-side price lookup
fix(tenant): copy header value before caching the pool
test(isolation): prove tenant A cannot read tenant B's order
docs: explain how to add a fourth tenant
chore: add .env.example
```

A commit should be one logical change that compiles. "WIP", "fix", "asdf" are not acceptable messages.

### 6.5 Pull requests

Open the PR **into `dev`**, titled like your commit (`feat(tenant): ...`). Use this description template:

```markdown
## What
One or two sentences: what does this PR add?

## Why
Which feature in the assignment, and the task id (DEV-xxxx).

## How to test
Exact curl commands, including the X-Tenant-Code header, and the expected result.

## Tenant-safety checklist
- [ ] No handler or repository opens its own DB connection
- [ ] The tenant comes only from the X-Tenant-Code header
- [ ] No customer PII or credentials are logged
- [ ] I tested with at least two different tenants

## Checks
- [ ] gofmt / go vet / go test all pass
- [ ] No secrets or .env files committed
```

**Review etiquette**

- Your mentor reviews every PR. Expect comments — that is the process, not a verdict on you.
- Reply to every comment (fix it, or explain why not). Push fixes as new commits.
- Never merge your own PR. Merge happens only after approval.
- Resolve conflicts by **rebasing on `dev`**, not by merging `dev` into your branch.

### 6.6 Promoting to `qa` and `prod`

When several features are in `dev` and it works, you open a **promotion PR**: base `qa`, compare `dev`,
titled `release: dev → qa (YYYY-MM-DD)`, listing the features included. Your mentor reviews and merges
it. Once `qa` has been tested, repeat with base `prod`, compare `qa`. You will do this for real in
Feature 9.

---

## 7. The assignment, feature by feature

Do the features **in order**. Each one is one branch and one PR. Do not start the next feature until
the previous PR is merged into `dev`.

### Feature 0 — Repository set-up *(branch: `project-skeleton`)*

*Your mentor first creates the empty GitHub repo `mini-orders-service` with the `dev`, `qa` and `prod`
branches and branch protection (no direct pushes, one approving review required). You clone it.*

You deliver:
- `go mod init`, a sensible folder layout (`cmd/api`, `internal/...`, `migrations/`, `seeds/`)
- A Fiber app with `GET /health` returning `{"status":"ok"}`
- `README.md` (how to run it), `.gitignore` (includes `.env`), `.env.example`
- Graceful shutdown on `SIGINT`/`SIGTERM`

**Done when:** `go run ./cmd/api` starts and `curl localhost:3000/health` returns 200.
**Git focus:** your first branch, commit and PR. Expect your mentor's first review comments here.

### Feature 1 — Databases *(branch: `database-setup`)*

You deliver:
- `docker-compose.yml` running MySQL 8
- SQL files for sections 4.1–4.4, organised under `migrations/master`, `migrations/tenant` and `seeds/`
- A script (`make db-reset` or `scripts/db-reset.sh`) that drops and recreates all four databases,
  applies the schemas and loads the seeds — runnable repeatedly with the same result

**Done when:** after `make db-reset`, `SELECT COUNT(*) FROM menu_items` returns 3 in each tenant DB and
the master DB lists exactly three tenants.

### Feature 2 — Tenant registry *(branch: `tenant-registry`)*

You deliver:
- Config loading (env vars) and a master DB connection (`master_reader`)
- A `Tenant` struct and a repository function `FindByCode(ctx, code) (*Tenant, error)` that reads the
  `tenants` table and rejects `is_active = 0`
- Update `/health` to also ping the master DB
- Unit/integration tests: known code, unknown code, inactive tenant

**Done when:** tests prove the three lookups behave correctly. No HTTP tenant logic yet.

### Feature 3 — Tenant middleware *(branch: `tenant-middleware`)*

The heart of the assignment.

You deliver:
- Middleware implementing the full table in section 5 (400 / 401 / 503 / continue)
- A cached per-tenant connection pool (`map[string]*sql.DB` + `sync.RWMutex`)
- The `Tenant` runtime struct on the request and a `TenantFrom(c)` helper
- `GET /tenant` returning the resolved tenant
- Request logging that includes `tenant_code`

**Done when:**
```bash
curl -i localhost:3000/tenant                                   # 400
curl -i -H "X-Tenant-Code: nope" localhost:3000/tenant          # 401
curl -i -H "X-Tenant-Code: pizza_palace" localhost:3000/tenant  # 200, GBP
curl -i -H "X-Tenant-Code: sushi_house"  localhost:3000/tenant  # 200, EUR
```
and 100 concurrent requests for the same tenant create **exactly one** pool (prove it in a test).

### Feature 4 — Menu items *(branch: `menu-items`)*

You deliver `GET /menu-items`, with handler → service → repository layers. The repository receives the
tenant's `*sql.DB` as a dependency — it does not find a database itself.

**Done when:** `pizza_palace` returns Margherita/Pepperoni/Garlic Bread, `sushi_house` returns the sushi
menu, and an unavailable item (set `is_available = 0` manually) is hidden.

### Feature 5 — Create order *(branch: `create-order`)*

You deliver `POST /orders`:
- Validate the body (non-empty name, valid `order_type`, ≥ 1 item, quantity ≥ 1, sensible max)
- Look prices up **server-side** from `menu_items`; reject unknown or unavailable items with `422`
- Insert `orders` + `order_items` **in one transaction** — all or nothing
- Generate `order_number` (`ORD-` + zero-padded id) and return the full order shape from section 3.1

**Done when:** an order created for `pizza_palace` gets `id 2`, an order created for `sushi_house`
*also* gets `id 2`, and neither shows up in the other's database. A request that sends a client-supplied
price ignores it.

### Feature 6 — Read orders *(branch: `get-orders`)*

You deliver `GET /orders/{id}` and `GET /orders` with `status`, `limit` and `offset`.
List responses omit `customer_phone`. `GET /orders/1` must return a *different* order for each tenant.
An id that does not exist in **this** tenant returns `404` — even if it exists in another tenant.

### Feature 7 — Order status *(branch: `order-status`)*

You deliver `PATCH /orders/{id}/status` enforcing the state machine in section 3.1. Illegal transitions
return `409`. Make the update safe against two concurrent requests (hint: `UPDATE ... WHERE id = ? AND
status = ?` and check rows affected).

### Feature 8 — Isolation tests *(branch: `isolation-tests`)*

A feature whose whole product is **proof**. Write integration tests that start the app against the
Docker MySQL and assert:

1. Order created under tenant A is `404` when fetched with tenant B's code, even using the same id.
2. `GET /orders` for tenant A never contains an order created for tenant B.
3. A missing, unknown or inactive tenant code is rejected on **every** tenant-scoped route (table-driven
   across all routes).
4. Two tenants creating orders concurrently never cross-contaminate.

If a test cannot fail, it proves nothing — confirm each test fails when you deliberately break the
isolation (for example, hardcode one tenant's pool) and write that in the PR description.

### Feature 9 — Release practice *(branch: `docs-and-release`)*

You deliver:
- `docs/ADD_A_TENANT.md` — and **prove it works**: add a fourth tenant `taco_town` using only SQL
  (a database, a user, a `tenants` row) and the existing running binary. **Zero Go changes.**
- `docs/ANSWERS.md` — written answers to the questions in section 9
- Then the promotion flow: open `release: dev → qa`, and after your mentor merges it,
  `release: qa → prod`

**Done when:** `prod` contains everything, the fourth tenant works against the same binary, and your
promotion PRs are merged.

### Stretch goals *(optional, after Feature 9)*

- Structured JSON logging with `request_id` and `tenant_code` on every line
- A GitHub Actions workflow running `gofmt`, `go vet` and `go test` on every PR into `dev`
- Replace the pool-cache mutex with `singleflight` so concurrent first requests for a tenant coalesce
- Evict and close the pool of a tenant that was deactivated

---

## 8. Definition of done (for *every* PR)

- [ ] Branch named `{feature_name}/{task_id}/{github_username}`, cut from latest `dev`
- [ ] Rebased on `dev`, no merge commits from `dev`
- [ ] `gofmt -l .` prints nothing; `go vet ./...` and `go test ./...` pass
- [ ] New behaviour has tests, including at least one with **two different tenants**
- [ ] No secrets, `.env` files, customer PII or passwords in code or logs
- [ ] PR description filled in using the template
- [ ] Reviewed and approved by your mentor — merged by your mentor

## 9. Questions to answer in `docs/ANSWERS.md`

Short, in your own words (3–6 sentences each):

1. Why does Delivergate use a database per tenant instead of one database with a `tenant_id` column?
   Give one advantage and one cost.
2. What would happen if the tenant middleware were skipped on one route? Walk through exactly what a
   request would do.
3. Why must the tenant code come from the header and never from the request body?
4. Why do we cache one connection pool per tenant instead of opening a connection per request?
5. Order `id = 1` exists in all three tenants. Why is that a *good* thing during development?
6. A new restaurant signs up. List every step needed to onboard them in your service — and which of
   those steps need a code deployment.
7. Why are prices looked up on the server rather than accepted from the client?
8. You are halfway through a feature when someone merges to `dev`. What do you do, and why rebase
   rather than merge?
9. Why is nobody — not even your mentor — allowed to push directly to `qa` or `prod`?

## 10. How you will be assessed

| Area | What we look for | Weight |
|---|---|---|
| Tenant isolation | Header-only resolution, middleware-only DB selection, pool cache, isolation tests that can fail | 30% |
| Correctness | All endpoints and state machine behave as specified; transactions used correctly | 20% |
| Git & PR discipline | Branch naming, small PRs, clean commits, rebased history, no direct pushes, responding to review | 20% |
| Code quality | Layering, error handling, `context` use, no globals for DB, readable Go | 15% |
| Testing | Table-driven tests, integration tests, two-tenant coverage | 10% |
| Written answers | Evidence you understand *why*, not just *how* | 5% |

**Ask questions early.** A question asked on day two costs nothing; a misunderstanding discovered in
Feature 8 costs a rewrite. Your mentor would rather answer ten questions than review one leaky
tenant lookup.

Good luck — and welcome to the team.
