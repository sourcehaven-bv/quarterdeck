.PHONY: build run clean install lint test release-snapshot

# Version info
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE    ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

# Build for current platform
build:
	go build -ldflags "$(LDFLAGS)" -o bin/quarterdeck .

# Build for Linux (server deployment)
build-linux:
	GOOS=linux GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/quarterdeck-linux .

# Run locally
run: build
	./bin/quarterdeck

# Install dependencies
deps:
	go mod download
	go mod tidy

# Clean build artifacts
clean:
	rm -rf bin/

# Install to /usr/local/bin
install: build
	cp bin/quarterdeck /usr/local/bin/

# Lint
lint:
	golangci-lint run

# Test
test:
	go test ./...

# Build release snapshot (for testing goreleaser locally)
release-snapshot:
	goreleaser release --snapshot --clean
