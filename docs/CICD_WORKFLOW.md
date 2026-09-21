# Development & CI/CD Workflow — As Implemented

## Who this doc is for

Anyone touching this repo's branching, GitHub Actions, or EC2 deploy path —
assumes basic Git and GitHub Actions familiarity, not necessarily Terraform.
This doc is verified against the actual workflow YAML and Makefiles in the
repo today, not the design intent behind them — where the two disagree,
that's called out explicitly rather than smoothed over. It complements
[infra/terraform/docs/ci_cd_implementation.md](../infra/terraform/docs/ci_cd_implementation.md),
which is the *options/design* doc written before any of this existed; this
doc describes what actually got built from it (Option A), and by now
supersedes that doc's "nothing here has been wired up yet" framing.

## Section 1 — Architecture at a Glance

Two independent GitHub Actions workflows, triggered by different events, own
different halves of "getting code live":

| Concern | Owner (file) | Analogy |
|---|---|---|
| Which branch can PR into which | [.github/workflows/pr-branch-rules.yml](../.github/workflows/pr-branch-rules.yml) | A bouncer checking the guest list before you're let near the merge button |
| Build, test, and deploy `core-service` | [.github/workflows/core-service-ci.yml](../.github/workflows/core-service-ci.yml) | An assembly line: verify → package → ship |
| Manual infra-level redeploy (nginx, env vars, rate limits) | root [Makefile](../Makefile) (`make deploy` / `make tf-redeploy`) | The escape hatch for anything the assembly line doesn't cover |

Neither workflow calls the other — `pr-branch-rules.yml` only ever blocks or
allows a PR from opening cleanly; `core-service-ci.yml` doesn't know branch
rules exist. The thing that actually connects them is GitHub's own merge
button: a PR can't be merged if `pr-branch-rules.yml` failed on it, and
merging to `main` is the event that triggers `core-service-ci.yml`'s `deploy`
job.

## Section 2 — The Branch Flow

The intended flow, per the repo owner: `epic/* → dev → main`. Feature work
happens on an `epic/*` branch, that PRs into `dev`, and `dev` PRs into `main`.

`pr-branch-rules.yml` (.github/workflows/pr-branch-rules.yml:1-37) is the only
thing that enforces this, and it fires on `pull_request` events targeting
`main` or `dev` (lines 4-8). Its logic (lines 22-35):

```bash
if [ "$BASE_REF" = "main" ]; then
  if [ "$HEAD_REF" != "dev" ]; then
    echo "::error::Only the 'dev' branch is allowed to open a pull request into 'main'."
    exit 1
  fi
elif [ "$BASE_REF" = "dev" ]; then
  case "$HEAD_REF" in
    epic/*) ;;
    *)
      echo "::error::Only branches matching 'epic/*' are allowed to open a pull request into 'dev'."
      exit 1
      ;;
  esac
fi
```

| Target branch (base) | Allowed source branch (head) | Anything else |
|---|---|---|
| `main` | exactly `dev` | job fails, PR shows a red check |
| `dev` | anything matching `epic/*` | job fails, PR shows a red check |
| anything else | not checked at all | job doesn't even run (trigger is scoped to `branches: [main, dev]`) |

**Worth flagging — this is a check, not a gate.** The job sets a failing
status check; it does not by itself stop the merge button from being
clickable. Whether a red check actually blocks merging depends on branch
protection rules configured in GitHub's repo settings (Settings → Branches),
which live outside this repo's source and aren't visible from the code. If
those protection rules aren't turned on for `main`/`dev`, this workflow is
advisory only — it'll show red, but someone can still merge past it.

**Worth flagging — the design doc's premise is only half true today.**
[ci_cd_implementation.md](../infra/terraform/docs/ci_cd_implementation.md)'s
opening line says the branch flow is "enforced by pr-branch-rules.yml *and
the local pre-commit hook* that blocks direct commits to main/dev." There is
no `.git/hooks` entry, `.husky` directory, or `.pre-commit-config.yaml`
anywhere in this repo as of this writing — no such hook exists. Direct
commits to `main`/`dev` (bypassing PRs entirely) are only as blocked as
GitHub's own branch protection settings make them; nothing in the repo itself
prevents `git push origin main`.

