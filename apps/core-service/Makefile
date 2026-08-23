test:
	go test -v ./...

server:
	go run ./cmd/server

migrate-create: 
	go run ./cmd/migration/main.go create $(name)

migrate-up:
	go run ./cmd/migration/main.go up

sqlc: 
	sqlc generate




generate-mock:
	go run go.uber.org/mock/mockgen@latest -package mockdb -destination db/mock/querier.go github.com/DewaSRY/core-service/db/sqlc Querier

.PHONY: test server migrate-create migrate-up sqlc generate-mock