SHELL := /bin/bash

.PHONY: db-up db-down db-logs db-setup db-verify db-reset

db-up: ## Start the local MySQL container
	docker compose up -d

db-down: ## Stop the local MySQL container (keeps data)
	docker compose down

db-logs: ## Follow MySQL logs
	docker compose logs -f mysql

db-setup: ## Apply all migrations, seeds and database users
	./scripts/db-setup.sh

db-verify: ## Check schemas, seed data and tenant isolation
	./scripts/db-verify.sh

db-reset: ## Destroy the MySQL volume and rebuild everything
	docker compose down -v
	docker compose up -d
	./scripts/db-setup.sh
