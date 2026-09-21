# NEED_TO_IMPROVE.md

Findings from a full read-through of the codebase (2026-09-19), verified against
the actual source (file:line), not against `AGENTS.md`'s description of it.
Items already called out in `AGENTS.md` (no startup config validation, stateless
JWT with no revocation, no role system, CORS wildcard+credentials footgun,
`transferTx`/`deleteAccountTx` missing dedicated unit tests) are **not**
repeated here unless a new, more specific angle was found.

## Critical — fix before next deploy

4. ~~**No `.gitignore` anywhere in the repo.**~~ A root `.gitignore` now
   exists and covers `app.prod.env`, `*.pem`, `*.tfstate*`, `*.tfvars`
   (`!*.tfvars.example`) — verified none of those are tracked. Terraform
   itself also moved, from `terraform/` (inside this app) to `infra/terraform/`
   (repo root) — it now provisions the EC2 host, nginx, and core-service
   together, not core-service alone. `jwt_secret_key`/`db_source` still land
   in `infra/terraform/terraform.tfstate` in plaintext (Terraform's
   `sensitive = true` only redacts terminal output, not the state file
   itself) — that risk is unchanged by the gitignore existing.

5. **Secrets written into EC2 instance user-data in plaintext.**
   `infra/terraform/user_data.sh.tpl` interpolates `DB_SOURCE` and
   `JWT_SECRET_KEY` directly into the instance's user-data script, which AWS
   stores in plaintext instance metadata — readable by anyone with
   `ec2:DescribeInstanceAttribute` or console access to that account. This
   leaks secrets outside of git entirely; fix independently of item 4 (e.g.
   pull secrets from SSM Parameter Store / Secrets Manager at boot instead).

6. **Production secrets baked into the pushed Docker image.**
   `Dockerfile.prod:12` does `COPY app.prod.env ./app.env`, baking the prod
   secrets file into an image layer that's then pushed to Docker Hub as
   `sdewa/core-service-dep:latest` (`Makefile:63-64`). Anyone who pulls that
   image can recover the file via `docker history`/`docker save`, even if a
   later layer "removes" it. Secrets should be injected at container runtime
   (env vars / mounted file / secrets manager), never copied into the image.

## High

7. **SSH open to the world by default.**
   `infra/terraform/variables.tf` default `ssh_cidr_blocks` (and
   `app_cidr_blocks`, which now gates nginx's public port rather than
   core-service's) is `0.0.0.0/0`. `terraform.tfvars.example` even comments
   "lock this down" but nothing enforces it — there's no `validation` block
   like the one guarding `jwt_secret_key`. Add one, or at least flip the
   default to require an explicit override.

8. **No CI/CD pipeline.** No `.github/` or equivalent exists — `go test`,
   `go vet`/lint, and `terraform validate` aren't automatically enforced
   before merge or deploy. Given the security-sensitive Terraform above, this
   is the mechanism that would have caught several of these findings pre-merge.

9. **Login/register anti-enumeration is inconsistent.**
   `internal/api/auth/handlers.go` (`registerUser`, ~L109-114) returns a
   distinct `"email_exists"` 400 when `GetUserByEmail` succeeds, while login
   deliberately returns a uniform "invalid username or password" for both
   "no such user" and "wrong password". The register endpoint currently
   undoes login's own anti-enumeration effort — an attacker enumerates valid
   emails via `/auth/register` instead of `/auth/login`.

10. **No per-route rate limiting on `/auth/login` or `/auth/register`.**
    `core.RateLimitMiddleware` and nginx's `limit_req`
    (`infra/terraform/nginx.conf.tpl`) both apply globally, per client IP,
    across every route — neither singles out the auth endpoints for a
    tighter limit. That caps blunt credential-stuffing/enumeration (item 9)
    at the same rate as any other endpoint, but a distributed attacker
    spreading requests across many IPs, or one staying just under the global
    limit, is still unmitigated for auth specifically.

## Medium

## Low

21. **`docker-dep-build`/`docker-dep-push` only ever produce `:latest`**
    (`Makefile:60-64`) — no version/git-sha tag, so a bad push overwrites the
    only reference with no rollback target, and `terraform` always deploys
    whatever `:latest` currently points to.

22. **Duplicate index.** Migration `000009_account_view_table.up.sql` adds
    `idx_accounts_user_id ON accounts(user_id)`, which duplicates the partial
    index `accounts_user_id_active_idx` already added in
    `000006_account_lifecycle.up.sql` — pure write overhead, no query benefit.
