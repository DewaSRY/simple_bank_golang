


migrate-create: 
	go run ./cmd/migration/main.go create $(name)

migrate-up:
	go run ./cmd/migration/main.go up

sqlc: 
	sqlc generate

.PHONY: migrate-create migrate-up sqlc