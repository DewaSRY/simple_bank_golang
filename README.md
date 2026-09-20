# Simple Bank

A simple core banking application for managing accounts and account-to-account transfers, built with a ledger-based balance model (every transaction writes an immutable entry to a transaction history table).

## Features

- Account management (create, view, update, deactivate)
- Deposits, withdrawals, and transfers between accounts
- Immutable transaction/ledger history per account
- JWT-based authentication

## Structure

This is a monorepo with two apps and shared infra:

- [`apps/core-service`](apps/core-service) — Go backend (Gin, PostgreSQL, JWT auth, Swagger docs)
- [`apps/portal`](apps/portal) — Next.js frontend (React, Tailwind, React Query)
- [`infra/terraform`](infra/terraform) — Terraform for the EC2 deployment: core-service behind an nginx reverse proxy that rate-limits at the edge (`make tf-plan` / `make tf-apply` / `make deploy` from the repo root)

## Getting started

Spin up Postgres and the core service with Docker Compose:

```bash
docker compose up
```

See each app's own README/docs for local development setup:

- [apps/core-service/docs](apps/core-service/docs)
- [apps/portal/docs](apps/portal/docs)
- [infra/terraform/docs](infra/terraform/docs) — deployment/infra deep dives

## Author

- Dewa Surya Ariesta ([sdewa6645@gmail.com](mailto:sdewa6645@gmail.com))
