-- Seed the master tenants registry.
-- Applied to mini_master by the MySQL entrypoint.
-- Keep these host/port/credential values in sync with docker-compose.yaml.
-- Safe to re-run.

INSERT INTO tenants (x_tenant_code, site_name, host, port, database_name, username, password, currency)
VALUES
    ('pizza_palace', 'Pizza Palace', '127.0.0.1', '3307', 'tenant_pizza_palace', 'pizza_palace_user', 'pizza_local_pw', 'GBP'),
    ('sushi_house',  'Sushi House',  '127.0.0.1', '3308', 'tenant_sushi_house',  'sushi_house_user',  'sushi_local_pw',  'EUR')
AS new
ON DUPLICATE KEY UPDATE
    site_name     = new.site_name,
    host          = new.host,
    port          = new.port,
    database_name = new.database_name,
    username      = new.username,
    password      = new.password,
    currency      = new.currency,
    is_active     = 1;
