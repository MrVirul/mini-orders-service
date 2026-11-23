-- Creates the four databases that make up the local environment.
-- Run against the MySQL server (no database selected).
-- Safe to re-run.

CREATE DATABASE IF NOT EXISTS mini_master
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

CREATE DATABASE IF NOT EXISTS tenant_pizza_palace
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

CREATE DATABASE IF NOT EXISTS tenant_sushi_house
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;

CREATE DATABASE IF NOT EXISTS tenant_burger_barn
    CHARACTER SET utf8mb4
    COLLATE utf8mb4_unicode_ci;
