
CREATE DATABASE IF NOT EXISTS mini_master
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

USE mini_master;

CREATE TABLE tenants (
    id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    x_tenant_code VARCHAR(64) NOT NULL,
    site_name VARCHAR(255) NOT NULL,
    host VARCHAR(255) NOT NULL,
    port VARCHAR(8) NOT NULL DEFAULT '3306',
    database_name VARCHAR(64) NOT NULL,
    username VARCHAR(64) NOT NULL,
    password VARCHAR(255) NOT NULL,
    currency CHAR(3) NOT NULL DEFAULT 'GBP',
    is_active TINYINT(1) NOT NULL DEFAULT 1,
    created_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NULL DEFAULT CURRENT_TIMESTAMP
        ON UPDATE CURRENT_TIMESTAMP,

    PRIMARY KEY (id),
    UNIQUE KEY uq_tenants_x_tenant_code (x_tenant_code),
    UNIQUE KEY uq_tenants_database_name (database_name)
) ENGINE=InnoDB;

INSERT INTO tenants (
    x_tenant_code,
    site_name,
    host,
    port,
    database_name,
    username,
    password,
    currency
) VALUES
    ('pizza_palace', 'Pizza Palace', 'localhost', '3306',
     'tenant_pizza_palace', 'pizza_palace_user', 'pizza_local_pw', 'GBP'),

    ('sushi_house', 'Sushi House', 'localhost', '3306',
     'tenant_sushi_house', 'sushi_house_user', 'sushi_local_pw', 'EUR'),

    ('burger_barn', 'Burger Barn', 'localhost', '3306',
     'tenant_burger_barn', 'burger_barn_user', 'burger_local_pw', 'GBP');
