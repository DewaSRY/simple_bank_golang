# Terraform Make Commands — As Implemented

## Who this doc is for

You're assumed to already know *what* the Terraform config in [infra/terraform/](../) builds — if not, read [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) first, including its Terraform primer in Section 0. This doc is narrower: it's the day-to-day command reference for the `tf-*` targets (plus `tf-redeploy` and `deploy`) in the repo-root [Makefile](../../../Makefile) — the ones you actually type to provision, redeploy, or tear down the EC2 instance.

Verified against the root `Makefile` and [apps/core-service/Makefile](../../../apps/core-service/Makefile) as they exist today. Most of these targets are thin — a single `terraform` invocation with no extra scripting — but "thin" is exactly why a couple of them have sharp edges worth knowing before you run them, not after. `tf-redeploy` and `deploy` are the two exceptions: they do real work beyond a bare `terraform` call, and are covered in their own subsection.

## Section 0 — Background Primer: why a `make tf-*` wrapper at all

Every target here does nothing `terraform` itself can't do — it exists purely to save you from having to `cd infra/terraform` first or remember the `-chdir` flag, and to give the multi-step ones (`tf-redeploy`, `deploy`) a name shorter than the pipeline they run.

| Approach | What you type | Must remember | Typical use |
|---|---|---|---|
| Raw Terraform, from repo root | `terraform -chdir=infra/terraform plan` | The exact `-chdir` path, every time | Ad-hoc, one-off commands |
| Raw Terraform, after `cd` | `cd infra/terraform && terraform plan` | To `cd` back afterward, or every other Makefile target (run from repo root) breaks | Interactive exploration |
| **`make tf-plan`** | `make tf-plan` | Nothing — `TF_DIR` is hardcoded once | What this project uses |

**Gotchas that surprise newcomers:**

