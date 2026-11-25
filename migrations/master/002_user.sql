-- Read-only user for the service's master connection.
-- Applied to mini_master by the MySQL entrypoint.
-- Throw-away local Docker credential only.

CREATE USER IF NOT EXISTS 'master_reader'@'%' IDENTIFIED BY 'master_local_pw';
GRANT SELECT ON mini_master.* TO 'master_reader'@'%';
FLUSH PRIVILEGES;
