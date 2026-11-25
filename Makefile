VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -X main.version=$(VERSION)
SWAG_VERSION := v1.16.6
GOLANGCI_LINT_VERSION := v2.15.0
LEFTHOOK_VERSION := v2.2.3
COMMITLINT_VERSION := v0.12.0
BIN := $(CURDIR)/bin

# Load .env for local runs when it exists.
ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: run build test vet tidy docs tools hooks lint fmt docker-dev docker-build \
	docker-prod docker-prod-down docker-prod-logs docker-prod-ps prod-env

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

# Pinned dev tools installed into ./bin, outside go.mod so their dependencies never mix with ours.
tools:
	GOBIN=$(BIN) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	GOBIN=$(BIN) go install github.com/evilmartians/lefthook/v2@$(LEFTHOOK_VERSION)
	GOBIN=$(BIN) go install github.com/conventionalcommit/commitlint@$(COMMITLINT_VERSION)

hooks: tools
	$(BIN)/lefthook install

lint:
	$(BIN)/golangci-lint run ./...

fmt:
	$(BIN)/golangci-lint fmt ./...

PROD_ENV_FILE ?= .env.production
# env -i keeps the variables exported from .env above out of production: compose prefers
# the shell environment over --env-file, so a dev CORS origin would otherwise win.
PROD_COMPOSE = env -i PATH="$$PATH" HOME="$$HOME" $(if $(DOCKER_HOST),DOCKER_HOST="$(DOCKER_HOST)") \
	VERSION=$(VERSION) docker compose -f compose.prod.yaml --env-file $(PROD_ENV_FILE)

# Development stack: API with live reload plus Postgres.
docker-dev:
	docker compose up --build

# Production image, tagged with the git version.
docker-build:
	docker build --target prod --build-arg VERSION=$(VERSION) -t medpath-server:$(VERSION) .

prod-env:
	@test -f $(PROD_ENV_FILE) || { echo "$(PROD_ENV_FILE) not found; copy .env.example and set POSTGRES_PASSWORD and CORS_ALLOWED_ORIGINS" >&2; exit 1; }

# Production stack: build the image for this version and start it in the background.
docker-prod: prod-env
	$(PROD_COMPOSE) up -d --build

docker-prod-down: prod-env
	$(PROD_COMPOSE) down

docker-prod-logs: prod-env
	$(PROD_COMPOSE) logs -f api

docker-prod-ps: prod-env
	$(PROD_COMPOSE) ps
