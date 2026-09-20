# Terraform EC2 Deployment — As Implemented

## Who this doc is for

You're assumed to be comfortable with Go, Docker, and this project's Makefile — but you've said you haven't used Terraform before, so this doc treats it as unfamiliar. If you already know Terraform, skip [Section 0](#section-0--background-primer-what-terraform-actually-is) — it's a primer, not load-bearing for the rest of the doc.

Everything below is verified against the actual `.tf` files in [infra/terraform/](../), confirmed with a real `terraform validate` run and cross-checked against the live `terraform.tfstate` for the instance that's actually running — not the idealized behavior Terraform's own docs describe in the abstract. Read this before you change anything, because a couple of the choices below (a wide-open SSH rule, secrets in local state and in the `rendered_user_data` output) are deliberate trade-offs for a learning project, not hardened defaults.

This config, and the docs describing it, moved here from `apps/core-service/terraform/` — it's no longer core-service-only infrastructure. It now also provisions an nginx reverse proxy in front of core-service, so it lives at the repo root instead of inside one app.

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
2. Marking a variable `sensitive = true` (used for `db_source` and `jwt_secret_key` here, and now for the `rendered_user_data` output too) hides it from **terminal output** — it does **not** encrypt it in the state file. The plaintext Supabase URL and JWT secret both land in `terraform.tfstate` regardless. See [Section 8 — Variables & Secrets](#section-8--variablestf--input-schema-and-secrets).
3. `user_data` (the boot script) only runs **once**, on an instance's first boot. Editing [user_data.sh.tpl](../user_data.sh.tpl) after the instance already exists does nothing until you destroy and recreate it — Terraform won't rerun it in place. This is also why `main.tf` now pulls the rendered script into `local.user_data` and exposes it as the `rendered_user_data` output: `make tf-redeploy` uses that output to rerun the exact same script over SSH against an already-running instance, since Terraform itself has no built-in way to do that. See [Section 6](#section-6--maintf107-156--rendering-the-boot-script-and-the-ec2-instance), [Section 7](#section-7--user_datash-tpl--the-boot-script), and [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md).

## Section 1 — Architecture at a Glance

The composition root is [main.tf](../main.tf). It doesn't implement any business logic itself — it wires together a provider, two data lookups (what already exists in your AWS account), five resources (what Terraform should create), and a `locals` block that renders both the nginx reverse-proxy config and the instance's boot script, then hands that rendered script to the instance. Everything it depends on — the docker images, the Supabase URL, the JWT secret, the rate-limit numbers — comes in from [variables.tf](../variables.tf), never hardcoded.

| Concern | Owner | Analogy |
|---|---|---|
| Provider & version pinning | [main.tf:1-22](../main.tf#L1-L22) | The parts catalog, pinned to specific supplier versions |
| Input schema, defaults, secrets | [variables.tf](../variables.tf) | The order form's field list |
| "What do I already have?" lookups | [main.tf:26-52](../main.tf#L26-L52) (default VPC/subnet, latest AMI) | Surveying the lot before building |
| SSH access | [main.tf:56-70](../main.tf#L56-L70) | Cutting a spare key before the door has a lock |
| Firewall | [main.tf:78-105](../main.tf#L78-L105) | The guest list at the door — now it's nginx's door, not core-service's |
| Rendering nginx's config + the boot script | [main.tf:107-140](../main.tf#L107-L140) | The printer filling in the move-in checklist and the gate attendant's rulebook before either is handed over |
| The instance itself | [main.tf:144-156](../main.tf#L144-L156) | The house being built |
| First-boot script | [user_data.sh.tpl](../user_data.sh.tpl) | The move-in checklist taped to the counter — now it also unpacks a gate attendant |
| Reverse-proxy config | [nginx.conf.tpl](../nginx.conf.tpl) | The gate attendant's rulebook: one rate-limit rule, one proxy rule |
| Result surface | [outputs.tf](../outputs.tf) | The postcard telling you the new address |
| Real secret values | `terraform.tfvars` (gitignored, not in this repo) | The sealed envelope that fills in the order form |

It's split this way so the files you actually edit day-to-day (`terraform.tfvars`, to point at a different Docker image, a different Supabase project, or different rate-limit numbers) never touch the resource logic in `main.tf`. That split has a consequence worth flagging now and explaining properly in Section 8: because `terraform.tfvars` is gitignored, nobody else can `terraform apply` this config without you handing them real values out-of-band — there's no committed source of truth for the secrets, by design.

## Section 2 — `main.tf:1-22` — Provider & Version Locking

**The problem it solves.** Terraform itself doesn't know how to talk to AWS — that logic lives in a separate plugin ("provider") that Terraform downloads. Without pinning versions, a `terraform init` run six months from now could silently pull a newer major version of the AWS provider with breaking changes.

**How it's implemented** ([main.tf:1-22](../main.tf#L1-L22)):

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

**If you're new to Terraform:** `~> 6.0` means "allow 6.x, but not 7.0" — this is how the provider stays current with patch/minor fixes without you waking up one day to a major-version breaking change. The exact versions actually resolved (`aws` 6.64.0, `tls` 4.4.1, `local` 2.9.1 as of this doc) get written to `.terraform.lock.hcl`, which — unlike `terraform.tfstate` — **is** meant to be committed, so everyone gets the identical provider build. This is unchanged by the move from `apps/core-service/terraform/`; only the file's location changed, not its content.

`provider "aws" { region = var.aws_region }` reads from [variables.tf:1-5](../variables.tf#L1-L5), defaulted to `ap-southeast-1`.

## Section 3 — `main.tf:26-52` — "What Do I Already Have?" Lookups

**The problem it solves.** An EC2 instance needs a VPC, a subnet, and an AMI (machine image) ID to exist in. Rather than hardcoding IDs that are specific to one AWS account and would break for anyone else running this config, these are looked up live.

**How it's implemented** ([main.tf:26-52](../main.tf#L26-L52)):

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

## Section 4 — `main.tf:56-70` — SSH Key Pair Generation

**The problem it solves.** To SSH into the instance for debugging, AWS needs a public key registered as a "key pair" *before* the instance boots. Rather than asking you to have one ready, Terraform generates a fresh keypair and hands you the private half.

**How it's implemented** ([main.tf:56-70](../main.tf#L56-L70)):

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

Three resources chained together: `tls_private_key` generates the RSA keypair in memory, `aws_key_pair` registers only the *public* half with AWS, and `local_file` writes the *private* half to `infra/terraform/core-service-key.pem` on your machine with `0600` permissions (owner read/write only — SSH refuses to use a key that's more open than this).

**Deliberately not using an existing key.** You were asked whether to generate a new keypair, bring your own public key, or skip SSH entirely, and chose "generate a new one" — this is why the config has no variable for an existing public key path.

**Rough edge worth flagging — this private key lives in two places, unencrypted:** the `.pem` file on disk, and (per the Section 0 gotcha) inside `terraform.tfstate` as the `private_key_pem` attribute of `tls_private_key.this`. Both are covered by `.gitignore` entries at the repo root (`*.pem`, `*.tfstate`), so neither should ever reach git — but if you ever `cat` or share your `.tfstate` file for debugging, you're sharing the private key too.

## Section 5 — `main.tf:78-105` — Security Group (Firewall)

**The problem it solves.** By default AWS EC2 instances accept no inbound traffic at all. Now that nginx sits in front of core-service, only two ports need to be opened at the security-group level: 22 (SSH, for you to debug) and `var.nginx_port` (80 by default — nginx, the only public entrypoint).

**How it's implemented** ([main.tf:72-105](../main.tf#L72-L105)):

```hcl
# --- Security group: SSH + nginx (the only public entrypoint) ---
#
# app_port is deliberately not opened here — core-service is only reachable
# from nginx over the internal docker network (see user_data.sh.tpl), and
# from the instance itself via the 127.0.0.1-bound debug port.

resource "aws_security_group" "core_service" {
  name        = "core-service-sg"
  description = "Allow SSH and nginx traffic to the core-service host"
  vpc_id      = data.aws_vpc.default.id

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.ssh_cidr_blocks
  }

  ingress {
    description = "nginx (reverse proxy + rate limiter in front of core-service)"
    from_port   = var.nginx_port
    to_port     = var.nginx_port
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
| SSH | 22 | `var.ssh_cidr_blocks`, defaults to `0.0.0.0/0` | [variables.tf:25-29](../variables.tf#L25-L29) |
| nginx (public entrypoint) | `var.nginx_port` (80) | `var.app_cidr_blocks`, defaults to `0.0.0.0/0` | [variables.tf:31-35](../variables.tf#L31-L35), [variables.tf:45-49](../variables.tf#L45-L49) |
| Outbound (egress) | all ports/protocols | `0.0.0.0/0` (hardcoded, not a variable) | — needed so the instance can reach Docker Hub and Supabase |

Notice `var.app_port` (core-service's own 8080) does not appear in this security group at all — this is the headline architecture change in this revision. Previously the second ingress rule opened `var.app_port` directly, so core-service's own container published 8080 straight to the internet. Now that rule opens `var.nginx_port` instead, and [user_data.sh.tpl](../user_data.sh.tpl) (Section 7) never publishes `app_port` on the instance's public interface at all — it binds it to `127.0.0.1:${app_port}` for SSH-side `curl` debugging only. The only way to reach core-service from outside the instance is through nginx, over the `core-service-net` docker network, by container name (`proxy_pass http://core-service:${app_port}` in [nginx.conf.tpl](../nginx.conf.tpl)).

**Two rate limiters, by design — not redundant.** This revision adds rate limiting in two independent places on the request path:

1. **nginx's `limit_req`** ([nginx.conf.tpl](../nginx.conf.tpl), rendered in Section 6 below) does a coarse per-client-IP limit at the edge, keyed by `$binary_remote_addr`, sized by `var.nginx_rate_limit_rps`/`var.nginx_rate_limit_burst` (defaults 10 req/s, burst 20). It runs *before* a request ever reaches the Go process — an abusive client gets a 429 straight from nginx and never costs a database connection, a JWT parse, or anything else core-service would have spent doing the same rejection.
2. **core-service's own in-process token-bucket limiter** (`apps/core-service/internal/rate-limiter/`, gated by `RATE_LIMIT_ENABLED` — see [apps/core-service/AGENTS.md](../../../apps/core-service/AGENTS.md)) stays on behind it, now driven by the new `app_rate_limit_enabled`/`app_rate_limit_rps`/`app_rate_limit_burst` variables (defaults: enabled, 5 req/s, burst 20 — deliberately tighter than nginx's, since nginx already sheds the worst of it). This is groundwork for future per-route tuning that a blanket edge limit can't do — nginx has no idea whether a request is a cheap health check or an expensive multi-account transfer.

Worth flagging for accuracy: before this revision, `user_data.sh.tpl` never set `RATE_LIMIT_ENABLED` at all, so the previously-running instance had core-service's in-process limiter off despite the application code supporting it. This revision is the first time it's actually turned on in this deployment, via the `app_rate_limit_*` variables described in Section 8.

**Rough edge worth flagging on purpose — SSH is open to the entire internet by default.** `ssh_cidr_blocks` defaults to `["0.0.0.0/0"]`, meaning *anyone on the internet* can attempt to SSH in (they'd still need your private key to succeed, but the port is reachable and will show up in scans/logs). [terraform.tfvars.example:13-14](../terraform.tfvars.example#L13-L14) already flags this with a comment — narrowing it to your own IP (`["1.2.3.4/32"]`) is a one-line change in `terraform.tfvars` and worth doing before you leave this running unattended.

## Section 6 — `main.tf:107-156` — Rendering the Boot Script, and the EC2 Instance

**The problem it solves.** Two things need to happen before the instance boots: nginx needs a concrete config file, with real port numbers and rate-limit values filled in instead of hardcoded, and the instance needs a first-boot script that installs and wires together *two* containers instead of one. Both are assembled here, in a `locals` block, ahead of the `aws_instance` resource that actually consumes them.

**How it's implemented** ([main.tf:107-140](../main.tf#L107-L140)):

```hcl
# --- nginx config: rendered here so main.tf/variables.tf stay the single
#     source of truth for ports and rate-limit knobs, then shipped to the
#     instance base64-encoded inside user_data (see Section on user_data.sh.tpl) ---

locals {
  nginx_conf = templatefile("${path.module}/nginx.conf.tpl", {
    nginx_port       = var.nginx_port
    app_port         = var.app_port
    rate_limit_rps   = var.nginx_rate_limit_rps
    rate_limit_burst = var.nginx_rate_limit_burst
  })

  # Pulled into its own local (rather than inlined in aws_instance below) so
  # it can also be exposed via outputs.tf's rendered_user_data — EC2 only
  # runs this script on an instance's first boot, so re-running it by hand
  # over SSH is how an already-running instance picks up config changes
  # (see docs/TERRAFORM_EC2_DEPLOY.md Section 7). Reusing this exact value
  # for both means there's no separate, driftable "redeploy script."
  user_data = templatefile("${path.module}/user_data.sh.tpl", {
    docker_image              = var.docker_image
    app_port                  = var.app_port
    db_driver                 = var.db_driver
    db_source                 = var.db_source
    jwt_secret_key            = var.jwt_secret_key
    jwt_access_token_duration = var.jwt_access_token_duration
    cors_allowed_origins      = var.cors_allowed_origins
    rate_limit_enabled        = var.app_rate_limit_enabled
    rate_limit_rps            = var.app_rate_limit_rps
    rate_limit_burst          = var.app_rate_limit_burst
    nginx_image               = var.nginx_image
    nginx_port                = var.nginx_port
    nginx_conf_base64         = base64encode(local.nginx_conf)
  })
}
```

`local.nginx_conf` renders [nginx.conf.tpl](../nginx.conf.tpl) (covered line-by-line in Section 7) with the four values that actually vary: the port nginx listens on, the port it proxies to, and its rate-limit rate/burst. `local.user_data` renders [user_data.sh.tpl](../user_data.sh.tpl) with thirteen substituted values — the original six app-config values from before this revision, the three new `app_rate_limit_*` values, and three nginx-related values, one of which (`nginx_conf_base64`) is `base64encode(local.nginx_conf)`: the rendered nginx config gets embedded as a base64 blob *inside* the rendered boot script, which then decodes it back into a real file once it's running on the instance (Section 7). Terraform resolves the dependency between the two locals automatically; the file just orders them the same way for readability.

**Why `user_data` is pulled into a named `local` instead of being inlined directly into `aws_instance.core_service`'s `user_data` argument (which is how the pre-nginx version of this config did it):** so the exact same rendered value can also be exposed as the `rendered_user_data` output (Section 9). EC2 only ever runs `user_data` automatically on an instance's first boot (Section 0's third gotcha) — Terraform has no built-in "re-run this on the existing instance" primitive. Because `aws_instance.core_service.user_data` and `output.rendered_user_data` are now both drawn from the identical `local.user_data`, there's exactly one source of truth for "what should currently be running on this instance." `make tf-redeploy` (see [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md)) uses that output to re-run the current script over SSH against the already-running instance, instead of a second, hand-maintained redeploy script that could silently drift out of sync with what a freshly created instance actually boots with.

**The instance itself** ([main.tf:142-156](../main.tf#L142-L156)):

```hcl
# --- EC2 instance running core-service behind an nginx reverse proxy ---

resource "aws_instance" "core_service" {
  ami                    = data.aws_ami.al2023.id
  instance_type          = var.instance_type
  subnet_id              = data.aws_subnets.default.ids[0]
  key_name               = aws_key_pair.this.key_name
  vpc_security_group_ids = [aws_security_group.core_service.id]

  user_data = local.user_data

  tags = {
    Name = "core-service"
  }
}
```

Every argument here is a reference to something built in an earlier section — `ami` from Section 3, `key_name` from Section 4, `vpc_security_group_ids` from Section 5, `user_data` from the `locals` block above. `instance_type = var.instance_type` defaults to `t3.micro` ([variables.tf:7-11](../variables.tf#L7-L11)) — chosen because it's free-tier eligible (750 hrs/month for a new account's first 12 months) and comfortably enough for a small Go binary plus an nginx sidecar under light traffic.

## Section 7 — `user_data.sh.tpl` — The Boot Script

**The problem it solves.** A bare Amazon Linux 2023 AMI has no Docker installed and knows nothing about this project. Something has to install Docker, start core-service, start nginx in front of it, and wire the two together on a private network — the first time the instance boots, with no human present to type commands.

**How it's implemented**, in full, quoted verbatim from [user_data.sh.tpl](../user_data.sh.tpl):

```bash
#!/bin/bash
set -euxo pipefail

dnf update -y
dnf install -y docker
systemctl enable --now docker
usermod -aG docker ec2-user

# Shared network so nginx can reach core-service by container name; neither
# container needs to publish app_port on the host for that to work.
docker network inspect core-service-net >/dev/null 2>&1 || docker network create core-service-net

docker pull ${docker_image}
docker pull ${nginx_image}

docker rm -f nginx || true
docker rm -f core-service || true

mkdir -p /opt/core-service
cat > /opt/core-service/app.env <<EOF
DB_DRIVER=${db_driver}
DB_SOURCE=${db_source}
SERVER_ADDRESS=0.0.0.0:${app_port}
JWT_SECRET_KEY=${jwt_secret_key}
JWT_ACCESS_TOKEN_DURATION=${jwt_access_token_duration}
CORS_ALLOWED_ORIGINS=${cors_allowed_origins}
RATE_LIMIT_ENABLED=${rate_limit_enabled}
RATE_LIMIT_REQUESTS_PER_SECOND=${rate_limit_rps}
RATE_LIMIT_BURST=${rate_limit_burst}
EOF
chmod 600 /opt/core-service/app.env

# Bound to 127.0.0.1 only, for SSH-side debugging (curl from the instance
# itself) — not reachable from outside since it's not in the security group.
# The public entrypoint is nginx below.
docker run -d \
  --name core-service \
  --restart unless-stopped \
  --network core-service-net \
  -p 127.0.0.1:${app_port}:${app_port} \
  -v /opt/core-service/app.env:/app/app.env:ro \
  ${docker_image}

mkdir -p /opt/nginx
echo '${nginx_conf_base64}' | base64 -d > /opt/nginx/default.conf
chmod 644 /opt/nginx/default.conf

docker run -d \
  --name nginx \
  --restart unless-stopped \
  --network core-service-net \
  -p ${nginx_port}:${nginx_port} \
  -v /opt/nginx/default.conf:/etc/nginx/conf.d/default.conf:ro \
  ${nginx_image}
```

Step by step: the Docker install/enable/`usermod` lines are unchanged from before this revision. `docker network inspect core-service-net >/dev/null 2>&1 || docker network create core-service-net` creates the private bridge network both containers join — written idempotently (inspect first, create only if missing) so re-running this exact script, e.g. via `make tf-redeploy`, doesn't fail just because the network already exists. This network is what lets nginx reach core-service by the hostname `core-service` without either container publishing `app_port` to the host. Both images are pulled up front; both old containers are removed by name with `|| true` so a first run with nothing to remove doesn't fail the script.

**core-service now gets a real `app.env` file, not just `-e` flags.** Previously this script passed every config value as a `docker run -e` flag; now it writes `/opt/core-service/app.env` (mode `0600`, so only root can read the plaintext DB/JWT secrets in it) and bind-mounts it read-only into the container at `/app/app.env`, using the same `config.LoadConfig(".")` file-loading path documented in [CONFIG_ENV_VARIABLE.md](../../../apps/core-service/docs/CONFIG_ENV_VARIABLE.md). This mount matters for a reason specific to this project: [Dockerfile.prod](../../../apps/core-service/Dockerfile.prod) (the image `docker_image` actually builds from — see Cross-Feature Coupling below) already bakes `app.prod.env` into the image as `/app/app.env` at build time. Without the `-v` mount here, the container would boot on whatever was baked in at image-build time, silently ignoring every Terraform variable below. The mount overrides that baked-in file with the one rendered from `terraform.tfvars`.

The nine keys written to `app.env` are a 1:1 match to the `mapstructure` tags on `Config` in [internal/config/config.go:13-53](../../../apps/core-service/internal/config/config.go#L13-L53) — six from before this revision, three new:

| Key in app.env | Config field | Note |
|---|---|---|
| `DB_DRIVER` | `DBDriver` | Hardcoded to `postgres` via [variables.tf:65-69](../variables.tf#L65-L69)'s default |
| `DB_SOURCE` | `DBSource` | Your Supabase connection string |
| `SERVER_ADDRESS` | `ServerAddress` | Hardcoded to `0.0.0.0:${app_port}` here, not a separate variable |
| `JWT_SECRET_KEY` | `JWTSecretKey` | Validated ≥32 chars at plan time (Section 8) |
| `JWT_ACCESS_TOKEN_DURATION` | `JWTAccessTokenDuration` | Defaults to `60m` |
| `CORS_ALLOWED_ORIGINS` | `CORSAllowedOrigins` | Defaults to empty, which per [CONFIG_ENV_VARIABLE.md](../../../apps/core-service/docs/CONFIG_ENV_VARIABLE.md) deliberately disables CORS entirely |
| `RATE_LIMIT_ENABLED` | `RateLimitEnabled` | New in this revision — defaults to `true` via `var.app_rate_limit_enabled` |
| `RATE_LIMIT_REQUESTS_PER_SECOND` | `RateLimitRequestsPerSecond` | New — defaults to `5` via `var.app_rate_limit_rps` |
| `RATE_LIMIT_BURST` | `RateLimitBurst` | New — defaults to `20` via `var.app_rate_limit_burst` |

core-service is started with `--network core-service-net -p 127.0.0.1:${app_port}:${app_port}` — joined to the private network (so nginx can reach it by name) but published only to the loopback interface, not the instance's public one. That publish is purely a debugging convenience (SSH in, `curl 127.0.0.1:8080/...`); it isn't how real traffic reaches core-service.

**Then nginx.** `/opt/nginx/default.conf` is written by base64-decoding `${nginx_conf_base64}` — the pre-rendered `local.nginx_conf` from Section 6, decoded back into a real file (`0644`, since it holds no secrets) and bind-mounted read-only over the stock `nginx:1.27-alpine` image's default vhost config. nginx joins the same `core-service-net` network and is the only one of the two containers that publishes its port to the instance's *public* interface (`-p ${nginx_port}:${nginx_port}`) — matching the security group from Section 5, which opens exactly `nginx_port` and nothing else for public traffic.

**Rough edge worth flagging — this script only runs automatically once.** Per the Section 0 gotcha, EC2 `user_data` executes on first boot only. Section 6 above covers how this revision addresses that at the Terraform level: the exact same rendered script is available as the `rendered_user_data` output, and `make tf-redeploy` (see [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md)) pipes it over SSH to rerun on the existing instance. The script is written to be safe to rerun that way — the idempotent `docker network inspect ... ||`, the `docker rm -f ... || true` pair, and unconditional `docker run` each time — so a redeploy is an expected, ordinary use of this exact file, not a workaround bolted on separately.

**Rough edge worth flagging — secrets are visible in plain text on the instance and in the `rendered_user_data` output.** `DB_SOURCE` and `JWT_SECRET_KEY` land in `/opt/core-service/app.env` (`chmod 600`, root-readable only) and, separately, in the rendered `user_data` itself, which AWS stores as instance metadata (readable from inside the instance via `curl http://169.254.169.254/latest/user-data`) and which the `rendered_user_data` output also carries verbatim. That's exactly why that output is marked `sensitive = true` (Section 9) — the same display-only-redaction caveat from Section 0's second gotcha applies to it too: `sensitive = true` hides it from your terminal, it doesn't encrypt it in `terraform.tfstate`.

## Section 8 — `variables.tf` — Input Schema and Secrets

**The problem it solves.** Every value that should differ between "your Supabase project" and someone else's, between environments, or between a cautious and a permissive rate-limit posture, needs one typed, documented place — instead of being pasted directly into `main.tf`.

| Variable | Default | Sensitive? | Consumed by |
|---|---|---|---|
| `aws_region` | `ap-southeast-1` | No | `provider "aws"` ([main.tf:20-22](../main.tf#L20-L22)) |
| `instance_type` | `t3.micro` | No | `aws_instance.core_service` |
| `docker_image` | `sdewa/core-service-dep:latest` | No | `user_data`, matches `DEP_IMAGE_NAME` in [apps/core-service/Makefile:56](../../../apps/core-service/Makefile#L56) |
| `app_port` | `8080` | No | `user_data` only — no longer the security group (Section 5); internal-only, not reachable from the internet |
| `ssh_cidr_blocks` | `["0.0.0.0/0"]` | No | security group's SSH rule (see Section 5's callout) |
| `app_cidr_blocks` | `["0.0.0.0/0"]` | No | security group's nginx-port rule — gates `nginx_port`, not `app_port` |
| `nginx_image` | `nginx:1.27-alpine` | No | `user_data`'s nginx container |
| `nginx_port` | `80` | No | security group, `user_data`, `nginx.conf.tpl`, `outputs.tf`'s `app_url`/`app_swagger_url` |
| `nginx_rate_limit_rps` | `10` | No | `nginx.conf.tpl`'s `limit_req_zone` rate |
| `nginx_rate_limit_burst` | `20` | No | `nginx.conf.tpl`'s `limit_req` burst |
| `db_driver` | `postgres` | No | `user_data`'s `DB_DRIVER` |
| `db_source` | *(none — required)* | **Yes** | `user_data`'s `DB_SOURCE` |
| `jwt_secret_key` | *(none — required)* | **Yes**, validated ≥32 chars | `user_data`'s `JWT_SECRET_KEY` |
| `jwt_access_token_duration` | `60m` | No | `user_data`'s `JWT_ACCESS_TOKEN_DURATION` |
| `cors_allowed_origins` | `""` | No | `user_data`'s `CORS_ALLOWED_ORIGINS` |
| `app_rate_limit_enabled` | `true` | No | `user_data`'s `RATE_LIMIT_ENABLED` |
| `app_rate_limit_rps` | `5` | No | `user_data`'s `RATE_LIMIT_REQUESTS_PER_SECOND` |
| `app_rate_limit_burst` | `20` | No | `user_data`'s `RATE_LIMIT_BURST` |
| `log_level` | `info` | No | Declared, but not currently threaded into `user_data`'s `templatefile()` call (Section 6) — has no effect on the deployed instance today |
| `log_format` | `json` | No | Same as `log_level` — declared but currently unused by `user_data` |

**The validation block**, worth showing on its own ([variables.tf:82-85](../variables.tf#L82-L85)):

```hcl
validation {
  condition     = length(var.jwt_secret_key) >= 32
  error_message = "jwt_secret_key must be at least 32 characters."
}
```

This mirrors the exact check in [internal/token/jwt_maker.go:11,20-23](../../../apps/core-service/internal/token/jwt_maker.go#L11) (`minSecretKeySize = 32`) — the goal is to fail at `terraform plan` time, before any AWS resources are touched, rather than have the container start and crash on the instance where the failure is harder to see.

**If you're new to Terraform:** `db_source` and `jwt_secret_key` have no `default` and are marked `sensitive = true`. No default means `terraform plan`/`apply` will refuse to run until you supply them (via `terraform.tfvars`, a `-var` flag, or an interactive prompt). `sensitive = true` means Terraform prints `(sensitive value)` instead of the real string in plan/apply output and in `terraform show` — but as flagged in Section 0, this is a display-only redaction, not encryption; the real value is still in `terraform.tfstate` (and now, also, in the `rendered_user_data` output — Section 9).

**How you're expected to supply them:** [terraform.tfvars.example](../terraform.tfvars.example) is the checked-in template. You copy it to `terraform.tfvars` (gitignored — see the root `.gitignore` entries: `*.tfvars` with a `!*.tfvars.example` exception) and fill in your real Supabase connection string and a real JWT secret. All of the nginx/rate-limit variables have working defaults and are commented out in the example file — uncomment only the ones you want to override. Nobody else with this repo gets your secrets unless you hand them `terraform.tfvars` directly.

## Section 9 — `outputs.tf` — Result Surface

**The problem it solves.** After `terraform apply` finishes, you need the instance's public IP (and now, the rendered boot script) to actually work with it — outputs are how Terraform surfaces values computed only after resources exist.

| Output | Value | Purpose |
|---|---|---|
| `instance_id` | `aws_instance.core_service.id` | Reference for AWS CLI/Console lookups |
| `public_ip` | `aws_instance.core_service.public_ip` | Raw IP |
| `app_url` | `http://<public_ip>:<nginx_port>` | Ready-to-curl URL, through nginx (was `:<app_port>` before this revision) |
| `app_swagger_url` | `http://<public_ip>:<nginx_port>/swagger/index.html#/` | Swagger UI, through nginx |
| `ssh_command` | `ssh -i <pem path> ec2-user@<public_ip>` | Copy-pasteable SSH command, using the exact `.pem` path written by [main.tf:66-70](../main.tf#L66-L70) |
| `rendered_user_data` | `local.user_data` | The exact first-boot script `aws_instance.core_service` was given (Section 6/7). **Sensitive** — embeds `db_source`/`jwt_secret_key` in plaintext, same caveat as the existing plaintext-secrets rough edge. `make tf-redeploy` (see [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md)) pipes this over SSH to bring an already-running instance in line with the current config |

**Rough edge worth flagging — the public IP is not static.** A plain `aws_instance` gets a new public IP any time it's stopped and restarted (not on every `apply`, but on a real reboot/stop-start cycle) unless you attach an Elastic IP, which this config does not do. For a learning deployment this is fine; if you start relying on a fixed address (e.g. pointing a domain at it), that's a gap to close later.

## Section 10 — Supporting Files (No Logic)

Files that exist alongside the `.tf` files but aren't themselves executed logic:

- [terraform.tfvars.example](../terraform.tfvars.example) — a template for the real `terraform.tfvars`, with placeholder values and inline comments (e.g. "use the pooler host for EC2" for Supabase, and which nginx/rate-limit variables have working defaults). Terraform never reads the `.example` file directly; you copy it.
- `.terraform.lock.hcl` (created by `terraform init`, not committed by hand but meant to be committed) — pins the exact provider builds resolved during this doc's verification (`aws` 6.64.0, `tls` 4.4.1, `local` 2.9.1).

[nginx.conf.tpl](../nginx.conf.tpl) is *not* in this list — unlike the two files above, it **is** read by Terraform itself, via `templatefile()` in the `locals` block covered in Section 6. Treat it the same way you'd treat `user_data.sh.tpl`: a template Terraform renders, not a static supporting file.

## Cross-Feature Coupling

- **This config never builds or pushes the core-service Docker image.** That happens entirely outside Terraform, via `make docker-dep-build` and `make docker-dep-push` ([apps/core-service/Makefile:60-64](../../../apps/core-service/Makefile#L60-L64)), which build from [Dockerfile.prod](../../../apps/core-service/Dockerfile.prod) and push to the `sdewa/core-service-dep` Docker Hub repo. The nginx image, by contrast, is a public, unmodified image (`nginx:1.27-alpine` by default) — nothing in this repo builds or pushes it; `user_data.sh.tpl` just `docker pull`s it straight from Docker Hub. If you change core-service's application code, you must rebuild and push its image *and* deal with the Section 7 "runs once automatically" gotcha (via `make tf-redeploy`, or destroy/recreate) to get the new image onto a running instance — `terraform apply` alone does not pick up new app code.
- **This config never runs database migrations.** The image built by [Dockerfile.prod:1-19](../../../apps/core-service/Dockerfile.prod#L1-L19) only contains the compiled server binary (`ENTRYPOINT ["/app/main"]`) plus the migration SQL files for reference — the separate migration CLI (`cmd/migration`, invoked via `make migrate-up` in [apps/core-service/Makefile](../../../apps/core-service/Makefile)) is not part of the image and is never invoked by `user_data`. Before the API on the instance can serve any request that touches the database, you need to run `go run ./cmd/migration/main.go up` from your own machine with `DB_SOURCE` pointed at the same Supabase URL you put in `terraform.tfvars`. This is easy to miss because the containers will start and look healthy — requests just fail at the query layer.
- **There are now three independent, unsynchronized places app config can live**, not two: [apps/core-service/app.env](../../../apps/core-service/app.env) (your local `docker-compose` Postgres, used by `make server`/`make docker-run`), `terraform.tfvars` (points at Supabase, feeds the render in Section 6), and `/opt/core-service/app.env` *on the instance itself*, written fresh by `user_data.sh.tpl`/`tf-redeploy` every time it runs from whatever `terraform.tfvars` currently says. Editing one never affects the others — worth knowing so you don't go looking for a shared config file that doesn't exist, and so you don't assume an SSH edit to the instance's `app.env` will survive the next `tf-redeploy` (it won't; that file gets fully overwritten).

## Summary — Data Flow

**Provisioning (`terraform apply`):**
```
terraform apply
  → reads terraform.tfvars for db_source, jwt_secret_key (+ any overrides)
  → data lookups resolve: default VPC, default subnet, latest AL2023 AMI
  → tls_private_key generates an RSA keypair
  → aws_key_pair registers the public half with AWS; local_file writes the private half to core-service-key.pem
  → aws_security_group opens ports 22 (SSH) and nginx_port (80) — app_port is not opened
  → local.nginx_conf renders nginx.conf.tpl with nginx_port/app_port/rate-limit values
  → local.user_data renders user_data.sh.tpl with 13 substituted values, embedding base64encode(local.nginx_conf)
  → aws_instance.core_service is created with local.user_data as its user_data
  → outputs.tf prints instance_id, public_ip, app_url, app_swagger_url, ssh_command (rendered_user_data is sensitive, so it's hidden unless you ask for it explicitly)
```

**First boot (on the EC2 instance, no human involved):**
```
instance boots
  → cloud-init runs user_data as root
  → dnf installs and starts Docker; docker network create core-service-net
  → docker pull <docker_image> and <nginx_image> from Docker Hub
  → /opt/core-service/app.env written (9 keys), chmod 600
  → core-service container started on core-service-net, bound to 127.0.0.1:app_port only
  → main binary starts, calls config.LoadConfig(".") → finds the mounted app.env → binds those values
  → connectDB(cfg) opens a connection to your Supabase Postgres using DB_SOURCE
  → /opt/nginx/default.conf written by base64-decoding the rendered nginx config
  → nginx container started on core-service-net, publishing nginx_port to the instance's public interface
```

**Request path (after boot):**
```
client → http://<public_ip>:<nginx_port>/...
  → security group allows it in on nginx_port only (Section 5) — app_port is not reachable from outside at all
  → nginx applies limit_req (edge rate limit, per client IP) — 429 here if exceeded
  → nginx proxy_passes to http://core-service:<app_port> over the core-service-net docker network
  → core-service's own in-process rate limiter applies (if RATE_LIMIT_ENABLED) — 429 here if exceeded
  → container's Gin router handles it (see apps/core-service/docs/GIN_HTTP_LAYER.md)
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
| `local_file.private_key` | resource | Write the private key to `infra/terraform/core-service-key.pem` |
| `aws_security_group.core_service` | resource | Firewall: open 22 (SSH) and `nginx_port` (nginx), allow all egress — `app_port` is not opened |
| `aws_instance.core_service` | resource | The EC2 instance itself, running core-service and nginx as two Docker containers via `user_data` |
| `local.nginx_conf` | local value | Rendered [nginx.conf.tpl](../nginx.conf.tpl) |
| `local.user_data` | local value | Rendered [user_data.sh.tpl](../user_data.sh.tpl); also exposed as the `rendered_user_data` output |

**Commands you'll actually run:** these now exist as `make tf-*` targets at the repo root — see [TERRAFORM_MAKE_COMMANDS.md](TERRAFORM_MAKE_COMMANDS.md) for the day-to-day reference. For direct `terraform` invocations (e.g. flags the Makefile wrappers don't expose), run them from `infra/terraform/` or with `-chdir=infra/terraform` from the repo root:

| Command | Effect |
|---|---|
| `terraform init` | Downloads the three providers, writes `.terraform.lock.hcl` |
| `terraform plan` | Shows what would change, without changing anything |
| `terraform apply` | Actually creates/updates the resources |
| `terraform destroy` | Tears everything down — including the EC2 instance and security group. Does **not** touch Supabase, since that's outside this config entirely |
