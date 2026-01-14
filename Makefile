ENV ?= prod
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

.PHONY: deploy-all deploy-web deploy-api release release-snapshot tag _check-version

## deploy-all: Deploy all components with the same version
deploy-all: deploy-web deploy-api release
	@echo "All deployments complete for version $(VERSION)"

## deploy-web: Deploy web to GCS + CDN
deploy-web:
	@echo "Deploying Web $(VERSION)..."
	$(MAKE) -C web deploy ENV=$(ENV) VERSION=$(VERSION)

## deploy-api: Deploy API to Cloud Run
deploy-api:
	@echo "Deploying API $(VERSION)..."
	$(MAKE) -C api deploy ENV=$(ENV) VERSION=$(VERSION)

## release: Create git tag and release CLI/Daemon via GoReleaser
release: _check-version tag
	@echo "Releasing CLI and Daemon $(VERSION)..."
	GITHUB_TOKEN=$(TEAM_ATTENTION_GITHUB_TOKEN) goreleaser release --clean

## release-snapshot: Build release without publishing (for testing)
release-snapshot:
	goreleaser release --snapshot --clean

## tag: Create git tag for release
tag: _check-version
	@if git rev-parse $(VERSION) >/dev/null 2>&1; then \
		echo "Tag $(VERSION) already exists"; \
	else \
		echo "Creating tag $(VERSION)..."; \
		git tag -a $(VERSION) -m "Release $(VERSION)"; \
	fi

## _check-version: Ensure VERSION is a valid semver tag
_check-version:
ifeq ($(VERSION),dev)
	$(error VERSION must be set. Use: make deploy-all VERSION=v1.0.0 ENV=prod)
endif
