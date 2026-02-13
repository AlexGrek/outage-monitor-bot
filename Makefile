.PHONY: build run clean setcap install test dev api-key version-bump-patch version-bump-minor version-bump-major docker-build docker-build-local docker-stop-test docker-buildx-setup docker-build-amd64 docker-build-arm64 docker-build-multiarch docker-login docker-push docker-release docker-release-multiarch docker-run docker-tag docker-clean helm-create helm-package helm-install helm-upgrade helm-reinstall helm-uninstall helm-clean production-patch production-minor production-major

# Build variables
BINARY_NAME=tg-monitor-bot
BUILD_DIR=bin
MAIN_PATH=./cmd/bot

# Docker variables
DOCKER_REGISTRY?=docker.io
DOCKER_IMAGE?=grekodocker/outage-monitor-bot
DOCKER_FULL_IMAGE=$(DOCKER_REGISTRY)/$(DOCKER_IMAGE)
VERSION?=$(shell cat VERSION 2>/dev/null || echo "dev")
DOCKER_TAG=$(DOCKER_FULL_IMAGE):$(VERSION)
DOCKER_LATEST=$(DOCKER_FULL_IMAGE):latest
BUILDX_BUILDER?=multiarch-builder

# Build the application
build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(BUILD_DIR)
	go build -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build and run
run: build
	@echo "Starting $(BINARY_NAME)..."
	./$(BUILD_DIR)/$(BINARY_NAME)

# Install dependencies
install:
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Run tests
test:
	@echo "Running tests..."
	go test -v ./...

# Set capabilities for ICMP (ping) - requires sudo
# This allows the binary to send ICMP packets without running as root
setcap: build
	@echo "Setting capabilities for ICMP..."
	sudo setcap cap_net_raw=+ep $(BUILD_DIR)/$(BINARY_NAME)
	@echo "Capabilities set. You can now run ping without root."

# Clean build artifacts
clean:
	@echo "Cleaning..."
	rm -rf $(BUILD_DIR)
	@echo "Clean complete"

# Version management
version-bump-patch:
	@./scripts/version-bump.sh patch

version-bump-minor:
	@./scripts/version-bump.sh minor

version-bump-major:
	@./scripts/version-bump.sh major

# Build for production (with optimizations)
build-prod:
	@echo "Building for production..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(MAIN_PATH)
	@echo "Production build complete"

# Create necessary directories
setup:
	@echo "Setting up directories..."
	@mkdir -p data logs
	@echo "Setup complete"

# Generate a secure API key
api-key:
	@echo "Generating secure API key..."
	@API_KEY=$$(openssl rand -hex 32 2>/dev/null || head -c 32 /dev/urandom | xxd -p); \
	echo ""; \
	echo "Your new API key:"; \
	echo "$$API_KEY"; \
	echo ""; \
	if [ -f .env ]; then \
		if grep -q "^API_KEY=" .env; then \
			if [[ "$$OSTYPE" == "darwin"* ]]; then \
				sed -i '' "s/^API_KEY=.*/API_KEY=$$API_KEY/" .env; \
			else \
				sed -i "s/^API_KEY=.*/API_KEY=$$API_KEY/" .env; \
			fi; \
			echo "✓ Updated API_KEY in .env"; \
		else \
			echo "API_KEY=$$API_KEY" >> .env; \
			echo "✓ Added API_KEY to .env"; \
		fi; \
	else \
		echo "⚠ .env file not found. Copy it to .env and run 'make api-key' again"; \
	fi

# Run both frontend and backend for development
dev:
	@./dev.sh

# Docker build (full multi-stage: frontend + backend + nginx)
docker-build:
	@echo "Building full-stack Docker image $(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_TAG) \
		-t $(DOCKER_LATEST) \
		.
	@echo "Docker image built: $(DOCKER_TAG)"