## Section 3 — `core-service-ci.yml`: Build, Test, Deploy

### The problem this piece solves

Every push or PR touching `apps/core-service/**` needs to be built, vetted,
and tested against a real Postgres before it's trusted — and once a change
actually lands on `main`, the running EC2 instance needs to pick it up
without anyone SSHing in by hand.

### Trigger scope

```yaml
on:
  push:
    branches: [main]
    paths:
      - "apps/core-service/**"
      - ".github/workflows/core-service-ci.yml"
  pull_request:
    branches: [main]
    paths:
      - "apps/core-service/**"
      - ".github/workflows/core-service-ci.yml"
```
(core-service-ci.yml:3-13)

Both `push` and `pull_request` are scoped to `branches: [main]` and to the
same `paths` filter. Two consequences worth being explicit about:

1. A PR from `epic/*` into `dev` never triggers this workflow at all — only
   `dev → main` and direct pushes to `main` do. `pr-branch-rules.yml` is what
   watches `epic/* → dev` PRs; `core-service-ci.yml` only watches the last
   hop.
2. Editing anything under `apps/portal/**` or root-level docs never triggers
   a `core-service` build, and editing `apps/core-service/**` never triggers
   anything portal-related — there's no shared/root CI workflow file in this
   repo today.

### Job 1 — `build-test`

Runs a real Postgres 16 service container (lines 24-37), applies migrations
with `golang-migrate` against it (line 53), then `go build`, `go vet`, and
`go test ./...` (lines 55-62) — all from `apps/core-service` as the working
directory (`defaults.run.working-directory`, lines 15-17).

**Worth flagging — `environment: production` is set on the wrong job.**
`build-test` (core-service-ci.yml:22) declares `environment: production`,
which is normally how you'd gate a job behind GitHub's environment
protection rules (required reviewers, wait timers, environment-scoped
secrets). The `deploy` job — the one that actually pushes an image and SSHes
into the real EC2 instance — declares no `environment:` at all
(core-service-ci.yml:64-70). If any protection rules are configured on the
`production` environment in GitHub's settings, they're currently gating the
test suite, not the deploy.

### Job 2 — `deploy`

```yaml
deploy:
  needs: build-test
  if: github.ref == 'refs/heads/main' && github.event_name == 'push'
```
(core-service-ci.yml:64-66)

Gated two ways: it only runs after `build-test` passes, and only on an actual
`push` to `main` — a `pull_request` targeting `main` runs `build-test` but
never `deploy`. Concretely: opening a PR from `dev` into `main` gets you
build/test feedback on the PR; the deploy only happens once that PR is
merged (which is itself a `push` event to `main`).

Steps (core-service-ci.yml:72-124):

| Step | What it does |
|---|---|
| Docker Buildx setup | Standard multi-platform build support |
| Check Docker Hub secrets | Echoes whether `DOCKERHUB_USERNAME`/`DOCKERHUB_TOKEN` are empty — a debug aid, not a hard failure; the job proceeds either way and fails later at login if they're actually missing |
| Log in to Docker Hub | `docker/login-action@v3` |
| Build and push image | `docker buildx build --platform linux/amd64` from `Dockerfile.prod`, tagged **both** `:${{ github.sha }}` and `:latest` |
| Redeploy on EC2 over SSH | `appleboy/ssh-action@v1`; SSHes in and rewrites the compose file's image tag, then `docker compose pull core-service && up -d core-service` |

The SSH step's actual payload:

```bash
cd /opt/core-service
sed -i "s#image: .*core-service-dep.*#image: ${{ secrets.DOCKERHUB_USERNAME }}/core-service-dep:${{ github.sha }}#" docker-compose.yml
docker compose pull core-service
docker compose up -d core-service
```
(core-service-ci.yml:121-124)

