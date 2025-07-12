.PHONY: help build clean test install test-coverage fmt lint check

# Default target - show help
help:
	@echo "lnk - Symlink Management Tool"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Variables:"
	@echo "  PREFIX=$(PREFIX)     Installation prefix (override with PREFIX=/path)"
	@echo "  BINDIR=$(BINDIR)     Binary directory (override with BINDIR=/path)"
	@echo ""
	@echo "Targets:"
	@echo "  help           Show this help message"
	@echo "  build          Build the lnk binary"
	@echo "  install        Build and install lnk to BINDIR"
	@echo "  clean          Remove build artifacts and installed binary"
	@echo "  test           Run all tests with verbose output"
	@echo "  test-coverage  Run tests with coverage report (generates HTML)"
	@echo "  fmt            Format all Go code"
	@echo "  lint           Run go vet for static analysis"
	@echo "  check          Run fmt, test, and lint in sequence"

# Installation prefix
PREFIX ?= /usr/local
BINDIR ?= $(PREFIX)/bin

# Build the lnk binary
build:
	mkdir -p bin
	@# Get version from git or use "dev" as fallback
	@VERSION=$$(git describe --tags --always 2>/dev/null || echo "dev"); \
	COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown"); \
	DATE=$$(date -u '+%Y-%m-%d %H:%M:%S UTC' 2>/dev/null || date); \
	echo "Building lnk version $$VERSION ($$COMMIT)..."; \
	go build -ldflags "-X 'main.version=$$VERSION' -X 'main.commit=$$COMMIT' -X 'main.date=$$DATE'" -o bin/lnk cmd/lnk/main.go

# Install lnk to BINDIR
install: build
	@# Check if we can write to the target directory
	@if [ -d "$(BINDIR)" ]; then \
		if [ ! -w "$(BINDIR)" ]; then \
			echo "Error: Cannot write to $(BINDIR)"; \
			echo "Try one of the following:"; \
			echo "  sudo make install"; \
			echo "  make install PREFIX=~/.local"; \
			exit 1; \
		fi \
	else \
		parent_dir=$$(dirname "$(BINDIR)"); \
		if [ ! -w "$$parent_dir" ]; then \
			echo "Error: Cannot write to $$parent_dir"; \
			echo "Try one of the following:"; \
			echo "  sudo make install"; \
			echo "  make install PREFIX=~/.local"; \
			exit 1; \
		fi \
	fi
	mkdir -p $(BINDIR)
	cp bin/lnk $(BINDIR)/lnk
	chmod +x $(BINDIR)/lnk

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html
	rm -f $(BINDIR)/lnk

# Run tests
test:
	go test -v ./...

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