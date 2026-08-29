test:
	go test -v ./...

server:
	go run ./cmd/server

migrate-create: 
	go run ./cmd/migration/main.go create $(name)

migrate-up:
	go run ./cmd/migration/main.go up

migrate-down:
	go run ./cmd/migration/main.go down

generate-mock:
	go run go.uber.org/mock/mockgen@latest -package mockdb -destination db/mock/querier.go github.com/DewaSRY/core-service/db/sqlc Querier

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

swag-gen:
	swag init -g server/main.go -d cmd,internal -o internal/docs --parseDependency

.PHONY: test server migrate-create migrate-up migrate-down sqlc generate-mock docker-build docker-run docker-stop swag-gen