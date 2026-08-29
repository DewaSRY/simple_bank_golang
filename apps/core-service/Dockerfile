# Build stage
FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o main ./cmd/server

# Run stage
FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/main .
COPY db/migrations ./db/migrations

# app.env holds secrets (JWT key, DB source) so it is not baked into the image;
# mount it at runtime, e.g. `docker run -v $(pwd)/app.env:/app/app.env ...`
EXPOSE 8080
ENTRYPOINT ["/app/main"]
