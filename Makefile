MODULE  := github.com/servusdei2018/sandbox
BINARY  := sandbox
OUTDIR  := bin

VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -X main.Version=$(VERSION) \
           -X main.Commit=$(COMMIT) \
           -X main.BuildDate=$(BUILD_DATE)

CMD_DIR := ./cmd/sandbox

.PHONY: all build build-prod clean test test-unit test-integration lint fmt vet install help version images image-claude image-codex image-opencode

all: build

## help: Show this help message.
help:
	@echo "Usage: make <target>"
	@echo ""
	@grep -E '^## [a-zA-Z_-]+:' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}' | \
		sed 's/## //'

## build: Build the debug binary to ./bin/sandbox.
build:
	@mkdir -p $(OUTDIR)
	go build -ldflags="$(LDFLAGS)" -o $(OUTDIR)/$(BINARY) $(CMD_DIR)
	@echo "→ Built $(OUTDIR)/$(BINARY) (version: $(VERSION))"

## build-prod: Build a stripped, statically linked production binary.
build-prod:
	@mkdir -p $(OUTDIR)
	CGO_ENABLED=0 go build \
		-ldflags="-s -w $(LDFLAGS)" \
		-trimpath \
		-o $(OUTDIR)/$(BINARY) \
		$(CMD_DIR)
	@echo "→ Built production binary $(OUTDIR)/$(BINARY)"

## version: Print the version that would be embedded in the binary.
version:
	@echo "version:    $(VERSION)"
	@echo "commit:     $(COMMIT)"
	@echo "build_date: $(BUILD_DATE)"

## install: Install the binary to GOPATH/bin.
install:
	go install -ldflags="$(LDFLAGS)" $(CMD_DIR)
	@echo "→ Installed $(BINARY) to $$(go env GOPATH)/bin"

## fmt: Format all Go source files.
fmt:
	gofmt -s -w .
	@echo "→ Formatted"

## vet: Run go vet.
vet:
	go vet ./...

## lint: Run golangci-lint (requires golangci-lint to be installed).
lint:
	@command -v golangci-lint > /dev/null || (echo "golangci-lint not found, install: https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

## test: Run all unit tests (excludes integration tests).
test: test-unit

## test-unit: Run unit tests only.
test-unit:
	go test -race -count=1 -timeout=60s ./pkg/... ./cmd/...

## test-integration: Run integration tests (requires a running Docker daemon).
test-integration:
	go test -race -count=1 -timeout=5m -tags=integration ./tests/integration/...

## clean: Remove build artifacts.
clean:
	rm -rf $(OUTDIR)
	@echo "→ Cleaned"

## images: Build all agent Docker images from the deployments directory.
images: image-claude image-codex image-opencode image-gemini image-kilocode

## publish-images: Publish all built agent Docker images to GHCR.
publish-images: images
	docker push ghcr.io/servusdei2018/sandbox-claude:latest
	docker push ghcr.io/servusdei2018/sandbox-codex:latest
	docker push ghcr.io/servusdei2018/sandbox-opencode:latest
	docker push ghcr.io/servusdei2018/sandbox-gemini:latest
	docker push ghcr.io/servusdei2018/sandbox-kilocode:latest

## image-claude: Build the sandbox-claude image.
image-claude:
	docker build -t sandbox-claude:latest -t ghcr.io/servusdei2018/sandbox-claude:latest -f deployments/Dockerfile.claude deployments/

## image-codex: Build the sandbox-codex image.
image-codex:
	docker build -t sandbox-codex:latest -t ghcr.io/servusdei2018/sandbox-codex:latest -f deployments/Dockerfile.codex deployments/

## image-opencode: Build the sandbox-opencode image.
image-opencode:
	docker build -t sandbox-opencode:latest -t ghcr.io/servusdei2018/sandbox-opencode:latest -f deployments/Dockerfile.opencode deployments/

## image-gemini: Build the sandbox-gemini image.
image-gemini:
	docker build -t sandbox-gemini:latest -t ghcr.io/servusdei2018/sandbox-gemini:latest -f deployments/Dockerfile.gemini deployments/

## image-kilocode: Build the sandbox-kilocode image.
image-kilocode:
	docker build -t sandbox-kilocode:latest -t ghcr.io/servusdei2018/sandbox-kilocode:latest -f deployments/Dockerfile.kilocode deployments/

## tidy: Tidy and verify Go modules.
tidy:
	go mod tidy
	go mod verify
