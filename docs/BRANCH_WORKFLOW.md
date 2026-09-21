# Branch Workflow — How Code Moves Through Branches

## Who this is for

You know Git basics (branch, commit, pull request) and have written code for
about a year. You don't need to know GitHub Actions or Terraform to
understand this doc — it's only about branches and pull requests (PRs).

## The rule, in one picture

```
epic/your-feature  →  dev  →  main
```

You always work on a branch named `epic/something` (for example
`epic/update_server`). When your feature is ready, you open a PR into `dev`.
Later, `dev` gets merged into `main`. That's it — three branches, one
direction.

## Why it's set up this way

- `epic/*` branches are where you actually write code for one feature.
- `dev` collects finished features before they go live.
- `main` is the branch that's actually running in production. Nothing goes
  live until it's on `main`.

## The robot that checks your PR

There's a GitHub Action file that checks this rule automatically:
[.github/workflows/pr-branch-rules.yml](../.github/workflows/pr-branch-rules.yml)

Every time you open (or update) a PR into `main` or `dev`, this check runs
and looks at two things: which branch you're merging **from** (called
`HEAD_REF`) and which branch you're merging **into** (called `BASE_REF`).

| You're opening a PR into... | It only allows a PR coming from... | Anything else |
|---|---|---|
| `main` | `dev` | ❌ fails with a red X |
| `dev` | a branch starting with `epic/` | ❌ fails with a red X |

So if you accidentally try to PR your `epic/foo` branch straight into `main`,
this check will fail and tell you why.

## Something worth knowing: this check can fail and still let you merge

This GitHub Action only puts a red ❌ or green ✅ on your PR — a "status
check." Whether GitHub actually **blocks** the merge button when it's red
depends on a separate setting in the repo (GitHub → Settings → Branches →
branch protection rules). That setting lives outside this repo's code, so it
can't be seen just by reading files. If it isn't turned on, a red check is
just a warning, not a hard stop.

Also: there is currently **no local hook** (nothing on your own computer)
that stops you from running `git push origin main` directly. The only thing
stopping a direct push to `main` is whatever branch protection is configured
on GitHub itself.

## Quick summary

1. Branch off as `epic/<name>` for your feature.
2. Open a PR: `epic/<name>` → `dev`. The bot checks your source branch starts
   with `epic/`.
3. Later, someone opens a PR: `dev` → `main`. The bot checks the source
   branch is exactly `dev`.
4. Merging into `main` is also the trigger that starts the deploy pipeline —
   see [CICD_PIPELINE.md](./CICD_PIPELINE.md) for what happens next.
