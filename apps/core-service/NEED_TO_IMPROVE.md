# NEED_TO_IMPROVE.md

Findings from a full read-through of the codebase (2026-09-19), verified against
the actual source (file:line), not against `AGENTS.md`'s description of it.
Items already called out in `AGENTS.md` (no startup config validation, stateless
JWT with no revocation, no role system, CORS wildcard+credentials footgun,
`transferTx`/`deleteAccountTx` missing dedicated unit tests) are **not**
repeated here unless a new, more specific angle was found.

## Critical — fix before next deploy

1. **IDOR: any user can read any other user's ledger entries.**
   `internal/api/transfer/handlers.go:137` (`listAccountEntriesByAccountId`)
   never calls `h.requireOwnedAccount(ctx, params.ID)`, unlike every sibling
   handler in the same file (`deposit` L90, `listRecentTransferDestinations`
   L213, `listAccountTransactionHistory` L269). `GET /accounts/{id}/entries`
   with any account ID returns that account's full entry history regardless
   of who owns it. Add the same `requireOwnedAccount` guard used elsewhere in
   this file.

2. **Excessive data exposure on account search.**
   `internal/api/account/handlers.go:143` (`searchAccountByNumber`) returns
   `accountuserResponse` (`internal/api/account/response.go:33-44` —
   `Balance`, `Description`, `Username`, `UserID`) for *any* matching account,
   not just the caller's own. This endpoint exists so a user can pick a
   transfer destination; it should return the minimal public shape already
   used by `listRecentTransferDestinations` (id/number/name), not another
   user's live balance and username.

3. **`deleteAccountTx` doesn't enforce ownership itself.**
   `internal/db/store/store_account.go:37-51` fetches the account by
   `arg.AccountID` and never compares `account.UserID` to `arg.UserID` — the
   check exists only in the caller (`internal/api/account/manage.go`).
   Business-rule invariants like this belong inside the `*Tx` function per
   this repo's own design (see `AGENTS.md`'s "business logic lives in
   `store`" rule); as written, any future direct caller (an admin tool, a
   batch job, or a store-level test written against the mocked `Querier`)
   can sweep and soft-delete another user's account.

4. **No `.gitignore` anywhere in the repo.**
   `app.prod.env`, `core-service-key.pem`, `terraform/terraform.tfstate*`,
   `terraform/terraform.tfvars`, and `coverage.out` are untracked only by
   discipline today (verified: none are currently tracked). A single
   `git add .` / `git add -A` would commit live DB credentials, the JWT
   secret, the SSH private key, and full Terraform state (which itself holds
   `jwt_secret_key`/`db_source` in plaintext, `terraform/main.tf:112-120`).
   Add a `.gitignore` now, independent of anything else on this list.

5. **Secrets written into EC2 instance user-data in plaintext.**
   `terraform/user_data.sh.tpl:14-21` interpolates `DB_SOURCE` and
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
   `terraform/variables.tf` default `ssh_cidr_blocks` (and `app_cidr_blocks`)
   is `0.0.0.0/0` (`terraform/main.tf:79-93`). `terraform.tfvars.example:14`
   even comments "lock this down" but nothing enforces it — there's no
   `validation` block like the one guarding `jwt_secret_key`. Add one, or at
   least flip the default to require an explicit override.

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

10. **No rate limiting on `/auth/login` or `/auth/register`.** Nothing in the
    middleware chain (`internal/api/server.go:68-73`) or `router.go` throttles
    repeated auth attempts — unlimited credential-stuffing and the
    enumeration in item 9 above are both unmitigated.

## Medium

11. **Query strings aren't redacted in logs.**
    `internal/api/core/logging_middleware.go:107` logs
    `ctx.Request.URL.Query()` verbatim. The existing redaction system covers
    headers and JSON bodies but not query params — if any current or future
    endpoint ever accepts a token/reset-code/API key as a query param, it
    lands in logs unredacted despite the redaction infra existing.

12. **Swagger docs are stale.** `GET /accounts/{id}/transactions`
    (`internal/api/transfer/transfer.go:22`, fully annotated in
    `transactions_handlers.go:238-255`) is missing from the generated
    `internal/docs`/`swagger.json`. Run `make swag-gen` and diff — this is
    the kind of drift `AGENTS.md` already warns to check for after godoc
    changes, and it's currently out of sync.

13. **Money negation via string concatenation, not `decimal`.**
    `internal/db/store/store_transaction.go:99` and
    `internal/db/store/store_account.go:89` build the negative amount as
    `"-" + arg.Amount` / `"-" + lockedAccount.Balance` instead of
    `amount.Neg().String()`. `decimal.NewFromString` accepts a leading `+`,
    so an input like `"+5.00"` would concatenate into `"-+5.00"` — not a
    valid Postgres numeric literal, failing deep inside an open transaction.
    Unreachable today only because handlers pre-sanitize via
    `decimal.StringFixed(2)`; fix it in the `*Tx` layer itself so it stays
    safe for any future/direct/test caller, matching this repo's own
    "money via decimal, not string surgery" intent.

