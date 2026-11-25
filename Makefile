SHELL := /bin/bash

.PHONY: db-up db-down db-logs db-reset

db-up: ## Start the local MySQL containers
	docker compose up -d

db-down: ## Stop the local MySQL containers (keeps data)
	docker compose down

db-logs: ## Follow logs from all MySQL containers
	docker compose logs -f

db-reset: ## Destroy the MySQL volumes and reprovision from the SQL files
	docker compose down -v
	docker compose up -d
