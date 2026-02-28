ENV ?= local
VERSION ?= dev

# Environment file configuration
include .meta/.env
-include .meta/.env.$(ENV)
-include .meta/.env.deploy.$(ENV)
export

## build: Build the SPA for production
.PHONY: build deploy
build:
	VITE_APP_VERSION=$(VERSION) npm run build

## deploy: Build and deploy to GCS with CDN cache invalidation
.PHONY: deploy
deploy: _check-env build _upload _invalidate-cdn
	@echo "Deployment complete: $(CDN_URL)"

## _check-env: Validate required environment variables
.PHONY: _check-env
_check-env:
ifndef GCS_BUCKET
	$(error GCS_BUCKET is not set. Check .meta/.env.$(ENV))
endif
ifndef GCP_PROJECT
	$(error GCP_PROJECT is not set. Check .meta/.env.$(ENV))
endif
ifndef CDN_URL_MAP
	$(error CDN_URL_MAP is not set. Check .meta/.env.$(ENV))
endif

## _upload: Upload built files to GCS
.PHONY: _upload
_upload:
	@echo "Uploading to gs://$(GCS_BUCKET)/ (version: $(VERSION))..."
	gsutil -m rsync -r -d -x 'releases/.*' dist/ gs://$(GCS_BUCKET)/
	@echo "Setting cache headers and version metadata..."
	gsutil -m setmeta -h "Cache-Control:public, max-age=31536000" -h "x-goog-meta-version:$(VERSION)" "gs://$(GCS_BUCKET)/assets/**"
	gsutil -m setmeta -h "Cache-Control:no-cache, no-store, must-revalidate" -h "x-goog-meta-version:$(VERSION)" "gs://$(GCS_BUCKET)/index.html"

## _invalidate-cdn: Invalidate Cloud CDN cache
.PHONY: _invalidate-cdn
_invalidate-cdn:
	@echo "Invalidating CDN cache..."
	gcloud compute url-maps invalidate-cdn-cache $(CDN_URL_MAP) \
		--path "/*" \
		--project $(GCP_PROJECT) \
		--async
