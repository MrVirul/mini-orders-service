-- Local-only database users.
-- Each tenant user can reach only its own database, and the master connection
-- is read-only on the tenant registry.
-- Safe to re-run.

CREATE USER IF NOT EXISTS 'pizza_palace_user'@'%' IDENTIFIED BY 'pizza_local_pw';
CREATE USER IF NOT EXISTS 'sushi_house_user'@'%'  IDENTIFIED BY 'sushi_local_pw';
CREATE USER IF NOT EXISTS 'burger_barn_user'@'%'  IDENTIFIED BY 'burger_local_pw';
CREATE USER IF NOT EXISTS 'master_reader'@'%'     IDENTIFIED BY 'master_local_pw';

-- Keep passwords in sync with this file when it is re-applied.
ALTER USER 'pizza_palace_user'@'%' IDENTIFIED BY 'pizza_local_pw';
ALTER USER 'sushi_house_user'@'%'  IDENTIFIED BY 'sushi_local_pw';
ALTER USER 'burger_barn_user'@'%'  IDENTIFIED BY 'burger_local_pw';
ALTER USER 'master_reader'@'%'     IDENTIFIED BY 'master_local_pw';

GRANT ALL PRIVILEGES ON tenant_pizza_palace.* TO 'pizza_palace_user'@'%';
GRANT ALL PRIVILEGES ON tenant_sushi_house.*  TO 'sushi_house_user'@'%';
GRANT ALL PRIVILEGES ON tenant_burger_barn.*  TO 'burger_barn_user'@'%';

-- The service's master connection: read-only on the registry.
GRANT SELECT ON mini_master.* TO 'master_reader'@'%';

FLUSH PRIVILEGES;
