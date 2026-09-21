# CI/CD Pipeline — What Happens After You Merge

## Who this is for

You know what a "build," a "test," and a Docker image are, and you've merged
a PR before. You don't need prior GitHub Actions or Terraform experience —
this doc explains it as it goes. This doc describes what the pipeline
actually does today, not what was originally planned for it.

## What CI/CD means here, in plain words

- **CI** (Continuous Integration) = every time code changes, a robot checks
  it still builds and passes tests.
- **CD** (Continuous Deployment) = if the code is good, a robot also puts it
  live on the server automatically, so no human has to do it by hand.

Both of those live in one file:
[.github/workflows/core-service-ci.yml](../.github/workflows/core-service-ci.yml).
It has two jobs, one after another: `build-test`, then `deploy`.

## When does this even run?

Only when:
- Something under `apps/core-service/` changes (not the portal, not docs), **and**
- It's either a push to `main`, or a pull request aimed at `main`.

So working on `epic/*` → `dev` doesn't trigger this pipeline at all — that
part of the flow is only watched by the branch-rule check (see
[BRANCH_WORKFLOW.md](./BRANCH_WORKFLOW.md)). This pipeline only cares about
the last step, `dev → main`.

## Job 1: `build-test` — "does the code actually work?"

This job:
1. Starts a temporary real Postgres database inside the CI runner.
2. Runs the database migrations against it.
3. Runs `go build`, `go vet`, and `go test` against your code.

If any of these steps fail, the job stops there — nothing gets deployed.
This job runs on **both** a push to `main` and a PR into `main`, so you get
this feedback on the PR itself, before it's even merged.

## Job 2: `deploy` — "put it on the real server"

This job only runs if:
- `build-test` passed, **and**
- It was an actual `push` to `main` (not just a PR being opened).

In other words: opening a PR from `dev` into `main` only builds and tests it.
The deploy itself only happens once that PR is actually **merged**.

What it does, step by step:

1. **Builds a Docker image** of core-service.
2. **Tags it twice**: once with `latest`, and once with the exact Git commit
   hash (like `a1b2c3d`). Think of the commit-hash tag as a permanent,
   never-changing label for "this exact version of the code."
3. **Pushes both tags** to Docker Hub (an online image storage service).
4. **Connects to the EC2 server over SSH** and tells it to:
   - Point its running setup at the new commit-hash image (not `latest`).
   - Pull that image.
   - Restart the `core-service` container with it.

Here's the actual command it runs on the server:
```bash
cd /opt/core-service
sed -i "s#image: .*core-service-dep.*#image: <your-dockerhub-user>/core-service-dep:<commit-sha>#" docker-compose.yml
docker compose pull core-service
docker compose up -d core-service
```

## Why tag with the commit hash instead of just using `latest`?

If it only ever deployed `:latest`, you'd have no easy way to know exactly
which version of the code is running, or to roll back to a specific older
version. By pinning the server to the exact commit hash each time, you
always know precisely what's live, and — if something breaks — you can
manually point it back at an older commit's image.

## Things this pipeline does NOT do

Worth knowing so you don't assume too much:

- **It doesn't touch nginx or the server's environment variables
  (`app.env`).** Those only change when someone runs `make tf-redeploy` by
  hand from their own computer.
- **It doesn't run new database migrations on the live server.** Migrations
  against production are still a manual step (`make migrate-up`).
- Both jobs use a GitHub "environment" called `production`. If your GitHub
  repo settings have protection rules on that environment (like requiring
  someone to click "approve" before it runs), those rules apply here — but
  that setting lives in GitHub's web settings, not in this repo's files.

## Quick summary

```
Merge dev → main
   ↓
build-test: spin up test DB, migrate, build, vet, test
   ↓ (only if it passed, and only because this was a real merge, not just a PR)
deploy: build Docker image → tag as latest + commit-hash → push to Docker Hub
   ↓
SSH into EC2 → point at new commit-hash image → pull → restart container
   ↓
New code is live. (nginx, env vars, and DB migrations still need a manual step)
```

For anything this pipeline doesn't cover (nginx changes, new env vars,
infra changes), the manual path is still `make deploy` from the root
`Makefile`.
