# Makefile for mlcartifact

.PHONY: all build build-server build-cli test clean dist-ts proto

all: build

# Go Build
build: build-server build-cli

build-server:
	go build -o ./bin/artifact-server ./cmd/server

build-cli:
	go build -o ./bin/artifact-cli ./cmd/cli/cmd/artifact-cli

test:
	go test ./...

clean:
	rm -rf ./bin ./client-ts/dist ./client-ts/src/gen

# Protobuf generation
proto:
	cd proto && protoc --go_out=. --go_opt=paths=source_relative 
		--go-grpc_out=. --go-grpc_opt=paths=source_relative 
		--connect-go_out=. --connect-go_opt=paths=source_relative 
		artifact.proto

# TypeScript Client Distribution Build
dist-ts:
	@echo "Building TypeScript Universal Library (ES6+)..."
	cd client-ts && npm install
	cd client-ts && npm run generate
	cd client-ts && npm run build
	@echo "Build complete. Artifacts are in client-ts/dist/"

# Help
help:
	@echo "Available targets:"
	@echo "  build         - Build Go server and CLI"
	@echo "  test          - Run Go tests"
	@echo "  proto         - Regenerate all Protobuf/Connect files"
	@echo "  dist-ts       - Build the universal TypeScript ES6+ library"
	@echo "  clean         - Remove build artifacts"