# Docker build and test locally
docker-build-local: docker-build
	@echo "Testing Docker image locally..."
	@echo "Starting container on port 3000..."
	@docker run --rm -d \
		--name $(BINARY_NAME)-test \
		-p 3000:80 \
		-p 8080:8080 \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		$(DOCKER_LATEST) && \
	echo "✓ Container started successfully" && \
	echo "" && \
	echo "Access the application:" && \
	echo "  Frontend: http://localhost:3000" && \
	echo "  API:      http://localhost:8080" && \
	echo "  Health:   http://localhost:3000/health" && \
	echo "" && \
	echo "To stop: make docker-stop-test" && \
	echo "To view logs: docker logs -f $(BINARY_NAME)-test"

# Stop test container
docker-stop-test:
	@echo "Stopping test container..."
	@docker stop $(BINARY_NAME)-test 2>/dev/null || true
	@echo "Test container stopped"

# Docker push
docker-push: docker-build
	@echo "Pushing Docker image $(DOCKER_TAG)..."
	docker push $(DOCKER_TAG)
	docker push $(DOCKER_LATEST)
	@echo ""
	@echo "✓ Docker images pushed successfully!"
	@echo "  - $(DOCKER_TAG)"
	@echo "  - $(DOCKER_LATEST)"

# Docker login
docker-login:
	@echo "Logging into Docker Hub..."
	@docker login
	@echo "✓ Logged in successfully"

# Docker push with login
docker-release: docker-login docker-push
	@echo ""
	@echo "✓ Release complete!"
	@echo "Image available at: $(DOCKER_LATEST)"

# Docker tag (create additional tags)
docker-tag:
	@echo "Tagging Docker image..."
	docker tag $(DOCKER_TAG) $(DOCKER_FULL_IMAGE):$(TAG)
	@echo "Tagged as $(DOCKER_FULL_IMAGE):$(TAG)"
	@echo ""
	@echo "To push this tag: docker push $(DOCKER_FULL_IMAGE):$(TAG)"

# Docker run locally
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it \
		--name $(BINARY_NAME) \
		-p 3000:80 \
		-p 8080:8080 \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		$(DOCKER_LATEST)

# Docker run in background
docker-run-detached:
	@echo "Running Docker container in background..."
	docker run -d \
		--name $(BINARY_NAME) \
		--restart unless-stopped \
		-p 3000:80 \
		-p 8080:8080 \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		$(DOCKER_LATEST)
	@echo "Container started successfully!"
	@echo ""
	@echo "Access the application:"
	@echo "  Frontend: http://localhost:3000"
	@echo "  API:      http://localhost:8080"
	@echo ""
	@echo "View logs with: make docker-logs"
	@echo "Stop with: make docker-stop"

# Stop Docker container
docker-stop:
	@echo "Stopping Docker container..."
	docker stop $(BINARY_NAME) || true
	docker rm $(BINARY_NAME) || true

# Docker logs
docker-logs:
	docker logs -f $(BINARY_NAME)

# Clean Docker images
docker-clean:
	@echo "Cleaning Docker images..."
	docker rmi $(DOCKER_TAG) $(DOCKER_LATEST) || true
	@echo "Docker images cleaned"

# Setup buildx builder for multi-arch
docker-buildx-setup:
	@echo "Setting up buildx builder..."
	@docker buildx create --name $(BUILDX_BUILDER) --use 2>/dev/null || \
		docker buildx use $(BUILDX_BUILDER) 2>/dev/null || \
		echo "Buildx builder already exists"
	@docker buildx inspect --bootstrap
	@echo "✓ Buildx builder ready"

# Build for AMD64 only (using buildx)
docker-build-amd64: docker-buildx-setup
	@echo "Building AMD64 image $(DOCKER_TAG)-amd64..."
	docker buildx build \
		--platform linux/amd64 \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_TAG)-amd64 \
		--load \
		.
	@echo "✓ AMD64 image built: $(DOCKER_TAG)-amd64"

