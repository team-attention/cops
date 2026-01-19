ENV ?= local

# Environment file configuration
include .meta/.env
-include .meta/.env.$(ENV)
export

## dev-build: Build CLI with debug symbols and race detection
.PHONY: dev-build
dev-build:
	go build -race -gcflags "all=-N -l" -o cops ./cmd/cops

## dev-install: Install CLI to $GOPATH/bin (or $HOME/go/bin)
.PHONY: dev-install
dev-install:
	go install ./cmd/cops

## dev-run: Run CLI command with local environment
.PHONY: dev-run
dev-run:
	COPS_LOG_LEVEL=debug go run ./cmd/cops $(ARGS)
