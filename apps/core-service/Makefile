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

# Runtime config (DB_SOURCE, JWT_SECRET_KEY, ports, log/rate-limit settings,
# etc.) is never baked into the image — it's supplied entirely at container
# start via `--env-file app.env` (docker-run below) or compose's `env_file:`.
# APP_VERSION is the only build-time arg the image accepts (see the ARG
# block in Dockerfile/Dockerfile.prod): pure metadata, never a secret, used
# to tag/label the image for traceability. Override with
# `make docker-dep-build APP_VERSION=v1.2.3`.
APP_VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo dev)

docker-build:
	docker build --build-arg APP_VERSION=$(APP_VERSION) -t $(IMAGE_NAME):latest .

docker-run:
	docker run --name $(CONTAINER_NAME) -p 8080:8080 --env-file app.env $(IMAGE_NAME):latest

docker-stop:
	docker stop $(CONTAINER_NAME)
	docker rm $(CONTAINER_NAME)

DEP_IMAGE_NAME := sdewa/core-service-dep
DEP_PLATFORM := linux/amd64

# Tags both :latest and the immutable :$(APP_VERSION) tag, so a deploy can
# pin/rollback to a specific build instead of only ever following :latest.
docker-dep-build:
	docker buildx build --platform $(DEP_PLATFORM) --build-arg APP_VERSION=$(APP_VERSION) \
		-t $(DEP_IMAGE_NAME):latest -t $(DEP_IMAGE_NAME):$(APP_VERSION) --load -f Dockerfile.prod .

docker-dep-push:
	docker push $(DEP_IMAGE_NAME):latest
	docker push $(DEP_IMAGE_NAME):$(APP_VERSION)

swag-gen:
	swag init -g server/main.go -d cmd,internal -o internal/docs --parseDependency --parseInternal

# Infra (Terraform, EC2, nginx) lives at the repo root now — see
# ../../Makefile and ../../infra/terraform. `make deploy` there builds and
# pushes this image, then redeploys the EC2 instance against it.

.PHONY: db-up db-down test test-failed test-router-coverage server migrate-create migrate-up migrate-down migrate-force migrate-goto sqlc generate-mock docker-build docker-run docker-stop docker-dep-build docker-dep-push swag-gen