**This is a refinement of Option A from the design doc, not a literal copy.**
[ci_cd_implementation.md](../infra/terraform/docs/ci_cd_implementation.md)'s
Option A example pulls `:latest`. The implemented version instead `sed`s the
on-instance `docker-compose.yml` to pin the exact `:${{ github.sha }}` tag
this run built, then pulls that. That's a deliberate improvement the design
doc flags as a possibility but doesn't specify — pinning by SHA means a
deploy is reproducible and the previous SHA's image is still sitting in
Docker Hub if you need to roll back by re-running the `sed` with an older
SHA by hand.

**Rough edge inherited from the design doc, still true:** this only ever
touches the `core-service` container. `app.env` and the `nginx` container on
the instance are untouched by this workflow — those still only change via
`make tf-redeploy` (root Makefile:35-37), which re-renders and re-ships
everything from `infra/terraform`'s `user_data.sh.tpl`
([TERRAFORM_EC2_DEPLOY.md](../infra/terraform/docs/TERRAFORM_EC2_DEPLOY.md)
Section 7). A schema migration is also not run by this workflow — per that
same doc, `make migrate-up` against the DB is still a manual step.

## Section 4 — The Manual Path (Makefile)

For anything the CI `deploy` job doesn't cover — nginx config, rate-limit
tuning, new env vars, a Terraform-level infra change — the root
[Makefile](../Makefile) is still the way to ship:

| Target | What it does |
|---|---|
| `make deploy` | `docker-dep-build` + `docker-dep-push` (in `apps/core-service`), then `tf-redeploy` — the full manual path CI's `deploy` job now automates for ordinary code changes |
| `make tf-redeploy` | Reads `terraform output rendered_user_data` and re-runs it over SSH against the already-running instance — idempotent, safe to rerun (Makefile:35-37) |
| `make tf-plan` / `tf-apply` / `tf-destroy` | Standard Terraform lifecycle against `infra/terraform` |

**Worth flagging — this needs `terraform.tfstate`, which only exists on
whoever's laptop ran `terraform apply` last.** Per
[ci_cd_implementation.md](../infra/terraform/docs/ci_cd_implementation.md)'s
"Why this isn't a two-line GitHub Actions job" section, this is *why* the CI
`deploy` job SSHes in directly with a fixed script instead of running
`make tf-redeploy` itself — GitHub Actions runners have no copy of that state
file, and it's gitignored on purpose (it also holds `db_source` and
`jwt_secret_key` in plaintext). Option C in that doc (remote state in S3 +
DynamoDB) is the way to make `make tf-redeploy` itself CI-runnable, and as of
this writing that option hasn't been picked up.

## Summary — Data Flow

**Feature branch to merged (branch-rule path):**
```
epic/foo → PR into dev → pr-branch-rules.yml checks HEAD_REF matches epic/*
  → (if GitHub branch protection is on) merge allowed → dev
dev → PR into main → pr-branch-rules.yml checks HEAD_REF == dev
  → merge allowed → main
```

**Merge to main to running on EC2 (deploy path):**
```
push to main (the dev→main merge) touching apps/core-service/**
  → core-service-ci.yml: build-test (Postgres service container, migrate, build, vet, test)
  → deploy (needs build-test, only on push+main):
      build & push image as :<sha> and :latest to Docker Hub
      → SSH into EC2, pin docker-compose.yml to :<sha>, pull, up -d core-service
  → nginx / app.env / DB schema unchanged — still require `make tf-redeploy` / `make migrate-up` by hand
```

## Final Reference — Workflow Triggers

| Workflow | Trigger | Branches | Path filter |
|---|---|---|---|
| pr-branch-rules.yml | `pull_request`: opened/edited/synchronize/reopened | base `main` or `dev` | none (repo-wide) |
| core-service-ci.yml (`build-test`) | `push` or `pull_request` | base/target `main` | `apps/core-service/**`, the workflow file itself |
| core-service-ci.yml (`deploy`) | `push` only (needs `build-test`) | `main` | same as above (inherited trigger) |
