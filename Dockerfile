# syntax=docker/dockerfile:1

# Keep in step with the go directive in go.mod.
ARG GO_VERSION=1.27.2

FROM golang:${GO_VERSION}-alpine AS base
WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# Development: source is mounted from the host and air rebuilds on every save.
FROM base AS dev
ARG AIR_VERSION=v1.67.4
RUN go install github.com/air-verse/air@${AIR_VERSION}
EXPOSE 8080
CMD ["air", "-c", ".air.toml"]

FROM base AS build
ARG VERSION=dev
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=${VERSION}" \
    -o /out/medpath-server ./cmd/api

# Production: a static binary on distroless, no shell or package manager, running as non-root.
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/medpath-server /medpath-server
USER nonroot:nonroot
ENV APP_ENV=production \
    HTTP_ADDR=:8080
EXPOSE 8080
HEALTHCHECK --interval=15s --timeout=5s --start-period=15s --retries=3 \
    CMD ["/medpath-server", "healthcheck"]
ENTRYPOINT ["/medpath-server"]
