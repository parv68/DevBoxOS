# DevBoxOS Makefile

VERSION ?= 0.1.0-dev
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

LDFLAGS := -ldflags "-X github.com/devboxos/devboxos/cli/cmd.version=$(VERSION) -X github.com/devboxos/devboxos/cli/cmd.commit=$(COMMIT) -X github.com/devboxos/devboxos/cli/cmd.date=$(DATE)"

.PHONY: all build build-cli build-engine clean test proto help

all: build

## build: Build CLI and Engine
build: build-cli build-engine

## build-cli: Build the CLI
build-cli:
	@echo "Building CLI..."
	cd cli && go build $(LDFLAGS) -o ../dist/devbox .

## build-engine: Build the engine daemon
build-engine:
	@echo "Building Engine..."
	cd engine && go build $(LDFLAGS) -o ../dist/devbox-engine .

## proto: Generate protobuf code
proto:
	@echo "Generating protobuf code..."
	cd engine && protoc \
		--go_out=. \
		--go_opt=paths=source_relative \
		--go-grpc_out=. \
		--go-grpc_opt=paths=source_relative \
		proto/engine.proto

## test: Run all tests
test:
	@echo "Running tests..."
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out

## test-unit: Run unit tests only
test-unit:
	@echo "Running unit tests..."
	go test -race ./...

## clean: Remove build artifacts
clean:
	@echo "Cleaning..."
	rm -rf dist/
	rm -f coverage.out

## install: Install CLI to GOPATH/bin
install:
	@echo "Installing CLI..."
	cd cli && go install $(LDFLAGS) .

## help: Show this help
help:
	@grep -E '^## ' Makefile | sed 's/## //g' | column -t -s ':'
