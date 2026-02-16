.PHONY: help build clean test test-unit test-e2e test-coverage clean-test fmt lint check

# Default target - show help
help:
	@echo "lnk - Symlink Management Tool"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  help           Show this help message"
	@echo "  build          Build the lnk binary"
	@echo "  clean          Remove build artifacts"
	@echo "  test           Run all tests with verbose output"
	@echo "  test-unit      Run unit tests only"
	@echo "  test-e2e       Run end-to-end tests only"
	@echo "  test-coverage  Run tests with coverage report (generates HTML)"
	@echo "  clean-test     Clean up test artifacts"
	@echo "  fmt            Format all Go code"
	@echo "  lint           Run go vet for static analysis"
	@echo "  check          Run fmt, test, and lint in sequence"

# Build the lnk binary
build:
	mkdir -p bin
	@# Generate dev+timestamp for local builds (releases override via ldflags)
	@VERSION=$$(date -u '+dev+%Y%m%d%H%M%S'); \
	echo "Building lnk $$VERSION..."; \
	go build -ldflags "-X 'main.version=$$VERSION'" -o bin/lnk cmd/lnk/main.go

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Clean test artifacts
clean-test:
	rm -rf e2e/testdata/
	@echo "Test data cleaned. Run 'scripts/setup-testdata.sh' to recreate."

# Run tests
test:
	go test -v ./...

# Run unit tests only
test-unit:
	go test -v ./internal/...

# Run E2E tests only
test-e2e:
	go test -v ./e2e/...

# Run tests with coverage
test-coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Format code
fmt:
	@if command -v goimports >/dev/null 2>&1; then \
		echo "Running goimports..."; \
		goimports -w .; \
	else \
		echo "goimports not found, using gofmt..."; \
		gofmt -w .; \
	fi

# Run static analysis with go vet
lint:
	@echo "Running go vet..."
	go vet ./...

# Run all checks
check: fmt test lint