# Build for ARM64 only (using buildx)
docker-build-arm64: docker-buildx-setup
	@echo "Building ARM64 image $(DOCKER_TAG)-arm64..."
	docker buildx build \
		--platform linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_TAG)-arm64 \
		--load \
		.
	@echo "✓ ARM64 image built: $(DOCKER_TAG)-arm64"

# Build multi-arch (AMD64 + ARM64) and push
docker-build-multiarch: docker-buildx-setup
	@echo "Building multi-arch image $(DOCKER_TAG)..."
	docker buildx build \
		--platform linux/amd64,linux/arm64 \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_TAG) \
		-t $(DOCKER_LATEST) \
		--push \
		.
	@echo ""
	@echo "✓ Multi-arch image built and pushed!"
	@echo "  Platforms: linux/amd64, linux/arm64"
	@echo "  Tags: $(DOCKER_TAG), $(DOCKER_LATEST)"

# Complete multi-arch release
docker-release-multiarch: docker-login docker-build-multiarch
	@echo ""
	@echo "✓ Multi-arch release complete!"
	@echo "  Image: $(DOCKER_LATEST)"
	@echo "  Platforms: AMD64, ARM64"

# Helm chart variables
HELM_CHART_NAME=tg-monitor-bot
HELM_CHART_DIR=helm/$(HELM_CHART_NAME)
HELM_RELEASE_NAME?=$(HELM_CHART_NAME)
HELM_NAMESPACE?=default

# Create Helm chart structure
helm-create:
	@echo "Creating Helm chart structure..."
	@./scripts/create-helm-chart.sh
	@echo "Helm chart created at $(HELM_CHART_DIR)"

# Package Helm chart
helm-package:
	@echo "Packaging Helm chart..."
	@helm package $(HELM_CHART_DIR) -d helm/
	@echo "Chart packaged successfully"

# Install Helm chart
helm-install: helm-package
	@echo "Installing Helm chart..."
	helm install $(HELM_RELEASE_NAME) \
		helm/$(HELM_CHART_NAME)-$(VERSION).tgz \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace
	@echo "Chart installed. Check status with: helm status $(HELM_RELEASE_NAME) -n $(HELM_NAMESPACE)"

# Upgrade Helm chart
helm-upgrade: helm-package
	@echo "Upgrading Helm chart..."
	helm upgrade $(HELM_RELEASE_NAME) \
		helm/$(HELM_CHART_NAME)-$(VERSION).tgz \
		--namespace $(HELM_NAMESPACE)
	@echo "Chart upgraded successfully"

# Uninstall Helm chart
helm-uninstall:
	@echo "Uninstalling Helm chart..."
	helm uninstall $(HELM_RELEASE_NAME) --namespace $(HELM_NAMESPACE)
	@echo "Chart uninstalled"

