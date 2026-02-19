# Mau E2E Testing Framework Makefile

.PHONY: help build-image install-cli test-e2e test-e2e-ci test-basic test-resilience test-stress test-security clean

# Configuration
DOCKER_IMAGE := mau-e2e:latest
DOCKER_IMAGE_DEV := mau-e2e:dev
GO_VERSION := 1.21

help: ## Show this help message
	@echo "Mau E2E Testing Framework"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

# Docker Image Build
build-image: ## Build Mau E2E Docker image
	@echo "Building Mau E2E Docker image..."
	docker build -t $(DOCKER_IMAGE) -f docker/Dockerfile.mau ..
	@echo "✓ Image built: $(DOCKER_IMAGE)"

build-image-dev: ## Build development image with debug symbols
	@echo "Building Mau E2E development image..."
	docker build --build-arg DEBUG=true -t $(DOCKER_IMAGE_DEV) -f docker/Dockerfile.mau ..
	@echo "✓ Development image built: $(DOCKER_IMAGE_DEV)"

push-image: build-image ## Push image to registry (requires login)
	docker tag $(DOCKER_IMAGE) ghcr.io/mau-network/mau-e2e:latest
	docker push ghcr.io/mau-network/mau-e2e:latest

# CLI Installation
install-cli: ## Install mau-e2e CLI tool
	@echo "Installing mau-e2e CLI..."
	go install ./cmd/mau-e2e
	@echo "✓ CLI installed to $(shell go env GOPATH)/bin/mau-e2e"
	@echo "  Run 'mau-e2e --help' to get started"

# Testing - Automated
test-e2e: build-image ## Run all E2E tests
	@echo "Running all E2E tests..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 60m ./scenarios/...

test-e2e-ci: build-image ## Run E2E tests optimized for CI
	@echo "Running E2E tests (CI mode)..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) \
	MAU_E2E_CONFIG=configs/ci.json \
	go test -v -timeout 30m -race ./scenarios/basic/... ./scenarios/resilience/...

test-basic: build-image ## Run basic functionality tests (Level 1)
	@echo "Running basic tests..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 10m ./scenarios/basic/...

test-resilience: build-image ## Run resilience tests (Level 2)
	@echo "Running resilience tests..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 30m ./scenarios/resilience/...

test-stress: build-image ## Run stress tests (Level 3, slow)
	@echo "Running stress tests..."
	@echo "⚠️  This may take 30+ minutes"
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 60m ./scenarios/stress/...

test-security: build-image ## Run security tests (Level 4)
	@echo "Running security tests..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 20m ./scenarios/security/...

test-watch: ## Run tests in watch mode (requires gotestsum)
	@which gotestsum > /dev/null || (echo "Install gotestsum: go install gotest.tools/gotestsum@latest" && exit 1)
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) gotestsum --watch -- -timeout 10m ./scenarios/basic/...

# Testing - Coverage
test-coverage: build-image ## Run tests with coverage report
	@echo "Running tests with coverage..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -v -timeout 30m -coverprofile=coverage.out ./scenarios/...
	go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report: coverage.html"

# Interactive CLI
cli-demo: build-image install-cli ## Run interactive CLI demo
	@echo "Starting interactive demo..."
	@echo "1. Run: mau-e2e up --peers 3"
	@echo "2. Run: mau-e2e shell"
	@echo ""
	mau-e2e shell || true

# Development
dev-setup: ## Set up development environment
	@echo "Setting up E2E development environment..."
	go mod download
	go install github.com/spf13/cobra-cli@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "✓ Development tools installed"

lint: ## Run linters
	golangci-lint run ./...

fmt: ## Format code
	go fmt ./...
	gofmt -s -w .

# Documentation
docs-serve: ## Serve documentation locally
	@which mkdocs > /dev/null || (echo "Install mkdocs: pip install mkdocs-material" && exit 1)
	mkdocs serve -f ../mkdocs.yml

# Cleanup
clean: ## Clean up Docker resources and test artifacts
	@echo "Cleaning up..."
	-docker rm -f $$(docker ps -aq --filter label=mau-e2e) 2>/dev/null
	-docker network rm $$(docker network ls --filter name=mau-test -q) 2>/dev/null
	-docker volume rm $$(docker volume ls --filter name=mau-test -q) 2>/dev/null
	rm -rf test-results/
	rm -rf coverage.out coverage.html
	rm -rf ~/.mau-e2e/current-env.json
	@echo "✓ Cleanup complete"

clean-images: ## Remove E2E Docker images
	docker rmi $(DOCKER_IMAGE) $(DOCKER_IMAGE_DEV) || true

clean-all: clean clean-images ## Deep clean (images + resources)

# Utilities
logs: ## Show logs from last test run
	@test -d test-results || (echo "No test results found" && exit 1)
	@echo "Test results in: test-results/"
	@find test-results -name "*.json" -exec echo "  {}" \;

snapshot: ## Create snapshot of current environment (requires active env)
	@which mau-e2e > /dev/null || (echo "Install CLI first: make install-cli" && exit 1)
	mau-e2e snapshot snapshot-$$(date +%Y%m%d-%H%M%S)

# CI Helpers
ci-test-basic: ## CI: Run basic tests with artifact upload
	$(MAKE) test-basic
	@mkdir -p ci-artifacts
	@cp -r test-results ci-artifacts/ 2>/dev/null || true

ci-test-resilience: ## CI: Run resilience tests
	$(MAKE) test-resilience
	@mkdir -p ci-artifacts
	@cp -r test-results ci-artifacts/ 2>/dev/null || true

# Scenarios
scenario-5-peer-mesh: build-image install-cli ## Run 5-peer mesh scenario interactively
	mau-e2e scenario 5-peer-mesh

scenario-chaos: build-image install-cli ## Run chaos scenario interactively
	mau-e2e scenario chaos-kill

# Benchmarks
benchmark: build-image ## Run performance benchmarks
	@echo "Running benchmarks..."
	MAU_E2E_IMAGE=$(DOCKER_IMAGE) go test -bench=. -benchmem -timeout 30m ./scenarios/stress/...

# Version info
version: ## Show version information
	@echo "Mau E2E Framework"
	@echo "  Docker Image: $(DOCKER_IMAGE)"
	@echo "  Go Version:   $(GO_VERSION)"
	@go version
	@docker version --format '  Docker:       {{.Server.Version}}'
