.PHONY: all build test test-race test-e2e test-e2e-short test-bench test-security test-integration clean lint vet fmt

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

GOEXE := $(shell $(GO) env GOEXE)
CLI_BIN := cli/devboxos$(GOEXE)

test-e2e:
	rm -f cli/devboxos cli/devboxos.exe
	$(GO) build -o $(CLI_BIN) ./cli
	$(GO) test $(GOFLAGS) -tags=e2e -count=1 ./tests/... -timeout 600s -v

test-e2e-short:
	rm -f cli/devboxos cli/devboxos.exe
	$(GO) build -o $(CLI_BIN) ./cli
	$(GO) test $(GOFLAGS) -tags=e2e -count=1 -short ./tests/... -timeout 300s -v

test-bench:
	rm -f cli/devboxos cli/devboxos.exe
	$(GO) build -o $(CLI_BIN) ./cli
	$(GO) test $(GOFLAGS) -tags=e2e -bench=. -benchtime=1x ./tests/... -run=^\$$ -timeout 600s -v 2>&1 | tee bench-results.txt

test-security:
	rm -f cli/devboxos cli/devboxos.exe
	$(GO) build -o $(CLI_BIN) ./cli
	$(GO) test $(GOFLAGS) -tags=e2e -count=1 -short ./tests/... -run TestSecurity -timeout 300s -v

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
