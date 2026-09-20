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

Copy the example env file and fill in your own `JWT_SECRET_KEY` (any local value is fine — never reuse a production secret):

```bash
cp .env.example .env
```

Spin up Postgres, core-service, and an nginx reverse proxy with Docker Compose:

```bash
docker compose up
```

The API is reachable at `http://localhost:8080` — nginx owns that port and proxies to core-service (not published directly), rate-limiting each client IP (`apps/nginx/nginx.conf`) the same way the production EC2 deploy does (`infra/terraform`).

See each app's own README/docs for local development setup:

- [apps/core-service/docs](apps/core-service/docs)
- [apps/portal/docs](apps/portal/docs)
- [infra/terraform/docs](infra/terraform/docs) — deployment/infra deep dives

## Deployment

**Build-time vs. runtime.** `apps/core-service/Dockerfile.prod` only accepts one build arg, `APP_VERSION` (pure metadata, baked in as an OCI label — never a secret). Every other setting (`DB_SOURCE`, `JWT_SECRET_KEY`, `SERVER_ADDRESS`, `CORS_ALLOWED_ORIGINS`, `LOG_*`, `RATE_LIMIT_*`, `DEVICE_FINGERPRINT_TRUSTED_IPS` — full list in [apps/core-service/app.env.example](apps/core-service/app.env.example)) is supplied at container start via `env_file`/`--env-file`, never baked into the image.

**Flow:** `git push` to `main` → [core-service-ci.yml](.github/workflows/core-service-ci.yml) builds `Dockerfile.prod`, pushes `sdewa/core-service-dep:<git-sha>` and `:latest` to Docker Hub, then SSHes into the EC2 instance, pins `docker-compose.yml`'s `core-service` image to that exact `<git-sha>` tag, and runs `docker compose pull core-service && docker compose up -d core-service`.

**GitHub Actions secrets** (Settings → Secrets and variables → Actions): `DOCKERHUB_USERNAME`, `DOCKERHUB_TOKEN`, `EC2_HOST`, `EC2_SSH_PRIVATE_KEY` — see [infra/terraform/docs/ci_cd_implementation.md](infra/terraform/docs/ci_cd_implementation.md) for where each comes from.

**EC2 `.env`.** There's no manual `.env` step on a fresh instance — Terraform's `user_data.sh.tpl` writes `/opt/core-service/app.env` (`chmod 600`) from `infra/terraform/terraform.tfvars` on first boot:

```bash
cd infra/terraform
cp terraform.tfvars.example terraform.tfvars   # fill in db_source, jwt_secret_key, etc.
cd ../..
make tf-init
make tf-apply
```

Changing `app.env`'s contents (a new CORS origin, a rate-limit tweak) means editing `terraform.tfvars` and running `make tf-redeploy` — the CI flow above only ever touches the `core-service` image tag, by design (see [infra/terraform/docs/TERRAFORM_EC2_DEPLOY.md](infra/terraform/docs/TERRAFORM_EC2_DEPLOY.md)'s "Cross-Feature Coupling" section).

**⚠️ Known issue:** a real-looking `JWT_SECRET_KEY` was committed in plaintext in an earlier version of this repo's root `docker-compose.yaml` (git history, not the current file). If that value was ever used in a real deployment, rotate it in `terraform.tfvars` and redeploy — a value that's been in git history should be treated as compromised even after being removed from the working tree.

## Author

- Dewa Surya Ariesta ([sdewa6645@gmail.com](mailto:sdewa6645@gmail.com))
