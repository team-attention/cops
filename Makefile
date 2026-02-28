ENV ?= prod
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")

# Environment file configuration
include .meta/.env
-include .meta/.env.$(ENV)
-include .meta/.env.deploy.$(ENV)
export

# DIST_DIR: GoReleaser snapshot output directory
DIST_DIR := dist

# RELEASE_PREFIX: GCS path prefix for releases
RELEASE_PREFIX := releases

.PHONY: deploy-all deploy-web deploy-api release release-snapshot tag _check-version _check-release-env _upload-release

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

## release: Build via GoReleaser snapshot and upload to GCS
release: _check-version _check-release-env
	@echo "Building release artifacts $(VERSION)..."
	goreleaser release --snapshot --clean
	@echo "Uploading release artifacts to GCS..."
	$(MAKE) _upload-release VERSION=$(VERSION)
	@echo "Release $(VERSION) complete"

## release-snapshot: Build release without uploading (for testing)
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
	$(error VERSION must be set. Use: make release VERSION=v1.0.0)
endif

## _check-release-env: Validate required environment variables for release
_check-release-env:
ifndef GCS_BUCKET
	$(error GCS_BUCKET is not set. Check .meta/.env.$(ENV))
endif

## _upload-release: Upload release artifacts to GCS
## Strips the leading 'v' from VERSION to match GoReleaser's output naming
## (e.g., VERSION=v1.2.3 -> GoReleaser produces cops_1.2.3_*.tar.gz)
## but uploads under gs://bucket/releases/v1.2.3/
_upload-release:
	$(eval SEMVER := $(patsubst v%,%,$(VERSION)))
	@echo "Uploading archives to gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/$(VERSION)/..."
	gsutil -m cp $(DIST_DIR)/cops_$(SEMVER)_*.tar.gz gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/$(VERSION)/
	@echo "Uploading checksums..."
	gsutil cp $(DIST_DIR)/checksums.txt gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/$(VERSION)/checksums.txt
	@echo "Setting cache headers on versioned archives (immutable, long TTL)..."
	gsutil -m setmeta \
		-h "Cache-Control:public, max-age=31536000, immutable" \
		"gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/$(VERSION)/*.tar.gz"
	gsutil setmeta \
		-h "Cache-Control:public, max-age=31536000, immutable" \
		"gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/$(VERSION)/checksums.txt"
	@echo "Creating and uploading latest version file..."
	echo -n "$(VERSION)" > $(DIST_DIR)/latest
	gsutil -h "Cache-Control:no-cache, no-store, must-revalidate" \
		cp $(DIST_DIR)/latest gs://$(GCS_BUCKET)/$(RELEASE_PREFIX)/latest
	@echo "Upload complete: $(VERSION)"
