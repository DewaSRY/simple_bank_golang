# Build stage
FROM --platform=$BUILDPLATFORM golang:1.25-alpine AS builder
ARG TARGETOS
ARG TARGETARCH
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o main ./cmd/server
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go build -o migration ./cmd/migration
# CLI the migration binary shells out to (see cmd/migration/main.go); needs
# the postgres driver build tag since golang-migrate ships drivers behind tags.
# Pinned to match the golang-migrate image version used by docker-compose.yml's
# local `migrate` service, so the CLI behaves identically in both places.
# `go install` puts the binary in a GOOS_GOARCH subdirectory of $GOPATH/bin
# instead of directly in it whenever TARGETARCH differs from the builder's
# native arch (true cross-compilation, e.g. building linux/amd64 from an
# arm64 host) — and GOBIN can't be set to force a fixed path together with
# cross-compilation ("cannot install cross-compiled binaries when GOBIN is
# set"). find+cp normalizes either outcome to one known path so the COPY
# below works regardless of the host's architecture.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@v4.19.1 \
    && find /go/bin -type f -name migrate -exec cp {} /go/bin/migrate-cli \;

# Run stage
FROM alpine:3.20
# bash: cmd/migration shells out via "bash -c"; ca-certificates: TLS to Postgres/etc.
RUN apk add --no-cache bash ca-certificates

WORKDIR /app
COPY --from=builder /app/main .
COPY --from=builder /app/migration .
COPY --from=builder /go/bin/migrate-cli /usr/local/bin/migrate
COPY --from=builder /app/internal/db/migrations ./internal/db/migrations
COPY docker-entrypoint.sh ./entrypoint.sh
RUN chmod +x ./entrypoint.sh

# Runtime config (DB_DRIVER, DB_SOURCE, SERVER_ADDRESS, JWT_SECRET_KEY,
# JWT_ACCESS_TOKEN_DURATION, CORS_ALLOWED_ORIGINS, LOG_*, RATE_LIMIT_*,
# DEVICE_FINGERPRINT_TRUSTED_IPS) is never baked into the image — it's
# supplied entirely at container start via `docker run --env-file app.env`
# (see Makefile's docker-run target) or compose's `env_file:`. See
# app.env.example for the full list of variables.
EXPOSE 8080
ENTRYPOINT ["./entrypoint.sh"]
