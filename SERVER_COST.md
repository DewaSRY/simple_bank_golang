# Server Cost — How to Count It

## Who this doc is for

You, working out what this project actually costs to run and how to re-check
that number later when something changes (bigger instance, a second
environment, more traffic). It covers two things: the general method for
counting server cost, and the concrete numbers for _this_ app's current
architecture (single EC2 instance + Supabase + Vercel).

Prices below are illustrative, pulled from public list pricing at the time of
writing (September 2026). AWS, Supabase, and Vercel all change pricing and
this project's `ap-southeast-1` region has its own rates — before trusting a
number, regenerate it with the tools in
[Section 3](#section-3--where-to-get-live-numbers), don't copy the figure out
of this doc.

## Section 0 — What "server cost" actually means here

This app is not one bill. It's three separate billing relationships, each
counted a different way:

| Piece                             | Runs on            | Who bills it |
| --------------------------------- | ------------------ | ------------ |
| `core-service` (Go API) + `nginx` | 1 AWS EC2 instance | AWS          |
| PostgreSQL                        | Supabase (managed) | Supabase     |
| `portal` (Next.js)                | Vercel             | Vercel       |
| Docker image hosting              | Docker Hub         | Docker Hub   |
| CI (build/test/deploy)            | GitHub Actions     | GitHub       |

"Count the server cost" really means: add up the recurring charge from each
row, then add the variable charges (data transfer, extra compute-hours,
overage) that only show up once you have real traffic.

## Section 1 — The general method

For any one billed resource (an EC2 instance, a database, a CDN), cost breaks
into three components. Missing one of these is the most common way an
estimate turns out wrong later:

1. **Base / reserved cost** — what you pay just for the resource existing,
   independent of usage. An EC2 instance's hourly rate × hours running, a
   Supabase project's monthly plan fee, a Vercel seat.
2. **Usage-based cost** — scales with what actually happens: GB of data
   transferred out to the internet, GB-hours of storage, number of function
   invocations, CI minutes consumed.
3. **Step costs** — costs that are flat right up until you cross a threshold,
   then jump: the free tier ending, a plan's included bandwidth being
   exceeded, an instance type no longer being big enough so you resize to the
   next size up.

The formula for any one resource over a month:

```
monthly_cost = base_cost + (usage_quantity × usage_rate) + step_cost_if_crossed
```

And the project total is just the sum of that across every row in the table
above. The rest of this doc fills that formula in for each row as it exists
today.

## Section 2 — This app's actual numbers

### 2.1 EC2 — `core-service` + `nginx`

From [infra/terraform/variables.tf](infra/terraform/variables.tf):

- Region: `ap-southeast-1` (Singapore) — `var.aws_region`
- Instance type: `t3.micro` — `var.instance_type`
- Count: 1 instance, always on (`restart: unless-stopped` in
  [docker-compose.prod.yaml](infra/terraform/docker-compose.prod.yaml), not
  scheduled to stop)
- No EBS volume is declared beyond the AMI's default root volume (8 GiB
  gp3 for AL2023), no Elastic IP, no load balancer, no RDS — Postgres lives
  on Supabase instead, so there's no separate AWS database charge.

Base cost = `t3.micro` on-demand hourly rate × ~730 hours/month, plus the
root EBS volume's GB-month rate. There is no reserved instance or savings
plan configured (`main.tf` provisions on-demand only), so this is the most
expensive way to run it — switching to a 1-year no-upfront reserved instance
or a Savings Plan is the first lever if this becomes a real recurring cost.

Usage-based: outbound data transfer from the instance to the internet (nginx
responses to clients). Inbound is free; the first ~100 GB/month outbound is
free across the account, then billed per GB — negligible for a
learning-project's traffic, worth checking once there's real usage.

Step cost: AWS's free tier includes 750 hours/month of `t2.micro`/`t3.micro`
(varies by account type and region) for the account's first 12 months. If
this AWS account is past its first year, or the free tier doesn't apply in
`ap-southeast-1` for this instance family, the full on-demand rate applies
from hour one.

### 2.2 Supabase — PostgreSQL

From [infra/terraform/terraform.tfvars.example](infra/terraform/terraform.tfvars.example),
`db_source` points at a Supabase connection string — Postgres is not
self-hosted on the EC2 instance in production (it only runs in Docker for
local dev, per [docker-compose.yaml](docker-compose.yaml)).

