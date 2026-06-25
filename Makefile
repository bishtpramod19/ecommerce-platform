.PHONY: help dev-up dev-down dev-logs dev-clean

# Variables
SERVICES := user-service product-service inventory-service order-service payment-service notification-service delivery-service analytics-service

help: ## Show available commands
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

dev-up: ## Start infrastructure (Postgres, MongoDB, Redis, Kafka)
	docker-compose up -d

dev-down: ## Stop infrastructure
	docker-compose down

dev-logs: ## View infrastructure logs
	docker-compose logs -f

dev-clean: ## Stop and remove volumes (clean slate)
	docker-compose down -v

.DEFAULT_GOAL := help
