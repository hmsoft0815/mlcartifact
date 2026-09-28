# Makefile for mlcartifact

# Variables
BINARY_DIR := ./bin
VERSION := $(shell cat VERSION 2>/dev/null || git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: all build build-server build-cli test test-all test-go test-ts test-python test-rust test-verbose test-cover lint tidy clean dist-ts proto run-server run-server-sse version-sync version-set help

all: build

# Go Build
build: build-server build-cli

build-server:
	mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-server ./cmd/artifact-server

build-cli:
	mkdir -p $(BINARY_DIR)
	go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-cli ./cmd/artifact-cli

# Cross-platform builds
build-all: build-linux build-windows build-macos

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-server-linux-amd64 ./cmd/artifact-server
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-cli-linux-amd64 ./cmd/artifact-cli

build-windows:
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-server-windows-amd64.exe ./cmd/artifact-server
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-cli-windows-amd64.exe ./cmd/artifact-cli

build-macos: build-macos-amd64 build-macos-arm64

build-macos-amd64:
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-server-macos-amd64 ./cmd/artifact-server
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-cli-macos-amd64 ./cmd/artifact-cli

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-server-macos-arm64 ./cmd/artifact-server
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BINARY_DIR)/artifact-cli-macos-arm64 ./cmd/artifact-cli

# Testing
test: test-go

test-all: test-go test-ts test-python test-rust

test-go:
	go test -count=1 ./...

test-ts:
	cd client-ts && npm test

test-python:
	PYTHONPATH=./client-python python3 -m unittest discover -s client-python/tests

test-rust:
	cd client-rust && cargo test

test-verbose:
	go test -v ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

# Linting & Maintenance
lint:
	golangci-lint run ./...

tidy:
	go mod tidy

clean:
	rm -rf $(BINARY_DIR) coverage.out coverage.html
	rm -rf ./client-ts/dist ./client-ts/src/gen

# Version Management
version-sync:
	./scripts/sync-version.sh

version-set:
	@if [ -z "$(V)" ]; then echo "Usage: make version-set V=x.y.z"; exit 1; fi
	echo "$(V)" > VERSION
	./scripts/sync-version.sh

# Running
run-server:
	go run ./cmd/artifact-server

run-server-sse:
	go run ./cmd/artifact-server -addr :8082 -grpc-addr :9590

# Protobuf generation
proto: proto-go proto-ts proto-python

proto-go:
	cd proto && protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--connect-go_out=. --connect-go_opt=paths=source_relative \
		artifact.proto

proto-ts:
	cd client-ts && npm run generate

proto-python:
	mkdir -p client-python/mlcartifact/gen
	python3 -m grpc_tools.protoc -I proto --python_out=client-python/mlcartifact/gen \
		--pyi_out=client-python/mlcartifact/gen \
		proto/artifact.proto
	touch client-python/mlcartifact/gen/__init__.py

# TypeScript Client Distribution Build
dist-ts:
	@echo "Building TypeScript Universal Library (ES6+)..."
	cd client-ts && npm ci
	cd client-ts && npm run build
	@echo "Build complete. Artifacts are in client-ts/dist/"

# Examples
.PHONY: run-example-go run-example-python run-example-ts run-example-rust run-examples

run-example-go:
	@echo "Running Go example..."
	go run ./examples/go/main.go

run-example-python:
	@echo "Running Python example..."
	PYTHONPATH=./client-python python3 client-python/example.py

run-example-ts:
	@echo "Running TypeScript example..."
	cd client-ts && npm run example

run-example-rust:
	@echo "Running Rust example..."
	cd client-rust && cargo run --example basic

run-examples: run-example-go run-example-python run-example-ts run-example-rust

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build Go server and CLI with version injection"
	@echo "  test          - Run Go tests"
	@echo "  test-all      - Run tests for Go, TypeScript, Python, and Rust clients"
	@echo "  test-verbose  - Run tests with verbose output"
	@echo "  test-cover    - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  tidy          - Tidy go.mod"
	@echo "  version-sync  - Sync VERSION file across all clients and server"
	@echo "  version-set   - Set new VERSION (e.g. make version-set V=0.5.6) and sync"
	@echo "  proto         - Regenerate all Protobuf/Connect files"
	@echo "  dist-ts       - Build the universal TypeScript ES6+ library"
	@echo "  run-server    - Run server in stdio mode"
	@echo "  run-server-sse - Run server in HTTP mode on :8082 (/mcp, /sse)"
	@echo "  run-example-go     - Run the Go client example"
	@echo "  run-example-python - Run the Python client example"
	@echo "  run-example-ts     - Run the TypeScript client example"
	@echo "  run-example-rust   - Run the Rust client example"
	@echo "  run-examples       - Run all client examples"
	@echo "  build-all          - Build for all platforms (Linux, Windows, macOS)"
	@echo "  build-linux        - Build for Linux amd64"
	@echo "  build-windows      - Build for Windows amd64"
	@echo "  build-macos        - Build for macOS (Intel & Silicon)"
	@echo "  clean              - Remove build artifacts"