Base cost: whatever Supabase plan the project is on (Free or Pro at the time
of writing — check the project's own dashboard, this repo has no file that
records which plan was picked). Usage-based: database size, egress, and
(on Free) the project pausing after a period of inactivity, which is a
step cost worth knowing about if uptime matters — a paused free project
needs a manual un-pause before `core-service` can reach it again.

### 2.3 Vercel — `portal`

Base cost: Hobby (free) or a paid team plan, again not recorded in this repo
— check the Vercel dashboard the project is under. Usage-based: build
minutes, bandwidth, and (on paid plans) serverless/edge function invocations
if `portal` uses any Next.js server routes beyond static pages.

### 2.4 Docker Hub

The image `sdewa/core-service-dep` ([variables.tf](infra/terraform/variables.tf)
default) is pulled from Docker Hub by the EC2 instance on redeploy
([TERRAFORM_MAKE_COMMANDS.md](infra/terraform/docs/TERRAFORM_MAKE_COMMANDS.md)).
Free for a public repo with reasonable pull rates; a private repo or high
pull volume would move this onto a paid plan.

### 2.5 GitHub Actions

CI/CD ([infra/terraform/docs/ci_cd_implementation.md](infra/terraform/docs/ci_cd_implementation.md))
runs on GitHub-hosted runners. Free minutes/month apply to public repos
unconditionally and to private repos up to a monthly quota by plan; beyond
that it's billed per minute, weighted by runner OS (Linux cheapest, Windows
and macOS cost multiples more per minute — this project's workflows should
stay on Linux runners to keep this at or near $0).

## Section 3 — Where to get live numbers

Don't hand-compute EC2/RDS rates from memory — pricing varies by region and
changes over time. Use these instead, in order of how closely they match
this project's actual setup:

1. **AWS Pricing Calculator** (https://calculator.aws) — build an estimate
   for exactly `t3.micro` in `ap-southeast-1`, on-demand, plus the root EBS
   volume. This is the authoritative source for Section 2.1's numbers.
2. **AWS Cost Explorer**, once the instance has been running a month or more
   — shows what was _actually_ billed, which also catches anything the
   calculator estimate misses (e.g. a data-transfer spike).
3. **Supabase / Vercel dashboards** — each project's billing page shows the
   current plan and this month's usage against it directly; there's no need
   to estimate these from public pricing pages once the project exists.
4. **GitHub → Settings → Billing** — Actions minutes used this cycle, against
   the plan's included quota.

## Section 4 — A worked estimate (fill in current rates from Section 3)

Use this shape to produce one total. Replace the `?` with the number pulled
from Section 3 at the time you're estimating:

```
EC2 (t3.micro, ap-southeast-1, on-demand, ~730 hrs)   $?/mo
EC2 root EBS volume (8 GiB gp3)                       $?/mo
EC2 outbound data transfer (est. GB × $/GB)            $?/mo
Supabase plan                                          $?/mo
Vercel plan                                             $?/mo
Docker Hub                                              $0 (public repo)
GitHub Actions (within free minutes)                    $0
-----------------------------------------------------------
Total                                                   $?/mo
```

If the free-tier EC2 hours still apply to this AWS account, the EC2 row is
$0 until that 12-month window ends — worth noting the account's creation
date so this doesn't silently jump later.

## Section 5 — Levers, if cost needs to come down or scale needs to go up

- **Reserved Instance / Savings Plan** on the EC2 instance once usage is
  steady — biggest single lever, since it's currently 100% on-demand.
- **Stop-on-schedule** for a non-production EC2 instance (e.g. a staging
  copy) if one gets added later — on-demand billing is per-second, so an
  instance only needs to run while someone's using it.
- **Supabase/Vercel plan tier** — both scale in discrete steps; check the
  dashboard before assuming an upgrade is needed versus trimming usage
  (unused DB branches, stale preview deployments) first.
- **Data transfer** — nginx already sits in front of `core-service`
  ([README.md](README.md), Infrastructure and deployment) partly as a rate
  limiter; it also caps how much abusive traffic can turn into an outbound
  data-transfer bill.
