# CI/CD Implementation — Auto-Deploying core-service to EC2

## The problem this doc solves

Your branch flow is `epic/* → dev → main` (enforced by [.github/workflows/pr-branch-rules.yml](../../../.github/workflows/pr-branch-rules.yml) and the local pre-commit hook that blocks direct commits to `main`/`dev`). Once a PR lands on `main`, [core-service-ci.yml](../../../.github/workflows/core-service-ci.yml) builds, vets, and tests it — but nothing after that touches the EC2 instance. Getting a merged change live today means you, personally, running:

```
make deploy   # = make -C apps/core-service docker-dep-build docker-dep-push, then make tf-redeploy
```

from your own machine. This doc is about closing that last step: what it would take for merging to `main` to rebuild the EC2 instance by itself, with no human running a Makefile target.

This is a design/options doc, not an implementation — nothing here has been wired up yet. It ends with a recommended path and the exact files you'd add to build it.

## Why this isn't a two-line GitHub Actions job

The instinctive move is "just call `make tf-redeploy` from CI instead of from your laptop." That target is:

```makefile
tf-redeploy:
	$(eval EC2_IP := $(shell terraform -chdir=$(TF_DIR) output -raw public_ip))
	terraform -chdir=$(TF_DIR) output -raw rendered_user_data | ssh -i $(TF_KEY) ec2-user@$(EC2_IP) 'sudo bash -s'
```