1. **`tf-apply` and `tf-destroy` still block on Terraform's own interactive `yes` confirmation** — the `make` wrapper doesn't add or remove that prompt, it just runs `terraform apply`/`terraform destroy` as-is. Running either target from a non-interactive context (a CI job, a background shell, a script piping output elsewhere) will hang forever waiting for input that never comes. See [Section 4](#section-4--tf-apply-tf-destroy-tf-redeploy-and-deploy--the-ones-that-touch-a-real-instance-or-ship-a-new-image).
2. **No target depends on any other.** Running `make tf-apply` before ever running `make tf-init` fails with Terraform's own "provider not installed" error, not a friendlier Makefile-level message — `make` has no idea `tf-init` was supposed to come first. Same for `tf-redeploy` before `tf-apply` has ever created an instance: it fails on the `terraform output -raw public_ip` step with no instance to look up. See [Section 2](#section-2--tf-init-tf-fmt-tf-validate--setup-and-hygiene).
3. **None of these targets touch `terraform.tfvars`.** They inherit whatever real secrets and settings are already sitting in `infra/terraform/terraform.tfvars` (gitignored, not part of this doc or the Makefile) — the commands here are only about *running* Terraform, not about supplying it input. See [TERRAFORM_EC2_DEPLOY.md Section 8](TERRAFORM_EC2_DEPLOY.md#section-8--variablestf--input-schema-and-secrets) for where that file comes from.
4. **`deploy` and `tf-redeploy` span two `Makefile`s.** `deploy` calls into [apps/core-service/Makefile](../../../apps/core-service/Makefile) (via `$(MAKE) -C apps/core-service ...`) to build and push the Docker image, then calls back into this same root `Makefile` for `tf-redeploy`. If you only ever ran the old, all-in-one `apps/core-service/Makefile` before this config moved, this cross-file jump is new — see [Section 1](#section-1--architecture-at-a-glance).

## Section 1 — Architecture at a Glance

The composition root is the `tf-*`/`deploy` block itself, the entire repo-root [Makefile](../../../Makefile) (46 lines). It implements no Terraform logic — every `tf-*` target is a one-line delegation to the real `terraform` binary, with `-chdir=$(TF_DIR)` doing the only actual work of pointing it at the right directory. `tf-redeploy` and `deploy` are the two targets that do more than delegate.

```make
CORE_SERVICE_DIR := apps/core-service

TF_DIR := infra/terraform
TF_KEY := $(TF_DIR)/core-service-key.pem

tf-init:
	terraform -chdir=$(TF_DIR) init
```

| Concern | Owner | Analogy |
|---|---|---|
| Directory pointers, defined once | [Makefile:1-4](../../../Makefile#L1-L4) (`CORE_SERVICE_DIR`, `TF_DIR`, `TF_KEY`) | The return address printed once at the top of a form, reused on every page |
| First-time setup | [Makefile:6-7](../../../Makefile#L6-L7) (`tf-init`) | Unpacking the toolbox before the first job |
| Style/hygiene | [Makefile:9-13](../../../Makefile#L9-L13) (`tf-fmt`, `tf-validate`) | Spell-check, run before you print |
| Dry run | [Makefile:15-16](../../../Makefile#L15-L16) (`tf-plan`) | A rehearsal — nothing is built |
| The real thing | [Makefile:18-19](../../../Makefile#L18-L19) (`tf-apply`) | Actually pouring the concrete |
| Reading the result | [Makefile:21-22](../../../Makefile#L21-L22) (`tf-output`) | Checking the nameplate after it's built |
| Teardown | [Makefile:24-25](../../../Makefile#L24-L25) (`tf-destroy`) | Demolition |
| Resyncing a live instance | [Makefile:27-36](../../../Makefile#L27-L36) (`tf-redeploy`) | Handing the current move-in checklist to a super who's already living there |
| Ship a new image + resync | [Makefile:38-43](../../../Makefile#L38-L43) (`deploy`) | Printing a fresh checklist, then handing it over, in one call |

It's split into one target per Terraform subcommand — rather than one target that chains `init && plan && apply` — so that `plan` (safe, read-only) and `apply` (creates real, billable AWS resources) stay two separate, deliberate commands you type. The consequence, per the Section 0 gotcha, is that nothing stops you from typing `make tf-apply` first on a machine that's never run `make tf-init` — the separation buys deliberateness at the cost of not enforcing order. `tf-redeploy` and `deploy` break from that "always separate" pattern on purpose — they're covered in Section 4.

## Section 2 — `tf-init`, `tf-fmt`, `tf-validate` — Setup and Hygiene

**The problem they solve.** Before Terraform can talk to AWS at all, it needs its provider plugins downloaded (`init`). Before you trust a plan, it's worth knowing the `.tf` files are consistently formatted (`fmt`) and internally consistent — right types, no dangling references (`validate`) — neither of which requires any AWS credentials or network calls to AWS itself.

**How they're implemented** ([Makefile:6-13](../../../Makefile#L6-L13)):

```make
tf-init:
	terraform -chdir=$(TF_DIR) init

tf-fmt:
	terraform -chdir=$(TF_DIR) fmt

tf-validate:
	terraform -chdir=$(TF_DIR) validate
```

| Target | What it does | Network/AWS calls | Changes files? |
|---|---|---|---|
| `tf-init` | Downloads the `aws`, `tls`, `local` providers pinned in `infra/terraform/main.tf`; writes/reads `.terraform.lock.hcl` | Yes — hits the Terraform provider registry, not AWS | Yes — creates `infra/terraform/.terraform/` (gitignored) |
| `tf-fmt` | Rewrites `.tf` files in place to canonical formatting (indentation, alignment) | No | Yes, if anything was misformatted |
| `tf-validate` | Type-checks and cross-references the config | No | No |

**When you actually need `tf-init`:** the first time you ever run anything in `infra/terraform/`, and again any time `infra/terraform/main.tf`'s `required_providers` block changes (a new provider, a version bump). Otherwise it's a no-op — safe to run again, it just confirms everything already installed still matches.

**Rough edge worth flagging:** `tf-fmt` silently rewrites files with no confirmation prompt and no diff shown — unlike `tf-plan`/`tf-apply`, there's nothing stopping it from reformatting a file you were mid-edit on. It only touches whitespace/alignment, never values, so this has low practical risk, but it's still a file-mutating command hiding among read-only-sounding ones. (Verified against the config as it stands today: `terraform fmt -diff -check` reports no pending changes.)

## Section 3 — `tf-plan` — The Dry Run

**The problem it solves.** You want to see exactly what Terraform would create/change/destroy against real AWS state, without it actually doing any of that yet.

**How it's implemented** ([Makefile:15-16](../../../Makefile#L15-L16)):

```make
tf-plan:
	terraform -chdir=$(TF_DIR) plan
```

This is a read-only command: it queries AWS for current state (does the security group already exist? what's the latest AMI right now?) and diffs that against `infra/terraform/terraform.tfvars` + the `.tf` files, then prints the result — nothing is created, modified, or destroyed.

**If you're new to Terraform:** running `make tf-plan` twice in a row with nothing changed should print `No changes.` — if it instead shows changes every time (e.g. the AMI lookup resolving to a different ID because Amazon shipped a new image), that's expected drift from the `most_recent = true` AMI data source in `main.tf`, not a bug in this target.

**Rough edge worth flagging:** this target doesn't pass `-out=<file>`, so the plan it shows is *not* saved anywhere. Terraform's own hint at the bottom of the output ("Note: You didn't use the -out option...") applies here — if you run `make tf-plan` and then `make tf-apply` separately, `apply` re-computes its own plan from scratch and could, in theory, differ if AWS state changed in between (e.g. someone deleted the default VPC). For a solo learning deployment this gap is unlikely to bite, but it's why `plan` and `apply` showing different results is possible, not a contradiction.

## Section 4 — `tf-apply`, `tf-destroy`, `tf-redeploy`, and `deploy` — the ones that touch a real instance or ship a new image

**The problem they solve.** `tf-apply` is the command that actually reaches out and creates the EC2 instance, security group, key pair, and local `.pem` file described in [TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md). `tf-destroy` is its inverse — it tears down everything Terraform created. `tf-redeploy` is neither: the instance already exists, and this is how you get it to pick up a config or image change without destroying it. `deploy` is a convenience that chains a full "ship new code" pipeline ending in `tf-redeploy`.

**`tf-apply` / `tf-destroy`** ([Makefile:18-19](../../../Makefile#L18-L19), [24-25](../../../Makefile#L24-L25)):

```make
tf-apply:
	terraform -chdir=$(TF_DIR) apply

tf-destroy:
	terraform -chdir=$(TF_DIR) destroy
```

Both are bare `terraform` calls with **no `-auto-approve`** — this is deliberate, not an oversight. Terraform's own interactive prompt ("Do you want to perform these actions? Enter a value: yes") is the only thing standing between typing `make tf-apply` and real AWS billing starting, or between `make tf-destroy` and the instance (and its `.pem`-protected SSH access) being gone for good. Adding `-auto-approve` to either target would remove that pause; it was intentionally left out.

**Deliberately not chained to `tf-plan`.** Terraform's own `apply` already shows the same plan and asks for confirmation before doing anything, so chaining them here would just show the plan twice.

**Rough edge worth flagging — `tf-apply` does not rebuild or re-run anything on an *existing* instance.** Per [TERRAFORM_EC2_DEPLOY.md Section 7](TERRAFORM_EC2_DEPLOY.md#section-7--user_datash-tpl--the-boot-script), the boot script only runs automatically once. Running `make tf-apply` again after changing `docker_image` (or a nginx/rate-limit variable) in `terraform.tfvars` will not pull the new image or config onto an already-running instance — Terraform will show `0 to change` for the instance itself unless something forces replacement (e.g. changing `instance_type` or `ami`). This used to be a genuine dead end short of destroying and recreating the instance; `tf-redeploy` below is the fix that now exists for it.

**`tf-redeploy`** ([Makefile:27-36](../../../Makefile#L27-L36)):

```make
# Re-runs the exact first-boot script (docker network + core-service + nginx
# setup) against the already-running instance over SSH. EC2 only executes
# user_data automatically on an instance's first boot, so this is how an
# existing instance picks up a new image, a new nginx rate limit, a new CORS
# origin, etc. Idempotent (docker pull + docker rm -f + docker run each
# time) — safe to re-run. See infra/terraform/outputs.tf's rendered_user_data
# and apps/core-service/docs/TERRAFORM_EC2_DEPLOY.md Section 7.
tf-redeploy:
	$(eval EC2_IP := $(shell terraform -chdir=$(TF_DIR) output -raw public_ip))
	terraform -chdir=$(TF_DIR) output -raw rendered_user_data | ssh -i $(TF_KEY) ec2-user@$(EC2_IP) 'sudo bash -s'
```

Two steps: first, `$(eval EC2_IP := $(shell terraform -chdir=$(TF_DIR) output -raw public_ip))` shells out to read the instance's current public IP straight from state (the same value `tf-output` would print, just captured into a make variable instead of printed). Second, `terraform -chdir=$(TF_DIR) output -raw rendered_user_data` prints the *exact* rendered boot script — the same `local.user_data` value `aws_instance.core_service` was given on first boot (see [TERRAFORM_EC2_DEPLOY.md Section 6](TERRAFORM_EC2_DEPLOY.md#section-6--maintf107-156--rendering-the-boot-script-and-the-ec2-instance)) — and pipes it directly into `ssh -i $(TF_KEY) ec2-user@$(EC2_IP) 'sudo bash -s'`.

Three details in that pipeline are load-bearing, not incidental:

- **`-raw`**, not the default `terraform output` formatting. Without it, Terraform would print the value quoted and JSON-escaped (fine for a human to read, not fine to hand straight to `bash`). `rendered_user_data` is also marked `sensitive = true` — `-raw` is also what lets you retrieve the real value at all here; the bare `terraform output`/`make tf-output` command masks sensitive outputs as `(sensitive value)`.
- **`sudo bash -s`**, not a hand-written second script. `bash -s` reads the script from stdin; piping the *actual* `rendered_user_data` output into it means the instance re-runs the identical script EC2 would have run on a fresh first boot — same `docker network create` idempotency, same `docker rm -f ... || true`, same `docker run` invocations. There's no second, hand-maintained "redeploy script" to keep in sync with `user_data.sh.tpl` — see [TERRAFORM_EC2_DEPLOY.md Section 6](TERRAFORM_EC2_DEPLOY.md#section-6--maintf107-156--rendering-the-boot-script-and-the-ec2-instance) for why `local.user_data` was refactored into its own named local specifically to make this possible.
- **It's safe to run repeatedly**, by construction of the script itself, not because of anything in the Makefile — `tf-redeploy` has no idempotency logic of its own; it relies entirely on `user_data.sh.tpl` being written to tolerate re-running (Section 7 of the other doc).

**Rough edge worth flagging — `tf-redeploy` requires an instance that already exists.** It calls `terraform output -raw public_ip`, which fails if `tf-apply` has never successfully created `aws_instance.core_service`. It also requires the same SSH reachability `ssh_command`/`tf-output` depend on — if you've locked `ssh_cidr_blocks` down to your own IP (recommended in [TERRAFORM_EC2_DEPLOY.md Section 5](TERRAFORM_EC2_DEPLOY.md#section-5--maintf78-105--security-group-firewall)) and your IP has since changed, `tf-redeploy` will hang or fail the same way a manual `ssh` would.

**`deploy`** ([Makefile:38-43](../../../Makefile#L38-L43)):

```make
# Builds + pushes the core-service prod image, then redeploys the EC2
# instance against it (and against whatever nginx/rate-limit settings are
# currently in infra/terraform).
deploy:
	$(MAKE) -C $(CORE_SERVICE_DIR) docker-dep-build docker-dep-push
	$(MAKE) tf-redeploy
```

The one target here that reaches into the *other* Makefile: `$(MAKE) -C $(CORE_SERVICE_DIR) docker-dep-build docker-dep-push` runs those two targets inside [apps/core-service/Makefile](../../../apps/core-service/Makefile#L60-L64) (build from `Dockerfile.prod`, push to `sdewa/core-service-dep` on Docker Hub), then `$(MAKE) tf-redeploy` calls back into this same root Makefile to resync the instance. End to end: `make deploy` from the repo root is the full "I changed core-service code, get it running on the instance" pipeline in one command — no manual SSH, no separate redeploy script.

## Section 5 — `tf-output` — Reading the Result

**The problem it solves.** After `tf-apply` (or `tf-redeploy`) finishes, its own terminal output already prints most of what you need once — `tf-output` exists for every time *after* that, when you've closed the terminal and just need the IP, SSH command, or app URL again without re-running `apply`.

**How it's implemented** ([Makefile:21-22](../../../Makefile#L21-L22)):

```make
tf-output:
	terraform -chdir=$(TF_DIR) output
```

Reads straight from the last-known state file (`infra/terraform/terraform.tfstate`) — no AWS calls, no re-planning. Prints all six outputs defined in `infra/terraform/outputs.tf`: `instance_id`, `public_ip`, `app_url`, `app_swagger_url`, `ssh_command`, and `rendered_user_data` — except the last one prints as `<sensitive>` in this default, unadorned form, since it's marked `sensitive = true`. To actually get the raw script (e.g. to inspect what a redeploy would send), you need `terraform -chdir=infra/terraform output -raw rendered_user_data` directly — which is exactly what `tf-redeploy` does internally (Section 4).

**Rough edge worth flagging:** because this reads only the *state file*, it can print a stale IP if the instance was stopped and restarted outside Terraform (see [TERRAFORM_EC2_DEPLOY.md Section 9](TERRAFORM_EC2_DEPLOY.md#section-9--outputstf--result-surface) — no Elastic IP is attached, so a real reboot can change the public IP). `tf-output` won't notice until you run `tf-plan`/`tf-apply` again to refresh state — and neither will `tf-redeploy`, which depends on the same stale value for its SSH target.

## Cross-Feature Coupling

- **`tf-apply` depends on the core-service image already existing on Docker Hub.** The instance's boot script `docker pull`s whatever `docker_image` is set to in `terraform.tfvars` — that image has to already be pushed via `make -C apps/core-service docker-dep-build docker-dep-push` ([apps/core-service/Makefile:60-64](../../../apps/core-service/Makefile#L60-L64)) *before* you run `make tf-apply`, or the new instance's first boot will fail its `docker pull` step with nothing else in this Makefile catching that error. The nginx image doesn't have this dependency — `nginx_image` defaults to the public `nginx:1.27-alpine`, which nothing in this repo builds or pushes.
- **`tf-redeploy` and `deploy` depend on an instance `tf-apply` already created**, and on SSH reachability to it — see Section 4's rough edge above.
- **None of the `tf-*` targets touch AWS credentials.** They rely entirely on whatever `aws configure` set up outside this repo (`~/.aws/credentials`) — there's no target here that checks credentials are present or valid before calling `terraform`; a missing/expired credential surfaces as a raw AWS SDK error from inside `terraform apply` (or, for `tf-redeploy`, from the `terraform output` calls it depends on), not from `make`.
- **`tf-destroy` does not touch Supabase or Docker Hub.** It only removes what `infra/terraform/main.tf` created (the EC2 instance, security group, key pair). Your Supabase database and the pushed Docker image both persist after `make tf-destroy` — worth knowing if you expect a clean slate.

## Adjacent Makefile Targets (Not Part of This Flow)

These targets look related but live in a *different* file — [apps/core-service/Makefile](../../../apps/core-service/Makefile), not the root `Makefile` this doc otherwise covers — and aren't invoked by, or required by, any `tf-*` target directly (only `deploy` reaches into this file, and only for `docker-dep-build`/`docker-dep-push`):

- `docker-build` / `docker-run` / `docker-stop` ([apps/core-service/Makefile:45-54](../../../apps/core-service/Makefile#L45-L54)) — build and run the **local** `core-service:latest` image against `app.env`, unrelated to the EC2 deployment's `sdewa/core-service-dep` image.
- `db-up` / `db-down` / `migrate-*` ([apps/core-service/Makefile:1-34](../../../apps/core-service/Makefile#L1-L34)) — manage the local `docker-compose` Postgres, not the Supabase database `terraform.tfvars` points the EC2 instance at. Running `make migrate-up` against Supabase requires temporarily pointing your local `DB_SOURCE` at the Supabase URL yourself — no target here does that for you.
- `sqlc`, `generate-mock`, `swag-gen` ([apps/core-service/Makefile:36-40](../../../apps/core-service/Makefile#L36-L40), [66-67](../../../apps/core-service/Makefile#L66-L67)) — code generation, entirely unrelated to deployment.

## Summary — Data Flow

**First-time provisioning:**
```
make -C apps/core-service docker-dep-build docker-dep-push   # image must exist on Docker Hub first
make tf-init                                     # once per machine / provider change, from repo root
make tf-plan                                     # optional, but read it
make tf-apply                                    # type "yes" when prompted -> real AWS resources created
make tf-output                                   # get public_ip / ssh_command / app_url (through nginx now)
```

**Later, checking on it:**
```
make tf-output      # reprint outputs from existing state, no AWS calls
make tf-plan        # confirm nothing has drifted
```

**Shipping a new version of the app** (the Section 4 rough edge — `tf-apply` alone won't do this — now has a real target for it):
```
make deploy
# — equivalent by hand to:
make -C apps/core-service docker-dep-build docker-dep-push
make tf-redeploy
```

**Resyncing config only** (new image tag not required — e.g. you only changed a nginx or rate-limit variable in `terraform.tfvars`):
```
make tf-redeploy
```

**Tearing down:**
```
make tf-destroy      # type "yes" when prompted -> instance, security group, key pair all removed
```

## Final Reference — Every Target

| Target | Underlying command | AWS calls | Mutates infra | Idempotent (safe to re-run) |
|---|---|---|---|---|
| `tf-init` | `terraform -chdir=infra/terraform init` | No (provider registry only) | No | Yes |
| `tf-fmt` | `terraform -chdir=infra/terraform fmt` | No | No (rewrites local files only) | Yes |
| `tf-validate` | `terraform -chdir=infra/terraform validate` | No | No | Yes |
| `tf-plan` | `terraform -chdir=infra/terraform plan` | Yes (read-only) | No | Yes |
| `tf-apply` | `terraform -chdir=infra/terraform apply` | Yes | **Yes** — creates/updates real resources | Yes, but see Section 4's "won't redeploy new images" callout |
| `tf-output` | `terraform -chdir=infra/terraform output` | No (reads local state) | No | Yes |
| `tf-destroy` | `terraform -chdir=infra/terraform destroy` | Yes | **Yes** — deletes real resources | Yes (no-op once nothing is left to destroy) |
| `tf-redeploy` | `terraform output -raw public_ip` + `terraform output -raw rendered_user_data \| ssh ... 'sudo bash -s'` | Yes (reads state only, no AWS mutation) | **Yes** — replaces both running containers on the instance | Yes, by construction of `user_data.sh.tpl` |
| `deploy` | `make -C apps/core-service docker-dep-build docker-dep-push` then `make tf-redeploy` | Yes (Docker Hub push + the above) | **Yes** — new image built, pushed, and deployed | Yes |