14. **`IncrementAccountBalance` has no `deleted_at IS NULL` guard.**
    `internal/db/query/account-transaction.sql` — every other
    account-mutating query filters soft-deleted rows; this one doesn't. It's
    only safe today because every caller pre-locks the row through a query
    that does filter (`GetAccountByIdForUpdate`). A future direct call
    would silently credit/debit a soft-deleted account.

15. **Ledger table has no composite index for its own hot query.**
    `entries` (migration `000001_init-transaction-feature`) has only a plain
    index on `account_id`, but `ListAccountTransactionHistory`
    (`internal/db/query/entries.sql`) filters `account_id` + a `created_at`
    range and orders by `created_at DESC, id DESC`. Add a composite
    `(account_id, created_at, id)` index before this table grows — right now
    every history page requires a sort step.

16. **Account search `LIKE` pattern doesn't escape wildcards.**
    `ListAccountsSearchByUserNumber` / `ListMeAccountsByUserId`
    (`internal/db/query/account.sql`) build
    `LIKE '%' || sqlc.arg(...) || '%'` without escaping `%`/`_` in the user
    -supplied term, so a search containing those characters returns
    incorrect matches — and the leading wildcard already rules out index use
    regardless.

17. **JWT payload has no `jti`.** `internal/token/payload.go` — adding a
    `jti` now costs nothing and is a prerequisite for ever adding
    server-side revocation later (today there's no way to name a specific
    token even if a blacklist mechanism were added).

18. **Docker images run as root.** Neither `Dockerfile` nor `Dockerfile.prod`
    sets a `USER` in the final stage — unnecessarily widens the blast radius
    of any RCE in the service.

19. **`make db-up`/`make test` invoke a `docker compose` setup that doesn't
    exist in the repo.** `Makefile:2-3` runs
    `docker compose up -d postgres` / `docker compose run --rm migrate`, but
    there is no `docker-compose.yml` anywhere in the repo. Either it was
    removed and needs restoring, or these targets are currently broken for
    anyone following `AGENTS.md`'s own documented workflow.

## Low

20. **Floating base image tags.** `Dockerfile:2` / `Dockerfile.prod:2` pin
    `golang:1.25-alpine` / `alpine:3.20` by tag, not digest — a rebuild on a
    different day can silently pull a different base image.

21. **`docker-dep-build`/`docker-dep-push` only ever produce `:latest`**
    (`Makefile:60-64`) — no version/git-sha tag, so a bad push overwrites the
    only reference with no rollback target, and `terraform` always deploys
    whatever `:latest` currently points to.

22. **`cmd/migration/main.go` uses `fmt.Println` throughout** instead of the
    shared `slog` logger used everywhere else — migration failures aren't
    captured in structured logs/log aggregation like the rest of the service.

23. **Duplicate index.** Migration `000009_account_view_table.up.sql` adds
    `idx_accounts_user_id ON accounts(user_id)`, which duplicates the partial
    index `accounts_user_id_active_idx` already added in
    `000006_account_lifecycle.up.sql` — pure write overhead, no query benefit.

24. **`account_entries` view has no `deleted_at` filter**
    (migration `000010_create-account_entries_table_view.up.sql`), unlike
    `account_user_details_view` — `ListAccountEntriesByAccountId` can return
    full history for a soft-deleted account with no way to tell from the row
    that the account is gone.

25. **`make generate-mock` uses an unpinned `mockgen@latest`**
    (`Makefile:36-37`) — non-reproducible mock generation across machines/CI.

26. **`execTx` sets no explicit isolation level or statement/lock timeout**
    (`internal/db/store/store.go:33-49`). Correctness currently relies
    entirely on the manual ascending-ID `FOR UPDATE` ordering; a stalled
    transaction elsewhere has no server-side timeout guard, only caller
    `ctx` cancellation.

27. **`terraform/outputs.tf` exposes `app_url`/`app_swagger_url` as
    `http://`** — API traffic, including JWT bearer tokens, travels
    unencrypted to the EC2 public IP. Worth a TLS story (ALB + ACM, or
    Caddy/nginx with Let's Encrypt on the instance) before real user data
    goes through this.

28. **`sqlc.yaml` uses the deprecated v1 `packages:` config format** instead
    of the current `version: "2"` / `sql:` block format — still works, but
    will eventually lose support.

29. **Possibly-abandoned transitive dependencies** worth auditing:
    `github.com/quic-go/quic-go` and `github.com/PuerkitoBio/purell` (pulled
    in transitively, likely via `swag`/viper) — not necessarily exploitable,
    but worth confirming they're actually needed.

## Not flagged (checked, found sound)

- `internal/db/mock/querier.go` matches the current `Storer`/`db.Querier`
  interface exactly — no generated-file drift.
- bcrypt cost factor, JWT algorithm-confusion guard (`WithValidMethods`),
  minimum secret length, and expiry checking all look correct as implemented.
- No git-tracked secrets today (`app.prod.env`, `*.pem`, `.tfstate` are all
  currently untracked) — item 4 above is about how fragile that is, not a
  claim that a secret is already committed.
