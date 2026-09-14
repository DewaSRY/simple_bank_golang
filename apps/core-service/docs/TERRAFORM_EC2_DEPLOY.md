# Terraform EC2 Deployment — As Implemented

## Who this doc is for

You're assumed to be comfortable with Go, Docker, and this project's Makefile — but you've said you haven't used Terraform before, so this doc treats it as unfamiliar. If you already know Terraform, skip [Section 0](#section-0--background-primer-what-terraform-actually-is) — it's a primer, not load-bearing for the rest of the doc.

Everything below is verified against the actual `.tf` files in [terraform/](../terraform/), confirmed with a real `terraform validate` and `terraform plan` run — not the idealized behavior Terraform's own docs describe in the abstract. This config has **not yet been applied** (no `terraform apply` has been run against your AWS account) — read this before you run it, because a couple of the choices below (a wide-open SSH rule, secrets in local state) are deliberate trade-offs for a learning project, not hardened defaults.

## Section 0 — Background Primer: what Terraform actually is

**The problem it solves.** Without it, "build me an EC2 instance" means clicking through the AWS Console by hand (not repeatable, no record of what you clicked) or writing AWS CLI/SDK calls yourself (repeatable, but you re-implement "check if it already exists before creating it" every time).

| Approach | Repeatable | Tracks what it created | Update-in-place | Typical use |
|---|---|---|---|---|
| AWS Console (clicking) | No | No | Manual | One-off exploration |
| AWS CLI (`aws ec2 run-instances ...`) | Yes, if scripted | No — you'd track IDs yourself | You write the diff logic | Quick scripts |
| AWS CloudFormation | Yes | Yes (a "stack") | Yes | AWS-only, YAML/JSON |
| **Terraform** | Yes | Yes (a "state file") | Yes | What this project uses — cloud-agnostic, HCL |

Terraform's job in one sentence: you declare the *end state* you want in `.tf` files (HCL — HashiCorp Configuration Language), and Terraform diffs that against a **state file** it maintains, then makes only the AWS API calls needed to close the gap.

**Gotchas that surprise newcomers** (each is the subject of its own section further down):

1. The state file, not your `.tf` files and not the AWS Console, is what Terraform treats as ground truth. If you change something by hand in the AWS Console, Terraform doesn't know — it'll either silently ignore the drift or try to "fix" it back on the next `apply`. See [Section 1](#section-1--architecture-at-a-glance).
2. Marking a variable `sensitive = true` (used for `db_source` and `jwt_secret_key` here) hides it from **terminal output** — it does **not** encrypt it in the state file. The plaintext Supabase URL and JWT secret both land in `terraform.tfstate` regardless. See [Section 8 — Variables & Secrets](#section-8--variablestf--input-schema-and-secrets).
3. `user_data` (the boot script) only runs **once**, on an instance's first boot. Editing [user_data.sh.tpl](../terraform/user_data.sh.tpl) after the instance already exists does nothing until you destroy and recreate it — Terraform won't rerun it in place. See [Section 7](#section-7-user_datash-tpl--the-boot-script).

## Section 1 — Architecture at a Glance

The composition root is [main.tf](../terraform/main.tf). It doesn't implement any business logic itself — it wires together a provider, two data lookups (what already exists in your AWS account), and four resources (what Terraform should create), then hands a rendered boot script to the instance. Everything it depends on — the docker image name, the Supabase URL, the JWT secret — comes in from [variables.tf](../terraform/variables.tf), never hardcoded.

| Concern | Owner | Analogy |
|---|---|---|
| Provider & version pinning | [main.tf:1-22](../terraform/main.tf#L1-L22) | The parts catalog, pinned to specific supplier versions |
| Input schema, defaults, secrets | [variables.tf](../terraform/variables.tf) | The order form's field list |
| "What do I already have?" lookups | [main.tf:24-52](../terraform/main.tf#L24-L52) (default VPC/subnet, latest AMI) | Surveying the lot before building |
| SSH access | [main.tf:54-70](../terraform/main.tf#L54-L70) | Cutting a spare key before the door has a lock |
| Firewall | [main.tf:72-101](../terraform/main.tf#L72-L101) | The guest list at the door |
| The instance itself | [main.tf:103-125](../terraform/main.tf#L103-L125) | The house being built |
| First-boot script | [user_data.sh.tpl](../terraform/user_data.sh.tpl) | The move-in checklist taped to the counter |
| Result surface | [outputs.tf](../terraform/outputs.tf) | The postcard telling you the new address |
| Real secret values | `terraform.tfvars` (gitignored, not in this repo) | The sealed envelope that fills in the order form |

It's split this way so the files you actually edit day-to-day (`terraform.tfvars`, to point at a different Docker image or a different Supabase project) never touch the resource logic in `main.tf`. That split has a consequence worth flagging now and explaining properly in Section 8: because `terraform.tfvars` is gitignored, nobody else can `terraform apply` this config without you handing them real values out-of-band — there's no committed source of truth for the secrets, by design.

## Section 2 — `main.tf:1-22` — Provider & Version Locking

**The problem it solves.** Terraform itself doesn't know how to talk to AWS — that logic lives in a separate plugin ("provider") that Terraform downloads. Without pinning versions, a `terraform init` run six months from now could silently pull a newer major version of the AWS provider with breaking changes.

**How it's implemented** ([main.tf:1-22](../terraform/main.tf#L1-L22)):

```hcl
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    tls = {
      source  = "hashicorp/tls"
      version = "~> 4.0"
    }
    local = {
      source  = "hashicorp/local"
      version = "~> 2.0"
    }
  }
}

provider "aws" {
  region = var.aws_region
}
```

Three providers are pulled in, not just `aws`:

| Provider | What it's used for here |
|---|---|
| `hashicorp/aws` | Every actual AWS resource — VPC lookup, AMI lookup, security group, key pair, EC2 instance |
| `hashicorp/tls` | Generating the SSH keypair locally (Section 4) — nothing AWS-specific about it |
| `hashicorp/local` | Writing that generated private key out to a `.pem` file on your machine (Section 4) |

**If you're new to Terraform:** `~> 6.0` means "allow 6.x, but not 7.0" — this is how the provider stays current with patch/minor fixes without you waking up one day to a major-version breaking change. The exact versions actually resolved (`aws` 6.64.0, `tls` 4.4.1, `local` 2.9.1 as of this doc) get written to `.terraform.lock.hcl`, which — unlike `terraform.tfstate` — **is** meant to be committed, so everyone gets the identical provider build.

`provider "aws" { region = var.aws_region }` reads from [variables.tf:1-5](../terraform/variables.tf#L1-L5), defaulted to `ap-southeast-1`. This replaced a hardcoded `region = "ap-southeast-1"` that was here before this config existed — worth knowing if you diff against git history and wonder why the region became a variable.

## Section 3 — `main.tf:24-52` — "What Do I Already Have?" Lookups

**The problem it solves.** An EC2 instance needs a VPC, a subnet, and an AMI (machine image) ID to exist in. Rather than hardcoding IDs that are specific to one AWS account and would break for anyone else running this config, these are looked up live.

**How it's implemented** ([main.tf:26-52](../terraform/main.tf#L26-L52)):

```hcl
data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
}

data "aws_ami" "al2023" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*-x86_64"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}
```

| Data source | Resolves to (verified against your account) | Used by |
|---|---|---|
| `data.aws_vpc.default` | `vpc-04e8feeb806de220b` — every new AWS account gets one default VPC per region | `aws_security_group.core_service` (Section 5), the subnet lookup below |
| `data.aws_subnets.default` | All subnets inside that default VPC | `aws_instance.core_service` picks `.ids[0]` — the first one returned |
| `data.aws_ami.al2023` | Whatever AMI ID is currently the newest Amazon Linux 2023 x86_64 HVM image | `aws_instance.core_service`'s `ami` field |

**If you're new to Terraform:** blocks starting with `data` don't create anything — they're read-only queries against your AWS account, run fresh on every `plan`/`apply`. Blocks starting with `resource` are the ones that actually create/modify/destroy infrastructure.

**Rough edge worth flagging:** `data.aws_subnets.default.ids[0]` picks whichever subnet AWS's API happens to return first — there's no guarantee it's the "best" one (e.g. a specific availability zone). For a single learning instance this doesn't matter; it would if you ever needed the instance in a specific AZ.

## Section 4 — `main.tf:54-70` — SSH Key Pair Generation

**The problem it solves.** To SSH into the instance for debugging, AWS needs a public key registered as a "key pair" *before* the instance boots. Rather than asking you to have one ready, Terraform generates a fresh keypair and hands you the private half.

**How it's implemented** ([main.tf:56-70](../terraform/main.tf#L56-L70)):

```hcl
resource "tls_private_key" "this" {
  algorithm = "RSA"
  rsa_bits  = 4096
}

resource "aws_key_pair" "this" {
  key_name   = "core-service-key"
  public_key = tls_private_key.this.public_key_openssh
}

resource "local_file" "private_key" {
  content         = tls_private_key.this.private_key_pem
  filename        = "${path.module}/core-service-key.pem"
  file_permission = "0600"
}
```

Three resources chained together: `tls_private_key` generates the RSA keypair in memory, `aws_key_pair` registers only the *public* half with AWS, and `local_file` writes the *private* half to `terraform/core-service-key.pem` on your machine with `0600` permissions (owner read/write only — SSH refuses to use a key that's more open than this).

**Deliberately not using an existing key.** You were asked whether to generate a new keypair, bring your own public key, or skip SSH entirely, and chose "generate a new one" — this is why the config has no variable for an existing public key path.

**Rough edge worth flagging — this private key lives in two places, unencrypted:** the `.pem` file on disk, and (per the Section 0 gotcha) inside `terraform.tfstate` as the `private_key_pem` attribute of `tls_private_key.this`. Both are covered by the `.gitignore` entries added alongside this config (`*.pem`, `*.tfstate`), so neither should ever reach git — but if you ever `cat` or share your `.tfstate` file for debugging, you're sharing the private key too.

## Section 5 — `main.tf:72-101` — Security Group (Firewall)

**The problem it solves.** By default AWS EC2 instances accept no inbound traffic at all. Two ports need to be opened: 22 (SSH, for you to debug) and 8080 (the app port, for anyone to reach the API).

**How it's implemented** ([main.tf:74-101](../terraform/main.tf#L74-L101)):

```hcl
resource "aws_security_group" "core_service" {
  name        = "core-service-sg"
  description = "Allow SSH and app traffic to core-service"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  ingress {
    description = "core-service app port"
    from_port   = var.app_port
    to_port     = var.app_port
    protocol    = "tcp"
    cidr_blocks = var.app_cidr_blocks
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}
```

| Rule | Port | Source | Controlled by |
|---|---|---|---|
| SSH | 22 | `var.ssh_cidr_blocks`, defaults to `0.0.0.0/0` | [variables.tf:25-29](../terraform/variables.tf#L25-L29) |
| App traffic | `var.app_port` (8080) | `var.app_cidr_blocks`, defaults to `0.0.0.0/0` | [variables.tf:31-35](../terraform/variables.tf#L31-L35) |
| Outbound (egress) | all ports/protocols | `0.0.0.0/0` (hardcoded, not a variable) | — needed so the instance can reach Docker Hub and Supabase |

**Rough edge worth flagging on purpose — SSH is open to the entire internet by default.** `ssh_cidr_blocks` defaults to `["0.0.0.0/0"]`, meaning *anyone on the internet* can attempt to SSH in (they'd still need your private key to succeed, but the port is reachable and will show up in scans/logs). [terraform.tfvars.example:13-14](../terraform/terraform.tfvars.example#L13-L14) already flags this with a comment — narrowing it to your own IP (`["1.2.3.4/32"]`) is a one-line change in `terraform.tfvars` and worth doing before you leave this running unattended.

## Section 6 — `main.tf:103-125` — The EC2 Instance

**The problem it solves.** This is the resource everything else in the file exists to support — the actual virtual machine that will run the Docker container.

**How it's implemented** ([main.tf:105-125](../terraform/main.tf#L105-L125)):

```hcl
resource "aws_instance" "core_service" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  key_name               = aws_key_pair.this.key_name
  vpc_security_group_ids = [aws_security_group.core_service.id]

  user_data = templatefile("${path.module}/user_data.sh.tpl", {
    docker_image              = var.docker_image
    app_port                  = var.app_port
    db_driver                 = var.db_driver
    db_source                 = var.db_source
    jwt_secret_key            = var.jwt_secret_key
    jwt_access_token_duration = var.jwt_access_token_duration
    cors_allowed_origins      = var.cors_allowed_origins
  })

  tags = {
    Name = "core-service"
  }
}
```

Every argument here is a reference to something built in an earlier section — `ami` from Section 3, `key_name` from Section 4, `vpc_security_group_ids` from Section 5. The one new piece of mechanism is `templatefile(...)`: it reads [user_data.sh.tpl](../terraform/user_data.sh.tpl) off disk, substitutes each `${...}` placeholder with the value on the right, and hands AWS the resulting plain-text script as the instance's "user data" — EC2's built-in first-boot hook (covered in Section 0's third gotcha, and in full in Section 7).

**If you're new to Terraform:** `instance_type = var.instance_type` defaults to `t3.micro` ([variables.tf:7-11](../terraform/variables.tf#L7-L11)) — chosen because it's free-tier eligible (750 hrs/month for a new account's first 12 months) and comfortably enough for a small Go binary with light traffic.

## Section 7 — `user_data.sh.tpl` — The Boot Script

**The problem it solves.** A bare Amazon Linux 2023 AMI has no Docker installed and knows nothing about this project. Something has to install Docker and start the container the first time the instance boots, with no human present to type commands.

**How it's implemented**, in full — it's short enough that every line matters ([user_data.sh.tpl:1-24](../terraform/user_data.sh.tpl#L1-L24)):

```bash
#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

docker pull ${docker_image}

docker rm -f core-service || true

docker run -d \
  --name core-service \
  --restart unless-stopped \
  -p ${app_port}:${app_port} \
  -e DB_DRIVER="${db_driver}" \
  -e DB_SOURCE="${db_source}" \
  -e SERVER_ADDRESS="0.0.0.0:${app_port}" \
  -e JWT_SECRET_KEY="${jwt_secret_key}" \
  -e JWT_ACCESS_TOKEN_DURATION="${jwt_access_token_duration}" \
  -e CORS_ALLOWED_ORIGINS="${cors_allowed_origins}" \
  ${docker_image}
```

Step by step: `dnf install -y docker` installs Docker from Amazon Linux 2023's own package repo (Amazon Linux uses `dnf`, not `apt`); `systemctl enable --now docker` starts the daemon and makes it survive reboots; `docker pull` fetches your image from Docker Hub by tag; `docker rm -f core-service || true` removes any previous container of the same name without failing the script if none exists yet; `docker run -d --restart unless-stopped` starts it detached and set to restart automatically if it crashes or the instance reboots.

**The six `-e` flags are not arbitrary** — they're a 1:1 match to the `mapstructure` tags on `Config` in [internal/config/config.go:13-19](../internal/config/config.go#L13-L19):

| Env var set here | Config field | Note |
|---|---|---|
| `DB_DRIVER` | `DBDriver` | Hardcoded to `postgres` via [variables.tf:39-43](../terraform/variables.tf#L39-L43)'s default |
| `DB_SOURCE` | `DBSource` | Your Supabase connection string |
| `SERVER_ADDRESS` | `ServerAddress` | Hardcoded to `0.0.0.0:${app_port}` here, not a separate variable |
| `JWT_SECRET_KEY` | `JWTSecretKey` | Validated ≥32 chars at plan time (Section 8) — but note the callout below |
| `JWT_ACCESS_TOKEN_DURATION` | `JWTAccessTokenDuration` | Defaults to `60m`, matching `app.env`'s local value |
| `CORS_ALLOWED_ORIGINS` | `CORSAllowedOrigins` | Defaults to empty, which per [docs/CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md#section-4--consumers-where-config-values-actually-get-used) deliberately disables CORS entirely |

No `app.env` file is created or mounted anywhere in this script — it relies entirely on the `AutomaticEnv()` + `BindEnv` mechanism documented in [docs/CONFIG_ENV_VARIABLE.md](CONFIG_ENV_VARIABLE.md#section-3--loadconfig--merging-a-file-with-the-environment), the same mechanism already proven out by the `core-services` service in the project's `docker-compose.yaml`.

**Rough edge worth flagging — this script only runs once.** Per the Section 0 gotcha, EC2 `user_data` executes on first boot only. If you push a new image tag to Docker Hub and want the running instance to pick it up, re-running `terraform apply` with a new `docker_image` value will **not** re-run this script on the existing instance — you'd need to either SSH in and re-run the `docker pull`/`docker run` lines by hand, or `terraform taint aws_instance.core_service` (or change something that forces replacement) so Terraform destroys and recreates it. This config has no auto-redeploy-on-new-image mechanism today.

**Rough edge worth flagging — secrets are visible in plain text on the instance.** `DB_SOURCE` and `JWT_SECRET_KEY` are interpolated directly into the rendered `user_data`, which AWS stores as instance metadata, readable from inside the instance via `curl http://169.254.169.254/latest/user-data` by anything running on it (or anyone who gets SSH access). This is standard practice for a simple `user_data` deploy and matches how `docker-compose.yaml` already handles these values (plain `environment:` entries) — but it's worth knowing this isn't a secrets-manager-grade setup.

## Section 8 — `variables.tf` — Input Schema and Secrets

**The problem it solves.** Every value that should differ between "your Supabase project" and someone else's, or between environments, needs one typed, documented place — instead of `sdewa/core-service-dep` or a Supabase URL being pasted directly into `main.tf`.

| Variable | Default | Sensitive? | Consumed by |
|---|---|---|---|
| `aws_region` | `ap-southeast-1` | No | `provider "aws"` ([main.tf:20-22](../terraform/main.tf#L20-L22)) |
| `instance_type` | `t3.micro` | No | `aws_instance.core_service` |
| `docker_image` | `sdewa/core-service-dep:latest` | No | `user_data`, matches `DEP_IMAGE_NAME` in [Makefile:51](../Makefile#L51) |
| `app_port` | `8080` | No | security group + `user_data` |
| `ssh_cidr_blocks` | `["0.0.0.0/0"]` | No | security group's SSH rule (see Section 5's callout) |
| `app_cidr_blocks` | `["0.0.0.0/0"]` | No | security group's app-port rule |
| `db_driver` | `postgres` | No | `user_data`'s `DB_DRIVER` |
| `db_source` | *(none — required)* | **Yes** | `user_data`'s `DB_SOURCE` |
| `jwt_secret_key` | *(none — required)* | **Yes**, validated ≥32 chars | `user_data`'s `JWT_SECRET_KEY` |
| `jwt_access_token_duration` | `60m` | No | `user_data`'s `JWT_ACCESS_TOKEN_DURATION` |
| `cors_allowed_origins` | `""` | No | `user_data`'s `CORS_ALLOWED_ORIGINS` |

**The validation block**, worth showing on its own ([variables.tf:56-59](../terraform/variables.tf#L56-L59)):

```hcl
validation {
  condition     = length(var.jwt_secret_key) >= 32
  error_message = "jwt_secret_key must be at least 32 characters."
}
```

This mirrors the exact check in [internal/token/jwt_maker.go:11,21-23](../internal/token/jwt_maker.go#L11) (`minSecretKeySize = 32`) — the goal is to fail at `terraform plan` time, before any AWS resources are touched, rather than have the container start and crash on the instance where the failure is harder to see.

**If you're new to Terraform:** `db_source` and `jwt_secret_key` have no `default` and are marked `sensitive = true`. No default means `terraform plan`/`apply` will refuse to run until you supply them (via `terraform.tfvars`, a `-var` flag, or an interactive prompt). `sensitive = true` means Terraform prints `(sensitive value)` instead of the real string in plan/apply output and in `terraform show` — but as flagged in Section 0, this is a display-only redaction, not encryption; the real value is still in `terraform.tfstate`.

**How you're expected to supply them:** [terraform.tfvars.example](../terraform/terraform.tfvars.example) is the checked-in template. You copy it to `terraform.tfvars` (gitignored — see the `.gitignore` entries this setup added: `*.tfvars` with a `!*.tfvars.example` exception) and fill in your real Supabase connection string and a real JWT secret. Nobody else with this repo gets your secrets unless you hand them `terraform.tfvars` directly.

## Section 9 — `outputs.tf` — Result Surface

**The problem it solves.** After `terraform apply` finishes, you need the instance's public IP to actually reach it — outputs are how Terraform surfaces values computed only after resources exist.

| Output | Value | Purpose |
|---|---|---|
| `instance_id` | `aws_instance.core_service.id` | Reference for AWS CLI/Console lookups |
| `public_ip` | `aws_instance.core_service.public_ip` | Raw IP |
| `app_url` | `http://<public_ip>:<app_port>` | Ready-to-curl URL |
| `ssh_command` | `ssh -i <pem path> ec2-user@<public_ip>` | Copy-pasteable SSH command, using the exact `.pem` path written by [main.tf:66-70](../terraform/main.tf#L66-L70) |

**Rough edge worth flagging — the public IP is not static.** A plain `aws_instance` gets a new public IP any time it's stopped and restarted (not on every `apply`, but on a real reboot/stop-start cycle) unless you attach an Elastic IP, which this config does not do. For a learning deployment this is fine; if you start relying on a fixed address (e.g. pointing a domain at it), that's a gap to close later.

## Section 10 — Supporting Files (No Logic)

Two files exist alongside the `.tf` files but aren't read by Terraform itself:

- [terraform.tfvars.example](../terraform/terraform.tfvars.example) — a template for the real `terraform.tfvars`, with placeholder values and inline comments (e.g. "use the pooler host for EC2" for Supabase). Terraform never reads the `.example` file directly; you copy it.
- `.terraform.lock.hcl` (created by `terraform init`, not committed by hand but meant to be committed) — pins the exact provider builds resolved during this doc's verification (`aws` 6.64.0, `tls` 4.4.1, `local` 2.9.1).

## Cross-Feature Coupling

- **This config never builds or pushes the Docker image.** That happens entirely outside Terraform, via `make docker-dep-build` and `make docker-dep-push` ([Makefile:53-57](../Makefile#L53-L57)), which build from [Dockerfile](../Dockerfile) and push to the `sdewa/core-service-dep` Docker Hub repo. If you change application code, you must rebuild and push the image *and* deal with the Section 7 "runs once" gotcha to get the new image onto a running instance — `terraform apply` alone does not pick up new app code.
- **This config never runs database migrations.** The Docker image built by [Dockerfile:1-18](../Dockerfile#L1-L18) only contains the compiled server binary (`ENTRYPOINT ["/app/main"]`) — the separate migration CLI (`cmd/migration`, invoked via `make migrate-up` per [Makefile:17-18](../Makefile#L17-L18)) is not part of the image and is never invoked by `user_data`. Before the API on the new instance can serve any request that touches the database, you need to run `go run ./cmd/migration/main.go up` from your own machine with `DB_SOURCE` pointed at the same Supabase URL you put in `terraform.tfvars`. This is easy to miss because the container will start and look healthy — requests just fail at the query layer.
- **Local `app.env` and this deployment are two independent, unsynchronized configs.** [app.env](../app.env) points at your local `docker-compose` Postgres; `terraform.tfvars` points at Supabase. Editing one never affects the other — worth knowing so you don't go looking for a shared config file that doesn't exist.

## Summary — Data Flow

**Provisioning (`terraform apply`):**
```
terraform apply
  → reads terraform.tfvars for db_source, jwt_secret_key (+ any overrides)
  → data lookups resolve: default VPC, default subnet, latest AL2023 AMI
  → tls_private_key generates an RSA keypair
  → aws_key_pair registers the public half with AWS; local_file writes the private half to core-service-key.pem
  → aws_security_group opens ports 22 and 8080
  → templatefile() renders user_data.sh.tpl with all seven substituted values
  → aws_instance.core_service is created with that rendered script as its user_data
  → outputs.tf prints instance_id, public_ip, app_url, ssh_command
```

**First boot (on the EC2 instance, no human involved):**
```
instance boots
  → cloud-init runs user_data as root
  → dnf installs and starts Docker
  → docker pull <docker_image> from Docker Hub
  → docker run -d --restart unless-stopped, with the 6 env vars set
  → main binary starts, calls config.LoadConfig(".") → finds no app.env → binds env vars directly (Section 7 table)
  → connectDB(cfg) opens a connection to your Supabase Postgres using DB_SOURCE
  → server listens on 0.0.0.0:8080
```

**Request path (after boot):**
```
client → http://<public_ip>:8080/... 
  → security group allows it in on port 8080 (Section 5)
  → container's Gin router handles it (see docs/GIN_HTTP_LAYER.md)
  → any DB-backed endpoint queries Supabase over the internet (no VPC peering/private link configured)
```

## Final Reference — Every Resource and Data Source

| Address | Kind | Purpose |
|---|---|---|
| `data.aws_vpc.default` | data | Look up the account's default VPC |
| `data.aws_subnets.default` | data | Look up subnets inside that VPC |
| `data.aws_ami.al2023` | data | Look up the latest Amazon Linux 2023 x86_64 AMI |
| `tls_private_key.this` | resource | Generate an SSH RSA keypair |
| `aws_key_pair.this` | resource | Register the public key with AWS as `core-service-key` |
| `local_file.private_key` | resource | Write the private key to `terraform/core-service-key.pem` |
| `aws_security_group.core_service` | resource | Firewall: open 22 (SSH) and 8080 (app), allow all egress |
| `aws_instance.core_service` | resource | The EC2 instance itself, running the Docker container via `user_data` |

**Commands you'll actually run**, none of which currently exist as `Makefile` targets — everything below is a direct `terraform` invocation from inside `terraform/`:

| Command | Effect |
|---|---|
| `terraform init` | Downloads the three providers, writes `.terraform.lock.hcl` |
| `terraform plan` | Shows what would change, without changing anything |
| `terraform apply` | Actually creates/updates the resources |
| `terraform destroy` | Tears everything down — including the EC2 instance and security group. Does **not** touch Supabase, since that's outside this config entirely |
