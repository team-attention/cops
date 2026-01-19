ENV ?= local

include .meta/.env
-include .meta/.env.$(ENV)
export

## migration-up: Run all pending migrations
.PHONY: migration-up
migration-up:
	npm run migrate:up

## migration-down: Rollback the last applied migration
.PHONY: migration-down
migration-down:
	npm run migrate:down
