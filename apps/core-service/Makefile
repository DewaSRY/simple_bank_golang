db-up:
	docker compose up -d postgres
	docker compose run --rm migrate

db-down:
	docker compose down

test: db-up
	go test -v ./...

test-failed: 
	@go test -v ./... 2>&1 | grep -E "^(--- FAIL|FAIL)" || echo "All tests passed"

test-coverage:
	go test -v -coverprofile=coverage.out -covermode=atomic ./internal/api/...

server:
	go run ./cmd/server

migrate-create: 
	go run ./cmd/migration/main.go create $(name)

migrate-up:
	go run ./cmd/migration/main.go up

migrate-down:
	go run ./cmd/migration/main.go down

migrate-force:
	go run ./cmd/migration/main.go force $(version)

migrate-goto:
	go run ./cmd/migration/main.go goto $(version)

generate-mock:
	go run go.uber.org/mock/mockgen@v0.6.0 -package mockdb -destination internal/db/mock/querier.go github.com/DewaSRY/core-service/internal/db/store Storer

sqlc: 
	sqlc generate

IMAGE_NAME := core-service
CONTAINER_NAME := core-service

# Build-time settings baked into the image as defaults (see the ARG block in
# Dockerfile/Dockerfile.prod) — override any of these via
# `make docker-build LOG_LEVEL=debug`. DB_SOURCE and JWT_SECRET_KEY are
# deliberately excluded: they have no safe default and must only be supplied
# at container-run time (--env-file), never baked into an image layer.
DB_DRIVER ?= postgres
SERVER_ADDRESS ?= 0.0.0.0:8080
JWT_ACCESS_TOKEN_DURATION ?= 15m
CORS_ALLOWED_ORIGINS ?=
LOG_LEVEL ?= info
LOG_FORMAT ?= json
LOG_PRETTY_JSON ?= false
LOG_REQUEST_BODY ?= false
LOG_RESPONSE_BODY ?= false
LOG_MAX_BODY_SIZE ?= 1048576
RATE_LIMIT_ENABLED ?= true
RATE_LIMIT_REQUESTS_PER_SECOND ?= 5
RATE_LIMIT_BURST ?= 20
DEVICE_FINGERPRINT_TRUSTED_IPS ?=

BUILD_ARGS := \
	--build-arg DB_DRIVER=$(DB_DRIVER) \
	--build-arg SERVER_ADDRESS=$(SERVER_ADDRESS) \
	--build-arg JWT_ACCESS_TOKEN_DURATION=$(JWT_ACCESS_TOKEN_DURATION) \
	--build-arg CORS_ALLOWED_ORIGINS=$(CORS_ALLOWED_ORIGINS) \
	--build-arg LOG_LEVEL=$(LOG_LEVEL) \
	--build-arg LOG_FORMAT=$(LOG_FORMAT) \
	--build-arg LOG_PRETTY_JSON=$(LOG_PRETTY_JSON) \
	--build-arg LOG_REQUEST_BODY=$(LOG_REQUEST_BODY) \
	--build-arg LOG_RESPONSE_BODY=$(LOG_RESPONSE_BODY) \
	--build-arg LOG_MAX_BODY_SIZE=$(LOG_MAX_BODY_SIZE) \
	--build-arg RATE_LIMIT_ENABLED=$(RATE_LIMIT_ENABLED) \
	--build-arg RATE_LIMIT_REQUESTS_PER_SECOND=$(RATE_LIMIT_REQUESTS_PER_SECOND) \
	--build-arg RATE_LIMIT_BURST=$(RATE_LIMIT_BURST) \
	--build-arg DEVICE_FINGERPRINT_TRUSTED_IPS=$(DEVICE_FINGERPRINT_TRUSTED_IPS)

docker-build:
	docker build $(BUILD_ARGS) -t $(IMAGE_NAME):latest .

docker-run:
	docker run --name $(CONTAINER_NAME) -p 8080:8080 --env-file app.env $(IMAGE_NAME):latest

docker-stop:
	docker stop $(CONTAINER_NAME)
	docker rm $(CONTAINER_NAME)

DEP_IMAGE_NAME := sdewa/core-service-dep
DEP_PLATFORM := linux/amd64


docker-dep-build:
	docker buildx build --platform $(DEP_PLATFORM) $(BUILD_ARGS) -t $(DEP_IMAGE_NAME):latest --load -f Dockerfile.prod .

docker-dep-push:
	docker push $(DEP_IMAGE_NAME):latest

swag-gen:
	swag init -g server/main.go -d cmd,internal -o internal/docs --parseDependency --parseInternal

# Infra (Terraform, EC2, nginx) lives at the repo root now — see
# ../../Makefile and ../../infra/terraform. `make deploy` there builds and
# pushes this image, then redeploys the EC2 instance against it.

.PHONY: db-up db-down test test-failed test-router-coverage server migrate-create migrate-up migrate-down migrate-force migrate-goto sqlc generate-mock docker-build docker-run docker-stop docker-dep-build docker-dep-push swag-gen