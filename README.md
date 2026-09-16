# Simple Bank

A simple core banking application for managing accounts and account-to-account transfers, built with a ledger-based balance model (every transaction writes an immutable entry to a transaction history table).

## Features

- Account management (create, view, update, deactivate)
- Deposits, withdrawals, and transfers between accounts
- Immutable transaction/ledger history per account
- JWT-based authentication

## Structure

This is a monorepo with two apps:

- [`apps/core-service`](apps/core-service) — Go backend (Gin, PostgreSQL, JWT auth, Swagger docs)
- [`apps/portal`](apps/portal) — Next.js frontend (React, Tailwind, React Query)

## Getting started

Spin up Postgres and the core service with Docker Compose:

```bash
docker compose up
```

See each app's own README/docs for local development setup:

- [apps/core-service/docs](apps/core-service/docs)
- [apps/portal/docs](apps/portal/docs)

## Author

- Dewa Surya Ariesta ([dewa.ariesta@cashenable.com](mailto:dewa.ariesta@cashenable.com))
