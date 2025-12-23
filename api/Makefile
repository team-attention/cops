ENV ?= local

# Docker Compose base command
COMPOSE_CMD := docker compose -f .meta/docker-compose.yaml

# Environment file configuration
ENV_FILES := --env-file .meta/.env
ifdef ENV
	ENV_FILES += --env-file .meta/.env.$(ENV)
endif

include .meta/.env
-include .meta/.env.$(ENV)
export

## dev: Start API server with Docker Compose
.PHONY: dev
dev:
	$(COMPOSE_CMD) $(ENV_FILES) up --build

## dev-down: Stop and remove all Docker Compose resources
.PHONY: dev-down
dev-down:
	$(COMPOSE_CMD) down --volumes --remove-orphans
