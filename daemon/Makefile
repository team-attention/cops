ENV ?= local

# Environment file configuration
include .meta/.env
-include .meta/.env.$(ENV)
export

## dev: Start daemon with Air (hot reload)
.PHONY: dev
dev:
	air -c .meta/.air.toml
