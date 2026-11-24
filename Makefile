VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)

# Load .env for local runs when it exists.
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run build test vet tidy

run:
	go run -ldflags "$(LDFLAGS)" ./cmd/api

build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o bin/medpath-server ./cmd/api

test:
	go test -race ./...

vet:
	go vet ./...

tidy:
	go mod tidy