Both `terraform output` calls need `terraform.tfstate` — and per [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) Section 0, that state file lives **only on your machine**, gitignored on purpose (it also holds `db_source` and `jwt_secret_key` in plaintext — see that doc's Section 8). A GitHub Actions runner is a fresh VM with no copy of it. So `terraform output` in CI would fail immediately, not because of a missing secret, but because there's no state to read from.

That one fact is what actually decides which options are realistic:

| Approach | Needs Terraform state in CI? | New moving parts | Matches current `tf-redeploy` exactly? |
|---|---|---|---|
| **A. Direct SSH redeploy job** (recommended) | No | None — reuses secrets you'd add anyway | Functionally yes; skips re-rendering nginx config each time |
| **B. Watchtower on the instance** | No | A third long-running container on EC2 | No — image-only, doesn't touch env vars or nginx |
| **C. Remote Terraform state (S3 + DynamoDB) + CI runs the real `tf-redeploy`** | Yes (that's what this adds) | An S3 bucket + DynamoDB lock table, AWS credentials in GitHub Secrets | Yes, exactly |

## Option A — GitHub Actions builds, pushes, then SSHes in and redeploys (recommended)

This mirrors what `make deploy` already does, just running on `push` to `main` instead of on your laptop, and skipping the Terraform-state dependency by SSHing straight in with a fixed, idempotent script instead of asking Terraform for one.

**Why this first, not C:** you already have a working, idempotent redeploy path (`docker compose pull` → `docker compose up -d` against `/opt/core-service`'s compose stack, safe to rerun — see [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) Section 7). Option A just triggers that same shape of command from CI instead of from your terminal. It adds zero new infrastructure. Option C is the "more correct" long-term answer but is a separate, bigger project (remote state) that's worth doing once you outgrow A, not before.

**What it looks like — a new job appended to `core-service-ci.yml`, gated on the existing `build-test` job passing and on the `main` branch:**

```yaml
  deploy:
    needs: build-test
    if: github.ref == 'refs/heads/main' && github.event_name == 'push'
    runs-on: ubuntu-latest
    defaults:
      run:
        working-directory: apps/core-service

    steps:
      - uses: actions/checkout@v4

      - name: Log in to Docker Hub
        uses: docker/login-action@v3
        with:
          username: ${{ secrets.DOCKERHUB_USERNAME }}
          password: ${{ secrets.DOCKERHUB_TOKEN }}

      - name: Build and push core-service image
        run: |
          docker buildx build --platform linux/amd64 \
            -t ${{ secrets.DOCKERHUB_USERNAME }}/core-service-dep:latest \
            --push -f Dockerfile.prod .

      - name: Redeploy on EC2 over SSH
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.EC2_HOST }}
          username: ec2-user
          key: ${{ secrets.EC2_SSH_PRIVATE_KEY }}
          script: |
            cd /opt/core-service
            docker compose pull core-service
            docker compose up -d core-service
```

This deliberately only touches the `core-service` container — `/opt/core-service/app.env` and the `nginx` container are left alone, since those only change when you edit `terraform.tfvars` or `nginx.conf.tpl`, not on every code push. When you do change those, you'd still run `make tf-redeploy` by hand once (it re-renders and re-pushes everything, including `app.env` and nginx's config) — this CI job only replaces the "I changed Go code and want it live" loop.

**GitHub Secrets you'd need to add** (Settings → Secrets and variables → Actions), none of which exist in the repo today:

| Secret | Value | Where it comes from |
|---|---|---|
| `DOCKERHUB_USERNAME` | Your Docker Hub username | Same account `make docker-dep-push` already pushes to |
| `DOCKERHUB_TOKEN` | A Docker Hub access token (not your password) | Docker Hub → Account Settings → Security → New Access Token |
| `EC2_HOST` | The instance's public IP or a stable hostname | `terraform output -raw public_ip`, or attach an Elastic IP first (see the "public IP is not static" rough edge in [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) Section 9) |
| `EC2_SSH_PRIVATE_KEY` | Contents of `infra/terraform/core-service-key.pem` | Generated by `tls_private_key` in `main.tf` — copy the `.pem` file's contents in as-is |

**Trigger scope** — reuse the same `paths:` filter `core-service-ci.yml` already has (`apps/core-service/**`), so a portal-only or docs-only merge to `main` doesn't rebuild the server for no reason.

**Rough edges worth knowing before you flip this on:**

1. **The EC2 SSH port is open to `0.0.0.0/0` by default** (`ssh_cidr_blocks` in `variables.tf`, flagged in [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) Section 5). Handing GitHub Actions your private key doesn't change that exposure, but it's a good prompt to actually narrow it — GitHub Actions runners use a published, changing IP range, so pinning `ssh_cidr_blocks` to "GitHub only" isn't practical; leaving it open but rotating the key if it ever leaks is the realistic mitigation.
2. **The public IP moves if the instance ever stops/restarts** (Section 9's rough edge) — `EC2_HOST` would go stale. Worth attaching an Elastic IP before relying on this, so `EC2_HOST` doesn't need updating by hand.
3. **This still doesn't run migrations.** Per [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md)'s Cross-Feature Coupling section, migrations are a separate manual step (`make migrate-up` against Supabase) today, and that stays true here — a schema-changing merge to `main` still needs you to run migrations yourself, before or after this job runs.

## Option B — Watchtower (mentioned for completeness, not recommended as the primary mechanism)

[Watchtower](https://containrrr.dev/watchtower/) is a container you'd add to the instance (a fourth block in `user_data.sh.tpl`, alongside `core-service` and `nginx`) that polls Docker Hub on an interval and auto-restarts any container whose image changed. Appeal: **zero SSH access needed from GitHub** — CI only needs to build and push the image (the first two steps of Option A), and Watchtower does the rest entirely from the instance side.

Why not lead with this: it's polling-based (a merge to `main` goes live on Watchtower's next poll, not immediately), it can't do anything beyond "pull new image, restart container" (an `app.env` or nginx change still needs `tf-redeploy`), and it's a permanent extra container that itself needs Docker Hub credentials and restart policy on the instance — a moving part with its own failure mode to learn, for a benefit (no SSH secret in GitHub) that's more about taste than necessity for a single learning instance. Worth revisiting if you later decide you don't want CI holding SSH access at all.

## Option C — Remote Terraform state, so CI can run the exact `tf-redeploy` path

The "no compromises" answer: move `terraform.tfstate` off your laptop into an S3 bucket (with a DynamoDB table for state locking), so any machine — including a GitHub Actions runner — can run `terraform output` / `terraform apply` against the same state. Then the Option A SSH step is replaced by literally running `make tf-redeploy` in CI, unchanged.

This is the more "correct" long-term setup (it's also how you'd eventually support more than one person operating this infra), but it's a separate project: provisioning the S3 bucket + DynamoDB table (themselves either clicked by hand once or bootstrapped by a tiny separate Terraform config), migrating existing state with `terraform init -migrate-state`, and adding AWS credentials as GitHub Secrets (`AWS_ACCESS_KEY_ID`/`AWS_SECRET_ACCESS_KEY`, ideally scoped to a least-privilege IAM user or OIDC role rather than your own). Given the state file's existing secrets problem (Section 0/8 of the deploy doc — `db_source` and `jwt_secret_key` sit in it as plaintext already), moving it to S3 also means locking that bucket down (private, encrypted, versioned) rather than making the exposure worse.

**Recommendation: don't start here.** Come back to Option C once Option A's manual "run `tf-redeploy` by hand when infra config changes" split actually bothers you, or once more than one person needs to run `terraform apply`.

## Recommended path

1. Ship **Option A** first — it's additive (one new CI job + four secrets), doesn't touch `main.tf`/`variables.tf`/`user_data.sh.tpl` at all, and directly closes the gap described at the top of this doc: merge to `main` → image rebuilt, pushed, and running on EC2, with no local `make deploy`.
2. Attach an Elastic IP to `aws_instance.core_service` (a small `main.tf` addition) so `EC2_HOST` never goes stale — do this before or right after Option A, not after.
3. Keep `make tf-redeploy` / `make deploy` as the manual path for anything Option A's fixed SSH script doesn't cover: nginx config changes, rate-limit tuning, new env vars, database migrations.
4. Revisit **Option C** only when the manual step in (3) becomes frequent enough to be worth the extra AWS setup.

## Cross-Feature Coupling

- Depends on [core-service-ci.yml](../../../.github/workflows/core-service-ci.yml) already passing `build-test` — the deploy job in Option A is written to run only `needs: build-test`, so a broken build/test never reaches the redeploy step.
- Depends on the `core-service` service definition in [docker-compose.prod.yaml](../docker-compose.prod.yaml) staying in sync — Option A's SSH step only runs `docker compose pull core-service && docker compose up -d core-service` against whatever `/opt/core-service/docker-compose.yml` already says, so a change to that service (a new volume mount, a new network, a new port) is picked up automatically the next time `make tf-redeploy` re-renders and re-writes the compose file — Option A's fixed two-line script itself never needs editing for that kind of change.
- Doesn't change anything in [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) or [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md) — `make deploy`/`make tf-redeploy` remain valid and are still how you'd push an infra-level change (nginx, rate limits, new env vars).
