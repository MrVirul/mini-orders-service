-- Tenant schema: identical in ALL three tenant databases.
-- Run once per tenant database, e.g.:
--   mysql -u root -p tenant_pizza_palace < migrations/tenant/001_init.sql
-- There is deliberately no USE statement here: the target database is chosen
-- by the caller, so the exact same file is applied to every tenant.
-- Safe to re-run.

CREATE TABLE IF NOT EXISTS menu_items (
    id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    name         VARCHAR(120)    NOT NULL,
    description  VARCHAR(500)    NULL,
    price_minor  INT UNSIGNED    NOT NULL,             -- pence/cents, never a float
    is_available TINYINT(1)      NOT NULL DEFAULT 1,
    created_at   TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMP       NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    PRIMARY KEY (id)
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS orders (
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

CREATE TABLE IF NOT EXISTS order_items (
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
