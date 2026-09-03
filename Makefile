MODULE     := github.com/samuelsulo/kitsu
BIN        := kitsu
VERSION    := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT     := $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE       := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS    := -X '$(MODULE)/internal/version.Version=$(VERSION)' \
              -X '$(MODULE)/internal/version.Commit=$(COMMIT)' \
              -X '$(MODULE)/internal/version.Date=$(DATE)'

.PHONY: build install run test lint fmt-check clean

build: ## Build the kitsu binary into ./bin
	go build -ldflags "$(LDFLAGS)" -o bin/$(BIN) ./cmd/kitsu

install: ## Build and install kitsu into $GOBIN (or $GOPATH/bin)
	go install -ldflags "$(LDFLAGS)" ./cmd/kitsu

run: ## Run kitsu from source, e.g. make run ARGS="version"
	go run ./cmd/kitsu $(ARGS)

test: ## Run the test suite
	go test ./...

fmt-check: ## Fail if any file is not gofmt-formatted
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "gofmt needed on:"; echo "$$unformatted"; exit 1; \
	fi

lint: fmt-check ## Static checks (gofmt + go vet)
	go vet ./...

clean: ## Remove build artifacts
	rm -rf bin
