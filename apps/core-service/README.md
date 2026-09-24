# core-service

A double-entry ledger / simple banking HTTP API. Users register, own one or
more accounts, and transfer money between accounts with full entry (ledger)
history — built as a learning project for a clean, layered Go service.

For a deep dive into the architecture, module layout, and conventions (aimed
at contributors and AI agents), see [AGENTS.md](AGENTS.md). This README is the
quick-start / orientation version.

## Features

- User registration, login, and profile (JWT-based auth)
- Multiple accounts per user, one designated `IsMain`
- Deposits and cross-account transfers with full entry history
- Account search, listing (paginated), update, and soft delete
- Soft-deleting a non-main account sweeps its balance to the main account first
- Structured JSON responses, request-ID tracing, structured logging (`slog`)
- Swagger/OpenAPI docs generated from handler annotations

## Tech Stack

| Concern          | Technology                                                                                            |
| ---------------- | ----------------------------------------------------------------------------------------------------- |
| Language         | Go 1.25                                                                                               |
| HTTP framework   | [Gin](https://github.com/gin-gonic/gin)                                                               |
| Database         | PostgreSQL (`lib/pq`)                                                                                 |
| Query layer      | [sqlc](https://sqlc.dev) (generated, no ORM)                                                          |
| Migrations       | [golang-migrate](https://github.com/golang-migrate/migrate)                                           |
| Money type       | `string`, parsed with [`shopspring/decimal`](https://github.com/shopspring/decimal)                   |
| Auth             | JWT (`golang-jwt/jwt/v5`), stateless bearer tokens                                                    |
| Config           | [`spf13/viper`](https://github.com/spf13/viper)                                                       |
| Logging          | stdlib `log/slog`                                                                                     |
| Validation       | `go-playground/validator/v10`                                                                         |
| Testing          | [`go.uber.org/mock`](https://github.com/uber-go/mock) (unit), real Postgres + `testify` (integration) |
| API docs         | `swaggo/swag` + `gin-swagger`                                                                         |
| Containerization | Docker                                                                                                |
| Infra            | Terraform (single EC2 instance)                                                                       |

## Architecture

Three layers, each only talking to the layer directly below:

```
internal/api/<domain>   HTTP handlers (Gin) — thin translators between HTTP and Storer
internal/db/store       Business logic + multi-step transactions
internal/db/sqlc        Generated, one method per SQL query — no business logic
```

```
cmd/server/            entry point: config, DB, logger, api.NewServer, starts Gin
cmd/migration/         standalone CLI wrapping golang-migrate
internal/api/
  core/                shared kernel: AppError, response envelope, middleware
  auth/                login, register, profile
  account/             create/list/search/get/update/delete account
  transfer/            deposit, cross-account transfer, entries, transaction history
internal/db/
  sqlc/                generated Querier + types (DO NOT hand-edit)
  store/               Storer implementation + *Tx business logic
  query/*.sql          hand-written SQL, source of truth for sqlc
  migrations/          golang-migrate up/down SQL pairs
  mock/querier.go      generated MockStorer (DO NOT hand-edit)
internal/token/        JWT Maker + Payload
internal/config/       viper-based Config + LoadConfig
internal/logger/       slog.Logger construction
internal/domain/constant/  shared enums (e.g. entry types)
```

Full rationale for this layering, including the seams that make store/handler
logic unit-testable without a database, is in
[AGENTS.md](AGENTS.md#what-the-architecture-looks-like).

## Getting Started

### Prerequisites

- Go 1.25+
- Docker (for local Postgres via `docker-compose`)

### Setup

```bash
cp app.env.example app.env
# fill in DB_DRIVER, DB_SOURCE, JWT_SECRET_KEY, SERVER_ADDRESS, etc.
```

See [docs/CONFIG_ENV_VARIABLE.md](docs/CONFIG_ENV_VARIABLE.md) for what each
variable does and the config-loading precedence rules. There is no startup
validation of required fields — a missing/empty value can fail deep inside a
library call rather than up front.

### Run

```bash
make db-up     # start Postgres via docker-compose and apply pending migrations
make server    # go run ./cmd/server
```

The API is served under `/api/v1`, with `/health` and `/swagger/*any` (Swagger
UI) at the root.

### Test

```bash
make test              # db-up, then go test -v ./... (unit + integration)
make test-failed        # full suite, only prints failures
make test-coverage      # coverage for internal/api/..., written to coverage.out
go test ./internal/db/store/...       # one package
```

## API Overview

All routes are under `/api/v1`. Routes marked 🔒 require
`Authorization: Bearer <token>`.

| Method    | Path                                | Description                                         |
| --------- | ----------------------------------- | --------------------------------------------------- |
| POST      | `/auth/register`                    | Register a new user                                 |
| POST      | `/auth/login`                       | Log in, receive an access token                     |
| GET 🔒    | `/auth/profile`                     | Get the current user's profile                      |
| POST 🔒   | `/accounts`                         | Create an account                                   |
| GET 🔒    | `/accounts/me`                      | List the current user's accounts                    |
| GET 🔒    | `/accounts/search-by-number`        | Look up an account by account number                |
| GET 🔒    | `/accounts/:id`                     | Get account detail                                  |
| PUT 🔒    | `/accounts/:id`                     | Update an account                                   |
| DELETE 🔒 | `/accounts/:id`                     | Soft-delete an account (sweeps balance if non-main) |
| POST 🔒   | `/accounts/:id/deposit`             | Deposit into an account                             |
| GET 🔒    | `/accounts/:id/entries`             | List ledger entries for an account                  |
| GET 🔒    | `/accounts/:id/recent-destinations` | Recent transfer destinations                        |
| GET 🔒    | `/accounts/:id/transactions`        | Transaction history for an account                  |
| POST 🔒   | `/transactions/transfer`            | Transfer between two accounts                       |
| GET       | `/health`                           | Health check                                        |
| GET       | `/swagger/*any`                     | Swagger UI                                          |

Every success response is `{"data": ..., "message": "...", "meta": ...}`
(`meta` omitted when absent); every error is
`{"error": {"code", "message", "details"}}`. Regenerate the Swagger spec with
`make swag-gen` after editing handler godoc annotations.

Manual `.http` request examples for exercising the API live in
[rest-testing/core-service.http](rest-testing/core-service.http).

## Common Make Targets

| Target                                                       | What it does                                                 |
| ------------------------------------------------------------ | ------------------------------------------------------------ |
| `make db-up` / `make db-down`                                | Start/stop local Postgres, apply migrations                  |
| `make server`                                                | Run the API server                                           |
| `make test` / `make test-failed` / `make test-coverage`      | Run tests                                                    |
| `make migrate-create name=...`                               | Create a new migration pair                                  |
| `make migrate-up` / `make migrate-down`                      | Apply / roll back migrations                                 |
| `make sqlc`                                                  | Regenerate `internal/db/sqlc` from `internal/db/query/*.sql` |
| `make generate-mock`                                         | Regenerate `internal/db/mock/querier.go` from `Storer`       |
| `make swag-gen`                                              | Regenerate Swagger docs from handler annotations             |
| `make docker-build` / `make docker-run` / `make docker-stop` | Local Docker image lifecycle                                 |
| `make tf-plan` / `make tf-apply`                             | Terraform plan/apply for the EC2 deployment                  |

`make sqlc` and `make generate-mock` are effectively one step — always run
`sqlc` first, `generate-mock` second, since the mock is generated from the
interface sqlc's output feeds into.

## Contributing / Extending

See [AGENTS.md](AGENTS.md#how-to-implement-a-new-feature) for the full
step-by-step for adding a feature (migration → query → mock → business logic
→ handler → error mapping → response DTO → Swagger → tests), plus the
project's hard constraints (money must stay `string`, account-lock ordering,
no `AllowOrigins: []string{"*"}`, etc.).

## Further Reading

Design-decision write-ups per subsystem live in [docs/](docs/):

- [CONFIG_ENV_VARIABLE.md](docs/CONFIG_ENV_VARIABLE.md) — config loading & precedence
- [CORS.md](docs/CORS.md) — CORS setup and why `*` doesn't work here
- [GIN_HTTP_LAYER.md](docs/GIN_HTTP_LAYER.md) — HTTP layer design
- [NORMALIZE_RESPONSE.md](docs/NORMALIZE_RESPONSE.md) — response envelope design
- [SQLC_SETUP.md](docs/SQLC_SETUP.md) / [SQLC_ROW_MAPPING.md](docs/SQLC_ROW_MAPPING.md) — sqlc setup & row mapping
- [MIGRATION_GUID.md](docs/MIGRATION_GUID.md) — migrations
- [GOMOCK_TESTING.md](docs/GOMOCK_TESTING.md) — testing strategy
- [LOGGING.md](docs/LOGGING.md) — logging setup
- [IMPLEMENT_AUTH.md](docs/IMPLEMENT_AUTH.md) — auth design
- [TERRAFORM_EC2_DEPLOY.md](docs/TERRAFORM_EC2_DEPLOY.md) / [TERRAFORM_MAKE_COMMANDS.md](docs/TERRAFORM_MAKE_COMMANDS.md) — deployment
- [VSCODE_DEBUGGING.md](docs/VSCODE_DEBUGGING.md) — local debugging setup

> **Note on doc drift:** this repo went through a per-domain refactor of
> `internal/api`. Some `docs/*.md` files predate it and reference stale file
> paths or function names. Their _design rationale_ is still reliable; verify
> file/function names against the current code. See
> [AGENTS.md's "Doc drift" section](AGENTS.md#doc-drift--read-this-before-trusting-claudemd-or-docsmd-blindly)
> for specifics.

## Security Note

`app.env`, `app.prod.env`, and `core-service-key.pem` in the repo root contain
or can contain secrets. Never commit real secrets in `app.env`, and don't add
new plaintext secret files alongside them.

## Copilot / Agent Instructions

Task-specific Copilot instructions (auto-discovered by description, no need
to attach them manually) live at the monorepo root under
[`.github/instructions/`](../../.github/instructions/):

- [core-service.instructions.md](../../.github/instructions/core-service.instructions.md) — always-relevant fast-lookup rules for this app (layering, hard constraints, error handling, commands)
- [core-service-new-model.instructions.md](../../.github/instructions/core-service-new-model.instructions.md) — creating a new DB entity: migration → query → sqlc → mock → store `*Tx`
- [core-service-new-module.instructions.md](../../.github/instructions/core-service-new-module.instructions.md) — creating a new HTTP API domain package: Handler, routes, DTOs, error mapping, Swagger, tests
