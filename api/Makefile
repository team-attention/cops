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
-include .meta/.env.deploy.$(ENV)
export

# ===========================================
# Cloud Run Environment Variables
# ===========================================
# Parses .env.{ENV} for gcloud deploy flags
# Uses ;; as delimiter to support values with commas and URLs (://)

RUNTIME_ENV_FILE := .meta/.env.$(ENV)

# Parse SECRET_ prefixed variables for --set-secrets
# Uses ;; as delimiter (gcloud ^;;^ syntax)
SECRETS_VARS := $(shell grep '^SECRET_' $(RUNTIME_ENV_FILE) 2>/dev/null | sed 's/^SECRET_//' | paste -sd ';' - | sed 's/;/;;/g')

# Parse non-SECRET_ variables for --set-env-vars
# Excludes comments and empty lines, uses ;; as delimiter
ENV_VARS := $(shell grep -v '^SECRET_' $(RUNTIME_ENV_FILE) 2>/dev/null | grep -v '^\s*\#' | grep -v '^\s*$$' | paste -sd ';' - | sed 's/;/;;/g')

# Build flags with ^;;^ prefix for alternate delimiter
SECRETS_FLAG := $(if $(SECRETS_VARS),--set-secrets="^;;^$(SECRETS_VARS)",)
ENV_VARS_FLAG := $(if $(ENV_VARS),--set-env-vars="^;;^$(ENV_VARS)",)

# ===========================================
# Development
# ===========================================

## dev: Start API server with Docker Compose
.PHONY: dev
dev:
	$(COMPOSE_CMD) $(ENV_FILES) up --no-attach mongodb

## dev-down: Stop and remove all Docker Compose resources
.PHONY: dev-down
dev-down:
	$(COMPOSE_CMD) down --volumes --remove-orphans

# ===========================================
# Deployment
# ===========================================

# Version from git tag or commit
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
IMAGE_TAG = $(ARTIFACT_REGISTRY)/$(GCP_PROJECT)/$(REGISTRY_REPOSITORY)/$(IMAGE_NAME):$(VERSION)

## build-image: Build Docker image
.PHONY: build-image
build-image: _check-image-env
	@echo "Building Docker image: $(IMAGE_TAG) (version: $(VERSION))"
	docker build --platform linux/amd64 --build-arg VERSION=$(VERSION) -t $(IMAGE_TAG) -f .meta/Dockerfile ..

## push-image: Push Docker image to Artifact Registry
.PHONY: push-image
push-image: _check-image-env
	@echo "Pushing $(IMAGE_TAG)..."
	docker push $(IMAGE_TAG)

## deploy-image: Deploy image to Cloud Run
.PHONY: deploy-image
deploy-image: _check-deploy-env
	@if [ ! -f $(RUNTIME_ENV_FILE) ]; then \
		echo "Error: $(RUNTIME_ENV_FILE) not found."; \
		exit 1; \
	fi
	@echo "Deploying to Cloud Run: $(CLOUD_RUN_SERVICE)"
	@echo "Image: $(IMAGE_TAG)"
	@echo "Runtime env: $(RUNTIME_ENV_FILE)"
	gcloud run deploy $(CLOUD_RUN_SERVICE) \
		--image $(IMAGE_TAG) \
		--region $(CLOUD_RUN_REGION) \
		--project $(GCP_PROJECT) \
		--platform managed \
		--allow-unauthenticated \
		--port 8080 \
		$(SECRETS_FLAG) \
		$(ENV_VARS_FLAG)

## deploy: Build, push, and deploy (all-in-one)
.PHONY: deploy
deploy: build-image push-image deploy-image
	@echo "Deployment complete"

## update-env: Update Cloud Run environment variables without rebuilding
.PHONY: update-env
update-env: _check-update-env
	@if [ ! -f $(RUNTIME_ENV_FILE) ]; then \
		echo "Error: $(RUNTIME_ENV_FILE) not found."; \
		exit 1; \
	fi
	@echo "Updating environment variables for: $(CLOUD_RUN_SERVICE)"
	@echo "Runtime env: $(RUNTIME_ENV_FILE)"
	gcloud run services update $(CLOUD_RUN_SERVICE) \
		--region $(CLOUD_RUN_REGION) \
		--project $(GCP_PROJECT) \
		$(SECRETS_FLAG) \
		$(ENV_VARS_FLAG)

# ===========================================
# Validation
# ===========================================

## _check-image-env: Validate image build/push variables
.PHONY: _check-image-env
_check-image-env:
ifndef ARTIFACT_REGISTRY
	$(error ARTIFACT_REGISTRY is not set)
endif
ifndef GCP_PROJECT
	$(error GCP_PROJECT is not set)
endif
ifndef REGISTRY_REPOSITORY
	$(error REGISTRY_REPOSITORY is not set)
endif
ifndef IMAGE_NAME
	$(error IMAGE_NAME is not set)
endif

## _check-deploy-env: Validate Cloud Run deploy variables
.PHONY: _check-deploy-env
_check-deploy-env: _check-image-env
ifndef CLOUD_RUN_SERVICE
	$(error CLOUD_RUN_SERVICE is not set)
endif
ifndef CLOUD_RUN_REGION
	$(error CLOUD_RUN_REGION is not set)
endif

## _check-update-env: Validate Cloud Run update variables (no image required)
.PHONY: _check-update-env
_check-update-env:
ifndef GCP_PROJECT
	$(error GCP_PROJECT is not set)
endif
ifndef CLOUD_RUN_SERVICE
	$(error CLOUD_RUN_SERVICE is not set)
endif
ifndef CLOUD_RUN_REGION
	$(error CLOUD_RUN_REGION is not set)
endif
