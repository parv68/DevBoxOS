.PHONY: all build test test-race test-integration clean lint vet fmt

GO ?= go
GOFLAGS ?= -mod=mod
TAGS ?=

all: vet build test

build:
	$(GO) build $(GOFLAGS) ./shared/...
	$(GO) build $(GOFLAGS) ./engine/...
	$(GO) build $(GOFLAGS) ./cli/...

test:
	$(GO) test $(GOFLAGS) -count=1 ./shared/... ./engine/... ./cli/... -timeout 180s

test-race:
	$(GO) test $(GOFLAGS) -race -count=1 ./shared/... ./engine/... ./cli/... -timeout 300s

test-integration:
	$(GO) test $(GOFLAGS) -tags=integration -count=1 ./shared/... ./engine/... ./cli/... -timeout 300s

test-verbose:
	$(GO) test $(GOFLAGS) -v -count=1 ./shared/... ./engine/... ./cli/... -timeout 180s

vet:
	$(GO) vet $(GOFLAGS) ./shared/...
	$(GO) vet $(GOFLAGS) ./engine/...
	$(GO) vet $(GOFLAGS) ./cli/...

fmt:
	$(GO) fmt ./shared/...
	$(GO) fmt ./engine/...
	$(GO) fmt ./cli/...

clean:
	$(GO) clean -cache
	rm -rf bin/

lint:
	$(GO) vet $(GOFLAGS) ./shared/...
	$(GO) vet $(GOFLAGS) ./engine/...
	$(GO) vet $(GOFLAGS) ./cli/...

coverage:
	$(GO) test $(GOFLAGS) -coverprofile=coverage.out -count=1 ./shared/... ./engine/... ./cli/... -timeout 180s
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out
