# Mau E2E Testing Framework Makefile

.PHONY: help build-image test-basic test-resilience test-stress test-security test-all clean

# Default target
help:
	@echo "Mau E2E Testing Framework"
	@echo ""
	@echo "Available targets:"
	@echo "  build-image       Build the Mau E2E Docker image"
	@echo "  test-basic        Run basic functionality tests (Level 1)"
	@echo "  test-resilience   Run resilience/chaos tests (Level 2)"
	@echo "  test-stress       Run stress tests (Level 3)"
	@echo "  test-security     Run security tests (Level 4)"
	@echo "  test-all          Run all test suites"
	@echo "  clean             Remove test artifacts and images"
	@echo ""
	@echo "Examples:"
	@echo "  make build-image"
	@echo "  make test-basic"
	@echo "  MAU_E2E_IMAGE=custom:tag make test-all"

# Configuration
MAU_E2E_IMAGE ?= mau-e2e:latest
DOCKER_BUILD_ARGS ?=
GO_TEST_FLAGS ?= -v -timeout 30m
TEST_RESULTS_DIR ?= test-results

# Build Mau E2E Docker image
build-image:
	@echo "Building Mau E2E Docker image: $(MAU_E2E_IMAGE)"
	@echo "Note: Dockerfile not yet created (Phase 1 deliverable)"
	@echo "Expected path: docker/Dockerfile.mau"
	# docker build -t $(MAU_E2E_IMAGE) -f docker/Dockerfile.mau $(DOCKER_BUILD_ARGS) ..

# Run basic tests (Level 1)
test-basic:
	@echo "Running basic functionality tests..."
	@echo "Note: Tests not yet implemented (Phase 1 deliverable)"
	# go test $(GO_TEST_FLAGS) ./scenarios/basic/...

# Run resilience tests (Level 2)
test-resilience:
	@echo "Running resilience/chaos tests..."
	@echo "Note: Tests not yet implemented (Phase 3 deliverable)"
	# go test $(GO_TEST_FLAGS) ./scenarios/resilience/...

# Run stress tests (Level 3)
test-stress:
	@echo "Running stress tests..."
	@echo "Note: Tests not yet implemented (Phase 4 deliverable)"
	# go test $(GO_TEST_FLAGS) ./scenarios/stress/...

# Run security tests (Level 4)
test-security:
	@echo "Running security tests..."
	@echo "Note: Tests not yet implemented (Phase 6 deliverable)"
	# go test $(GO_TEST_FLAGS) ./scenarios/security/...

# Run all test suites
test-all: test-basic test-resilience test-security
	@echo "All test suites completed"
	@echo "Note: Stress tests excluded (run separately with 'make test-stress')"

# Clean up test artifacts and images
clean:
	@echo "Cleaning up test artifacts..."
	rm -rf $(TEST_RESULTS_DIR)
	@echo "Note: Run 'docker rmi $(MAU_E2E_IMAGE)' manually to remove image"

# Validate framework design
validate-design:
	@echo "Validating E2E framework design..."
	@echo "✓ PLAN.md exists"
	@test -f PLAN.md || (echo "✗ PLAN.md missing" && exit 1)
	@echo "✓ README.md exists"
	@test -f README.md || (echo "✗ README.md missing" && exit 1)
	@echo "✓ Makefile exists"
	@test -f Makefile || (echo "✗ Makefile missing" && exit 1)
	@echo ""
	@echo "Design validation passed! Ready for Phase 1 implementation."
	@echo "See PLAN.md for implementation roadmap."
