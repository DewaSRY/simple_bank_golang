# Simple Bank

A simple core banking application for managing accounts and account-to-account transfers, built with a ledger-based balance model (every transaction writes an immutable entry to a transaction history table).

## Features

- Account management (create, view, update, deactivate)
- Deposits, withdrawals, and transfers between accounts
- Immutable transaction/ledger history per account
- JWT-based authentication

## Structure

This is a monorepo with three apps and shared infra:

- [`apps/core-service`](apps/core-service) — Go backend (Gin, PostgreSQL, JWT auth, Swagger docs)
- [`apps/portal`](apps/portal) — Next.js frontend (React, Tailwind, React Query)
- [`apps/nginx`](apps/nginx) — reverse proxy + per-IP rate limiter in front of core-service, used by local Docker Compose
- [`infra/terraform`](infra/terraform) — Terraform for the EC2 deployment: core-service behind the same kind of nginx reverse proxy, rate-limiting at the edge (`make tf-plan` / `make tf-apply` / `make deploy` from the repo root)

## Getting started

Spin up Postgres, core-service, and an nginx reverse proxy with Docker Compose:

```bash
docker compose up
```

The API is reachable at `http://localhost:8080` — nginx owns that port and proxies to core-service (not published directly), rate-limiting each client IP (`apps/nginx/nginx.conf`) the same way the production EC2 deploy does (`infra/terraform`).

See each app's own README/docs for local development setup:

- [apps/core-service/docs](apps/core-service/docs)
- [apps/portal/docs](apps/portal/docs)
- [infra/terraform/docs](infra/terraform/docs) — deployment/infra deep dives

## Author

- Dewa Surya Ariesta ([sdewa6645@gmail.com](mailto:sdewa6645@gmail.com))
