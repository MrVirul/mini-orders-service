-- Master database: the tenants registry.
--
-- Applied to mini_master by the MySQL entrypoint (MYSQL_DATABASE=mini_master).
-- Safe to re-run.

CREATE TABLE IF NOT EXISTS tenants (
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
