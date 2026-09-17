<!-- BEGIN:nextjs-agent-rules -->

# This is NOT the Next.js you know

This version has breaking changes — APIs, conventions, and file structure may all differ from your training data. Read the relevant guide in `node_modules/next/dist/docs/` (resolved from this file's directory; in monorepos the `next` package may not be visible from the repo root) before writing any code. Heed deprecation notices.

This block is written and re-added by `next dev` — verify at `node_modules/next/dist/server/lib/generate-agent-files.js`. Removing it from a diff only re-creates the uncommitted change; committing it with your work keeps the tree clean.

<!-- END:nextjs-agent-rules -->

# Portal — Project Guide

Next.js 16 (App Router) frontend for **Simple Bank**, a ledger-based core
banking demo. It talks to the Go backend in `../core-service`. See the
[root README](../../README.md) for the monorepo shape.

## Orientation

- **`docs/`** — one `SETUP_*.md` per subsystem, written "as implemented"
  with file:line citations (API client layer, auth forms/route groups, i18n,
  React Query, React Hook Form, shadcn/theme, top progress bar, logging).
  Read the relevant one before re-deriving a pattern from scratch, and update
  it in the same change if you alter that subsystem's shape.
- **`docs/IMPROVEMENT_OPPORTUNITIES.md`** — living list of known gaps/tech
  debt, each cited to file:line. Check it before "fixing" something that
  looks wrong in passing — it may already be a tracked, deliberate tradeoff.
- **`docs/DOC_STRUCTURE.md`** — the template/style ("as implemented, not as
  designed", cite everything to file:line, call out rough edges explicitly)
  to follow when writing a new doc here.
- **`docs/archive/`** — completed one-off task prompts kept for history.
  Move a root-level scratch `*.md` there once its work ships, instead of
  leaving it at the repo root.

## Architecture at a glance

- `feature/<name>/` — one folder per domain (`account`, `account-manage`,
  `account-transaction`, `auth`, `transfer`, `common`): `client.ts` (an
  Axios resource client extending `lib/api/base-client.ts`'s `BaseClient`),
  `hooks/query.ts` (TanStack Query hooks + a `queryKeys` object), `type.ts`,
  and sometimes `schema.ts` (Zod) / `utils.ts`.
- `lib/api/` — transport only: the shared Axios instance (`base-client.ts`)
  and one request+response interceptor (`api-interceptor.ts`) that attaches
  auth/timezone headers, logs every request/response via `lib/logger.ts`,
  and redirects to `/logout` on a 401. See `docs/SETUP_API_PROVIDER.md` and
  `docs/SETUP_LOGGING.md`.
- `components/` is organized by feature/domain, not by type. Multi-step
  modals (`deposite-model`, `transfer-modal`, `create-account-model`,
  `edit-account-model`, `delete-account-model`) each own a local Zustand
  `store.ts` for wizard-step state — Zustand is for local UI state only;
  server data stays in TanStack Query.
- `app/[locale]/(public)` vs `app/[locale]/(protected)` route groups split
  logged-out routes (landing, login, register, logout) from logged-in ones
  (dashboard, account detail). `proxy.ts` (this project's middleware —
  check `node_modules/next/dist/docs/` if that naming looks unfamiliar)
  handles the locale prefix + auth redirect at the edge; `feature/auth/dal.ts`'s
  `verifySession()` re-checks server-side in the protected layout.
- i18n: `react-i18next`, locales `en`/`id` (`i18n/settings.ts`), one JSON
  file per namespace under `messages/<locale>/<namespace>.json` (`common`,
  `auth`, `account`, `deposit`, `transfer`). Always add new user-facing
  strings through a translation key — the account-detail flow shipped
  hardcoded English once and had to be retrofitted (see
  `docs/IMPROVEMENT_OPPORTUNITIES.md` history).
- Design system: shadcn/ui components in `components/ui/` are owned source
  (edit directly, not a vendored dependency) on Base UI primitives +
  Tailwind v4 + `next-themes`. Brand color tokens (`--brand-*`, a
  yellow→lime→green spectrum in `app/globals.css`) are kept separate from
  theme tokens (`--background`/`--surface`/etc.) so brand identity stays
  constant across light/dark. See `docs/SETUP_SHADCN_THEME.md`.

## Conventions to know before editing

- **`queryKeys` name collision**: `feature/account`, `feature/account-manage`,
  and `feature/account-transaction` each export a plain `queryKeys` binding,
  so any call site needing more than one aliases the import
  (`import { queryKeys as accountQueryKeys } from ...`). Only `feature/auth`
  uses a namespaced `authQueryKeys`. Follow the alias-import pattern at new
  call sites — see `docs/IMPROVEMENT_OPPORTUNITIES.md` §2.2 for the proposed
  rename before doing a repo-wide one yourself.
- Known typos that are already public API — grep before renaming, and
  prefer fixing over copying: `AccountTranscationClient`
  (`feature/account-transaction/client.ts`), `RequestAccountbody`
  (`feature/account/type.ts`).
- Two API-error-shape types exist: `lib/api/types.ts`'s `ApiErrorResponse`
  (the one actually used, by `lib/api/error.ts`) and `feature/common/type.ts`'s
  `ErrorResponse` (unused). Use `ApiErrorResponse`; don't add a third.
- The session token cookie is deliberately **not httpOnly** — comments in
  `feature/auth/session.ts` / `session-client.ts` explain why (client code
  needs to read it for the browser-side request path). This is an accepted
  tradeoff, not a bug to fix in passing — read
  `docs/IMPROVEMENT_OPPORTUNITIES.md` §1.1 first if you touch auth cookies.
- No test suite and no CI for this app yet (`apps/core-service` has CI via
  `.github/workflows/core-service-ci.yml`; portal has none). Verify changes
  with `yarn lint`, `yarn tsc --noEmit`, and `yarn build`.
- `.env.local` must set `NEXT_PUBLIC_API_URL` (falls back to
  `http://localhost:8080/api/v1`), pointing at `apps/core-service`.
