VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
SWAG_VERSION := v1.16.6

# Load .env for local runs when it exists.
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run build test vet tidy docs

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

# Regenerate docs/ from the swag annotations. Run after changing a handler's annotations.
docs:
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) init \
		--generalInfo cmd/api/main.go --parseInternal \
		--output docs --outputTypes go,json
	go run github.com/swaggo/swag/cmd/swag@$(SWAG_VERSION) fmt
