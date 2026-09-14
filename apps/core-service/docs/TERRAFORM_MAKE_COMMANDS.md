# Terraform Make Commands — As Implemented

## Who this doc is for

You're assumed to already know *what* the Terraform config in [terraform/](../terraform/) builds — if not, read [docs/TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md) first, including its Terraform primer in Section 0. This doc is narrower: it's the day-to-day command reference for the `tf-*` targets in [Makefile:62-83](../Makefile#L62-L83), the ones you actually type to provision or tear down the EC2 instance.

Verified against the `Makefile` as it exists today. These targets are thin — each is a single `terraform` invocation with no extra scripting — but "thin" is exactly why a couple of them have sharp edges worth knowing before you run them, not after.

## Section 0 — Background Primer: why a `make tf-*` wrapper at all

Every target here does nothing `terraform` itself can't do — it exists purely to save you from having to `cd terraform` first or remember the `-chdir` flag.

| Approach | What you type | Must remember | Typical use |
|---|---|---|---|
| Raw Terraform, from repo root | `terraform -chdir=terraform plan` | The exact `-chdir` path, every time | Ad-hoc, one-off commands |
| Raw Terraform, after `cd` | `cd terraform && terraform plan` | To `cd` back afterward, or every other Makefile target (run from repo root) breaks | Interactive exploration |
| **`make tf-plan`** | `make tf-plan` | Nothing — `TF_DIR` is hardcoded once | What this project uses |

**Gotchas that surprise newcomers:**

1. **`tf-apply` and `tf-destroy` still block on Terraform's own interactive `yes` confirmation** — the `make` wrapper doesn't add or remove that prompt, it just runs `terraform apply` as-is. Running either target from a non-interactive context (a CI job, a background shell, a script piping output elsewhere) will hang forever waiting for input that never comes. See [Section 3](#section-3--tf-apply-and-tf-destroy--the-two-irreversible-ones).
2. **No target depends on any other.** Running `make tf-apply` before ever running `make tf-init` fails with Terraform's own "provider not installed" error, not a friendlier Makefile-level message — `make` has no idea `tf-init` was supposed to come first. See [Section 2](#section-2--tf-init-tf-fmt-tf-validate--setup-and-hygiene).
3. **None of these targets touch `terraform.tfvars`.** They inherit whatever real secrets and settings are already sitting in `terraform/terraform.tfvars` (gitignored, not part of this doc or the Makefile) — the commands here are only about *running* Terraform, not about supplying it input. See [docs/TERRAFORM_EC2_DEPLOY.md Section 8](TERRAFORM_EC2_DEPLOY.md#section-8--variablestf--input-schema-and-secrets) for where that file comes from.

## Section 1 — Architecture at a Glance

The composition root is the `tf-*` block itself, [Makefile:62-83](../Makefile#L62-L83). It implements no Terraform logic — every target is a one-line delegation to the real `terraform` binary, with `-chdir=$(TF_DIR)` doing the only actual work of pointing it at the right directory.

```make
TF_DIR := terraform

tf-init:
	terraform -chdir=$(TF_DIR) init
```

| Concern | Owner | Analogy |
|---|---|---|
| Directory pointer, defined once | [Makefile:62](../Makefile#L62) (`TF_DIR := terraform`) | The return address printed once at the top of a form, reused on every page |
| First-time setup | [Makefile:64-65](../Makefile#L64-L65) (`tf-init`) | Unpacking the toolbox before the first job |
| Style/hygiene | [Makefile:67-71](../Makefile#L67-L71) (`tf-fmt`, `tf-validate`) | Spell-check, run before you print |
| Dry run | [Makefile:73-74](../Makefile#L73-L74) (`tf-plan`) | A rehearsal — nothing is built |
| The real thing | [Makefile:76-77](../Makefile#L76-L77) (`tf-apply`) | Actually pouring the concrete |
| Reading the result | [Makefile:79-80](../Makefile#L79-L80) (`tf-output`) | Checking the nameplate after it's built |
| Teardown | [Makefile:82-83](../Makefile#L82-L83) (`tf-destroy`) | Demolition |

It's split into one target per Terraform subcommand — rather than one `tf-deploy` target that chains `init && plan && apply` — so that `plan` (safe, read-only) and `apply` (creates real, billable AWS resources) stay two separate, deliberate commands you type. The consequence, per the Section 0 gotcha, is that nothing stops you from typing `make tf-apply` first on a machine that's never run `make tf-init` — the separation buys deliberateness at the cost of not enforcing order.

## Section 2 — `tf-init`, `tf-fmt`, `tf-validate` — Setup and Hygiene

**The problem they solve.** Before Terraform can talk to AWS at all, it needs its provider plugins downloaded (`init`). Before you trust a plan, it's worth knowing the `.tf` files are consistently formatted (`fmt`) and internally consistent — right types, no dangling references (`validate`) — neither of which requires any AWS credentials or network calls to AWS itself.

**How they're implemented** ([Makefile:64-71](../Makefile#L64-L71)):

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
| `tf-init` | Downloads the `aws`, `tls`, `local` providers pinned in `terraform/main.tf`; writes/reads `.terraform.lock.hcl` | Yes — hits the Terraform provider registry, not AWS | Yes — creates `terraform/.terraform/` (gitignored) |
| `tf-fmt` | Rewrites `.tf` files in place to canonical formatting (indentation, alignment) | No | Yes, if anything was misformatted |
| `tf-validate` | Type-checks and cross-references the config | No | No |

**When you actually need `tf-init`:** the first time you ever run anything in `terraform/`, and again any time `terraform/main.tf`'s `required_providers` block changes (a new provider, a version bump). Otherwise it's a no-op — safe to run again, it just confirms everything already installed still matches.

**Rough edge worth flagging:** `tf-fmt` silently rewrites files with no confirmation prompt and no diff shown — unlike `tf-plan`/`tf-apply`, there's nothing stopping it from reformatting a file you were mid-edit on. It only touches whitespace/alignment, never values, so this has low practical risk, but it's still a file-mutating command hiding among read-only-sounding ones.

## Section 3 — `tf-plan` — The Dry Run

**The problem it solves.** You want to see exactly what Terraform would create/change/destroy against real AWS state, without it actually doing any of that yet.

**How it's implemented** ([Makefile:73-74](../Makefile#L73-L74)):

```make
tf-plan:
	terraform -chdir=$(TF_DIR) plan
```

This is a read-only command: it queries AWS for current state (does the security group already exist? what's the latest AMI right now?) and diffs that against `terraform/terraform.tfvars` + the `.tf` files, then prints the result — nothing is created, modified, or destroyed.

**If you're new to Terraform:** running `make tf-plan` twice in a row with nothing changed should print `No changes.` — if it instead shows changes every time (e.g. the AMI lookup resolving to a different ID because Amazon shipped a new image), that's expected drift from the `most_recent = true` AMI data source in `main.tf`, not a bug in this target.

**Rough edge worth flagging:** this target doesn't pass `-out=<file>`, so the plan it shows is *not* saved anywhere. Terraform's own hint at the bottom of the output ("Note: You didn't use the -out option...") applies here — if you run `make tf-plan` and then `make tf-apply` separately, `apply` re-computes its own plan from scratch and could, in theory, differ if AWS state changed in between (e.g. someone deleted the default VPC). For a solo learning deployment this gap is unlikely to bite, but it's why `plan` and `apply` showing different results is possible, not a contradiction.

## Section 4 — `tf-apply` and `tf-destroy` — The Two Irreversible Ones

**The problem they solve.** `tf-apply` is the command that actually reaches out and creates the EC2 instance, security group, key pair, and local `.pem` file described in [docs/TERRAFORM_EC2_DEPLOY.md](TERRAFORM_EC2_DEPLOY.md). `tf-destroy` is its inverse — it tears down everything Terraform created (and only what Terraform created; see the cross-feature section below).

**How they're implemented** ([Makefile:76-77](../Makefile#L76-L77), [82-83](../Makefile#L82-L83)):

```make
tf-apply:
	terraform -chdir=$(TF_DIR) apply

tf-destroy:
	terraform -chdir=$(TF_DIR) destroy
```

Both are bare `terraform` calls with **no `-auto-approve`** — this is deliberate, not an oversight. Terraform's own interactive prompt ("Do you want to perform these actions? Enter a value: yes") is the only thing standing between typing `make tf-apply` and real AWS billing starting, or between `make tf-destroy` and the instance (and its `.pem`-protected SSH access) being gone for good. Adding `-auto-approve` to either target would remove that pause; it was intentionally left out when these targets were written.

**Deliberately not chained to `tf-plan`.** You might expect `tf-apply` to depend on `tf-plan` in the Makefile (`tf-apply: tf-plan`) so you always see the diff first — it doesn't. Terraform's own `apply` already shows the same plan and asks for confirmation before doing anything, so chaining them here would just show the plan twice.

**Rough edge worth flagging — `tf-apply` does not rebuild or re-run anything on an *existing* instance.** Per [docs/TERRAFORM_EC2_DEPLOY.md Section 7](TERRAFORM_EC2_DEPLOY.md#section-7-user_datash-tpl--the-boot-script), the boot script only runs once. Running `make tf-apply` again after changing `docker_image` in `terraform.tfvars` will not pull the new image onto an already-running instance — Terraform will show `0 to change` for the instance itself unless something forces replacement (e.g. changing `instance_type` or `ami`). If you need the running container to pick up a new image, that's a manual SSH step or a deliberate `terraform taint aws_instance.core_service` before the next apply, not something typing `make tf-apply` alone accomplishes.

## Section 5 — `tf-output` — Reading the Result

**The problem it solves.** After `tf-apply` finishes, its own terminal output already prints the outputs once — `tf-output` exists for every time *after* that, when you've closed the terminal and just need the IP or SSH command again without re-running `apply`.

**How it's implemented** ([Makefile:79-80](../Makefile#L79-L80)):

```make
tf-output:
	terraform -chdir=$(TF_DIR) output
```

Reads straight from the last-known state file (`terraform/terraform.tfstate`) — no AWS calls, no re-planning. Prints all four outputs defined in `terraform/outputs.tf`: `instance_id`, `public_ip`, `app_url`, `ssh_command`.

**Rough edge worth flagging:** because this reads only the *state file*, it can print a stale IP if the instance was stopped and restarted outside Terraform (see [docs/TERRAFORM_EC2_DEPLOY.md Section 9](TERRAFORM_EC2_DEPLOY.md#section-9--outputstf--result-surface) — no Elastic IP is attached, so a real reboot can change the public IP). `tf-output` won't notice until you run `tf-plan`/`tf-apply` again to refresh state.

## Cross-Feature Coupling

- **`tf-apply` depends on an image that already exists on Docker Hub.** The instance's boot script `docker pull`s whatever `docker_image` is set to in `terraform.tfvars` — that image has to already be pushed via `make docker-dep-build` and `make docker-dep-push` ([Makefile:53-57](../Makefile#L53-L57)) *before* you run `make tf-apply`, or the new instance's first boot will fail its `docker pull` step with nothing else in this Makefile catching that error.
- **None of the `tf-*` targets touch AWS credentials.** They rely entirely on whatever `aws configure` set up outside this repo (`~/.aws/credentials`) — there's no target here that checks credentials are present or valid before calling `terraform`; a missing/expired credential surfaces as a raw AWS SDK error from inside `terraform apply`, not from `make`.
- **`tf-destroy` does not touch Supabase or Docker Hub.** It only removes what `terraform/main.tf` created (the EC2 instance, security group, key pair). Your Supabase database and the pushed Docker image both persist after `make tf-destroy` — worth knowing if you expect a clean slate.

## Adjacent Makefile Targets (Not Part of This Flow)

The same `Makefile` has other targets that look related but aren't invoked by, or required by, any `tf-*` target:

- `docker-build` / `docker-run` / `docker-stop` ([Makefile:38-49](../Makefile#L38-L49)) — build and run the **local** `core-service:latest` image against `app.env`, unrelated to the EC2 deployment's `sdewa/core-service-dep` image.
- `db-up` / `db-down` / `migrate-*` ([Makefile:1-27](../Makefile#L1-L27)) — manage the local `docker-compose` Postgres, not the Supabase database `terraform.tfvars` points the EC2 instance at. Running `make migrate-up` against Supabase requires temporarily pointing your local `DB_SOURCE` at the Supabase URL yourself — no target here does that for you.
- `sqlc`, `generate-mock`, `swag-gen` ([Makefile:29-33](../Makefile#L29-L33), [59-60](../Makefile#L59-L60)) — code generation, entirely unrelated to deployment.

## Summary — Data Flow

**First-time provisioning:**
```
make docker-dep-build && make docker-dep-push   # image must exist on Docker Hub first
make tf-init                                     # once per machine / provider change
make tf-plan                                     # optional, but read it
make tf-apply                                    # type "yes" when prompted -> real AWS resources created
make tf-output                                   # get public_ip / ssh_command
```

**Later, checking on it:**
```
make tf-output      # reprint IP/SSH command from existing state, no AWS calls
make tf-plan        # confirm nothing has drifted
```

**Shipping a new version of the app** (per the Section 4 rough edge, `tf-apply` alone won't do this):
```
make docker-dep-build && make docker-dep-push
# then either:
ssh -i terraform/core-service-key.pem ec2-user@<public_ip>   # from `make tf-output`
docker pull sdewa/core-service-dep:latest && docker restart core-service
# or: force instance replacement via terraform, then make tf-apply
```

**Tearing down:**
```
make tf-destroy      # type "yes" when prompted -> instance, security group, key pair all removed
```

## Final Reference — Every `tf-*` Target

| Target | Underlying command | AWS calls | Mutates infra | Idempotent (safe to re-run) |
|---|---|---|---|---|
| `tf-init` | `terraform -chdir=terraform init` | No (provider registry only) | No | Yes |
| `tf-fmt` | `terraform -chdir=terraform fmt` | No | No (rewrites local files only) | Yes |
| `tf-validate` | `terraform -chdir=terraform validate` | No | No | Yes |
| `tf-plan` | `terraform -chdir=terraform plan` | Yes (read-only) | No | Yes |
| `tf-apply` | `terraform -chdir=terraform apply` | Yes | **Yes** — creates/updates real resources | Yes, but see Section 4's "won't redeploy new images" callout |
| `tf-output` | `terraform -chdir=terraform output` | No (reads local state) | No | Yes |
| `tf-destroy` | `terraform -chdir=terraform destroy` | Yes | **Yes** — deletes real resources | Yes (no-op once nothing is left to destroy) |
