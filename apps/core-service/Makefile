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

docker-build:
	docker build -t $(IMAGE_NAME):latest .


docker-run:
	docker run --name $(CONTAINER_NAME) -p 8080:8080 -v $(CURDIR)/app.env:/app/app.env:ro $(IMAGE_NAME):latest

docker-stop:
	docker stop $(CONTAINER_NAME) 
	docker rm $(CONTAINER_NAME)

DEP_IMAGE_NAME := sdewa/core-service-dep
DEP_PLATFORM := linux/amd64


docker-dep-build:
	docker buildx build --platform $(DEP_PLATFORM) -t $(DEP_IMAGE_NAME):latest --load -f Dockerfile.prod .

docker-dep-push:
	docker push $(DEP_IMAGE_NAME):latest

swag-gen:
	swag init -g server/main.go -d cmd,internal -o internal/docs --parseDependency --parseInternal

TF_DIR := terraform

tf-init:
	terraform -chdir=$(TF_DIR) init

tf-fmt:
	terraform -chdir=$(TF_DIR) fmt

tf-validate:
	terraform -chdir=$(TF_DIR) validate

tf-plan:
	terraform -chdir=$(TF_DIR) plan

tf-apply:
	terraform -chdir=$(TF_DIR) apply

tf-output:
	terraform -chdir=$(TF_DIR) output

tf-destroy:
	terraform -chdir=$(TF_DIR) destroy

.PHONY: db-up db-down test test-failed test-router-coverage server migrate-create migrate-up migrate-down migrate-force migrate-goto sqlc generate-mock docker-build docker-run docker-stop docker-dep-build docker-dep-push swag-gen tf-init tf-fmt tf-validate tf-plan tf-apply tf-output tf-destroy