.PHONY: build build-all test test-cover lint clean install

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS := -ldflags "-X resticm/cmd.version=$(VERSION) \
                     -X resticm/cmd.commit=$(COMMIT) \
                     -X resticm/cmd.buildDate=$(DATE)"

# Binary name
BINARY := resticm

# Default target
all: build

# Build for current platform
build:
	go build $(LDFLAGS) -o $(BINARY) .

# Build for all platforms
build-all: clean
	mkdir -p dist
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64 .
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe .

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Run tests with race detector
test-race:
	go test -race -v ./...

# Lint
lint:
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install with:" && \
		echo "  brew install golangci-lint  (macOS)" && \
		echo "  or visit: https://golangci-lint.run/usage/install/" && \
		exit 1)
	golangci-lint run

# Clean build artifacts
clean:
	rm -f $(BINARY)
	rm -rf dist/
	rm -f coverage.out coverage.html

# Install to /usr/local/bin
install: build
	sudo cp $(BINARY) /usr/local/bin/$(BINARY)
	sudo chmod +x /usr/local/bin/$(BINARY)

# Uninstall
uninstall:
	sudo rm -f /usr/local/bin/$(BINARY)

# Generate shell completions
completions: build
	mkdir -p completions
	./$(BINARY) completion bash > completions/$(BINARY).bash
	./$(BINARY) completion zsh > completions/$(BINARY).zsh
	./$(BINARY) completion fish > completions/$(BINARY).fish

# Development: build and run
dev: build
	./$(BINARY)

# Show help
help:
	@echo "Available targets:"
	@echo "  build       - Build for current platform"
	@echo "  build-all   - Build for all platforms"
	@echo "  test        - Run tests"
	@echo "  test-cover  - Run tests with coverage"
	@echo "  test-race   - Run tests with race detector"
	@echo "  lint        - Run linter"
	@echo "  clean       - Clean build artifacts"
	@echo "  install     - Install to /usr/local/bin"
	@echo "  uninstall   - Remove from /usr/local/bin"
	@echo "  completions - Generate shell completions"
	@echo "  help        - Show this help"