# Clean Helm packages
helm-clean:
	@echo "Cleaning Helm packages..."
	rm -f helm/*.tgz
	@echo "Helm packages cleaned"

# Production deployment targets
production-patch: version-bump-patch docker-build-multiarch helm-reinstall
	@echo ""
	@echo "✓ Production patch release complete!"
	@echo "  Version: $$(cat VERSION)"
	@echo "  Image: $(DOCKER_TAG)"
	@echo "  Helm: $(HELM_RELEASE_NAME) (namespace: $(HELM_NAMESPACE))"

production-minor: version-bump-minor docker-build-multiarch helm-reinstall
	@echo ""
	@echo "✓ Production minor release complete!"
	@echo "  Version: $$(cat VERSION)"
	@echo "  Image: $(DOCKER_TAG)"
	@echo "  Helm: $(HELM_RELEASE_NAME) (namespace: $(HELM_NAMESPACE))"

production-major: version-bump-major docker-build-multiarch helm-reinstall
	@echo ""
	@echo "✓ Production major release complete!"
	@echo "  Version: $$(cat VERSION)"
	@echo "  Image: $(DOCKER_TAG)"
	@echo "  Helm: $(HELM_RELEASE_NAME) (namespace: $(HELM_NAMESPACE))"

# Reinstall Helm chart (uninstall + install)
helm-reinstall:
	@echo "Reinstalling Helm chart..."
	@helm uninstall $(HELM_RELEASE_NAME) --namespace $(HELM_NAMESPACE) 2>/dev/null || true
	@sleep 2
	@$(MAKE) helm-install
	@echo "Helm chart reinstalled successfully"

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  make dev           - Start both backend and frontend for development"
	@echo "  make api-key       - Generate a new secure API key and update .env"
	@echo ""
	@echo "Local build:"
	@echo "  make build         - Build the application"
	@echo "  make run           - Build and run the application"
	@echo "  make install       - Install dependencies"
	@echo "  make test          - Run tests"
	@echo "  make setcap        - Set capabilities for ICMP (requires sudo)"
	@echo "  make clean         - Remove build artifacts"
	@echo "  make build-prod    - Build optimized production binary"
	@echo "  make setup         - Create necessary directories"
	@echo ""
	@echo "Version Management:"
	@echo "  make version-bump-patch  - Bump patch version (x.y.Z)"
	@echo "  make version-bump-minor  - Bump minor version (x.Y.0)"
	@echo "  make version-bump-major  - Bump major version (X.0.0)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build              - Build full-stack Docker image (default arch)"
	@echo "  make docker-build-local        - Build and test locally on port 3000"
	@echo "  make docker-stop-test          - Stop test container"
	@echo "  make docker-buildx-setup       - Setup buildx for multi-arch builds"
	@echo "  make docker-build-amd64        - Build AMD64 image only"
	@echo "  make docker-build-arm64        - Build ARM64 image only"
	@echo "  make docker-build-multiarch    - Build and push multi-arch (AMD64+ARM64)"
	@echo "  make docker-login              - Login to Docker Hub"
	@echo "  make docker-push               - Build and push Docker image to registry"
	@echo "  make docker-release            - Login, build, and push (single arch)"
	@echo "  make docker-release-multiarch  - Login, build, and push (multi-arch)"
	@echo "  make docker-run                - Run Docker container interactively"
	@echo "  make docker-run-detached       - Run Docker container in background"
	@echo "  make docker-stop               - Stop and remove Docker container"
	@echo "  make docker-logs               - View Docker container logs"
	@echo "  make docker-tag                - Tag Docker image (use TAG=version)"
	@echo "  make docker-clean              - Remove Docker images"
	@echo ""
	@echo "Helm (Kubernetes):"
	@echo "  make helm-create    - Create Helm chart structure"
	@echo "  make helm-package   - Package Helm chart"
	@echo "  make helm-install   - Install chart to Kubernetes"
	@echo "  make helm-upgrade   - Upgrade existing installation"
	@echo "  make helm-reinstall - Uninstall and reinstall chart"
	@echo "  make helm-uninstall - Uninstall chart from Kubernetes"
	@echo "  make helm-clean     - Remove packaged charts"
	@echo ""
	@echo "Production Deployment (Kubernetes):"
	@echo "  make production-patch  - Bump patch, build multi-arch, deploy (x.y.Z)"
	@echo "  make production-minor  - Bump minor, build multi-arch, deploy (x.Y.0)"
	@echo "  make production-major  - Bump major, build multi-arch, deploy (X.0.0)"
	@echo ""
	@echo "Environment variables:"
	@echo "  DOCKER_REGISTRY     - Docker registry (default: docker.io)"
	@echo "  DOCKER_IMAGE        - Docker image name (default: grekodocker/outage-monitor-bot)"
	@echo "  VERSION             - Version tag (default: from VERSION file)"
	@echo "  TAG                 - Additional tag for docker-tag target"
	@echo "  HELM_RELEASE_NAME   - Helm release name (default: tg-monitor-bot)"
	@echo "  HELM_NAMESPACE      - Kubernetes namespace (default: default)"
	@echo "  BUILDX_BUILDER      - Buildx builder name (default: multiarch-builder)"
