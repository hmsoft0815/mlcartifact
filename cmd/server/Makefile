BINARY     := artifact-server
BUILD_DIR  := ./build
GO_FLAGS   := -trimpath
CGO_FLAGS  := CGO_ENABLED=0

PACKAGES := $(shell go list ./...)

.PHONY: all build test clean fmt vet lint check help test-client

all: check build ## Run check and build

deps: ## Download dependencies
	go mod download
	go mod tidy

fmt: ## Run go fmt
	go fmt $(PACKAGES)

vet: ## Run go vet
	$(CGO_FLAGS) go vet $(PACKAGES)

lint: ## Run golangci-lint
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not found, skipping."; \
	fi

check: fmt vet lint test ## Run all checks

build: deps ## Build binary
	@mkdir -p $(BUILD_DIR)
	$(CGO_FLAGS) go build $(GO_FLAGS) -o $(BUILD_DIR)/$(BINARY) ./main.go
	@echo "Built: $(BUILD_DIR)/$(BINARY)"

test: ## Run unit tests
	$(CGO_FLAGS) go test ./... -v -count=1

test-client: build ## Run integration test
	go run ./cmd/testclient/main.go

test-sse: build ## Run SSE integration test via curl
	./scripts/test_sse.sh

clean: ## Clean build artifacts
	rm -rf $(BUILD_DIR)
	rm -rf .artifacts

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
