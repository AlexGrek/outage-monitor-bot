.PHONY: build run clean setcap install test dev api-key docker-build docker-push docker-run docker-tag docker-clean

# Build variables
BINARY_NAME=tg-monitor-bot
BUILD_DIR=bin
MAIN_PATH=./cmd/bot

# Docker variables
DOCKER_REGISTRY?=docker.io
DOCKER_USERNAME?=yourusername
DOCKER_IMAGE=$(DOCKER_REGISTRY)/$(DOCKER_USERNAME)/$(BINARY_NAME)
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
DOCKER_TAG=$(DOCKER_IMAGE):$(VERSION)
DOCKER_LATEST=$(DOCKER_IMAGE):latest

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

# Docker build
docker-build:
	@echo "Building Docker image $(DOCKER_TAG)..."
	docker build \
		--build-arg VERSION=$(VERSION) \
		-t $(DOCKER_TAG) \
		-t $(DOCKER_LATEST) \
		.
	@echo "Docker image built: $(DOCKER_TAG)"

# Docker push
docker-push: docker-build
	@echo "Pushing Docker image $(DOCKER_TAG)..."
	docker push $(DOCKER_TAG)
	docker push $(DOCKER_LATEST)
	@echo "Docker images pushed successfully"

# Docker tag (create additional tags)
docker-tag:
	@echo "Tagging Docker image..."
	docker tag $(DOCKER_TAG) $(DOCKER_IMAGE):$(TAG)
	@echo "Tagged as $(DOCKER_IMAGE):$(TAG)"

# Docker run locally
docker-run:
	@echo "Running Docker container..."
	docker run --rm -it \
		--name $(BINARY_NAME) \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		$(DOCKER_LATEST)

# Docker run in background
docker-run-detached:
	@echo "Running Docker container in background..."
	docker run -d \
		--name $(BINARY_NAME) \
		--restart unless-stopped \
		--env-file .env \
		-v $(PWD)/data:/app/data \
		$(DOCKER_LATEST)
	@echo "Container started. View logs with: docker logs -f $(BINARY_NAME)"

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
	@echo "Docker:"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-push   - Build and push Docker image to registry"
	@echo "  make docker-run    - Run Docker container interactively"
	@echo "  make docker-run-detached - Run Docker container in background"
	@echo "  make docker-stop   - Stop and remove Docker container"
	@echo "  make docker-logs   - View Docker container logs"
	@echo "  make docker-tag    - Tag Docker image (use TAG=version)"
	@echo "  make docker-clean  - Remove Docker images"
	@echo ""
	@echo "Environment variables:"
	@echo "  DOCKER_REGISTRY    - Docker registry (default: docker.io)"
	@echo "  DOCKER_USERNAME    - Docker username (default: yourusername)"
	@echo "  VERSION            - Version tag (default: git describe)"
	@echo "  TAG                - Additional tag for docker-tag target"
