# Portal — Improvement Opportunities

_As observed in the codebase on 2026-09-11. Every claim below is cited to a
file, so it's greppable and easy to re-verify as the code moves._

This is a young, well-structured app (clean `feature/<name>/{client,hooks,type,schema}.ts`
layering, one Axios instance, a documented `docs/SETUP_*` series). The issues
below are the gaps between that intended shape and what's actually shipped —
not a rewrite list.

---

## 1. Security

### 1.1 Session token is readable/writable by any JS on the page

The auth token is stored in a **non-httpOnly** cookie by design:
[feature/auth/session.ts:4-6](../feature/auth/session.ts#L4-L6) and
[feature/auth/session-client.ts:6-13](../feature/auth/session-client.ts#L6-L13)
both say so explicitly — it has to be readable from `document.cookie` because
[lib/api/api-interceptor.ts:56-59](../lib/api/api-interceptor.ts#L56-L59)
reads it that way for client-side requests, and login/register write it from
the browser after a client-driven mutation
([login-form.tsx:32](../components/auth/login-form.tsx#L32)).

This is a real tradeoff, not an oversight, but it means any XSS anywhere in
the app (a rendered rich-text field, a compromised dependency, a bad `dangerouslySetInnerHTML`)
is a full session-token theft, not just a DOM defacement. Worth reconsidering
before this app handles anything higher-stakes than a demo bank:

- **Better**: route login/register through a Next.js Route Handler that sets
  the cookie server-side as `httpOnly`, and have the server attach the
  `Authorization` header itself (a BFF/proxy pattern) so client JS never
  needs to see the raw token at all.
- **If keeping this shape**: at minimum, add a CSP and shorten `expires_in`
  server-side, since the doc comment already accepts the risk but nothing
  currently mitigates it.

### 1.2 No response-side 401 handling — this is a known, documented gap

[lib/api/api-interceptor.ts](../lib/api/api-interceptor.ts) only implements
`setupRequestInterceptors()`. There is no response interceptor anywhere in
the app (`grep -rn "interceptors.response"` returns nothing), and the setup
doc calls this out directly as unbuilt:

> "For response-side concerns (e.g. centralized 401 → redirect-to-login,
> error normalization), add a `setupResponseInterceptors()` method..."
> — [docs/SETUP_API_PROVIDER.md:230-233](SETUP_API_PROVIDER.md#L230-L233)

Today, when a token expires mid-session, every query/mutation using it just
fails independently, and each screen shows its own generic error message
(e.g. "Unable to load account" in
[account-detail-view.tsx:38-45](../components/account/account-detail-view.tsx#L38-L45)).
There's no forced logout or redirect-to-login — the user is stuck on a page
that will never recover until they manually log out.
[proxy.ts](../proxy.ts) only checks that the session cookie *exists*
([proxy.ts:35-37](../proxy.ts#L35-L37)), not that the backend still
considers it valid, so this can't be caught upstream either.

**Fix**: implement `setupResponseInterceptors()` as the doc suggests — on a
401, clear the session cookie and redirect to `/login` (client-side via
`window.location`, or by having the interceptor call something equivalent to
`logoutAction`).

---

## 2. Dead / unfinished code

| Item | Evidence | Note |
|---|---|---|
| `feature/transfer/client.ts` | 0 bytes | Feature scaffolded, nothing implemented |
| `feature/transfer/hooks/query.ts` | 0 bytes | Same |
| `feature/account-transaction/schema.ts` | 0 bytes | No Zod schema backs `DepositRequestBody` |
| `useDeposit` hook | [account-transaction/hooks/query.ts:43-56](../feature/account-transaction/hooks/query.ts#L43-L56) | Zero call sites in `components/` or `app/` — the deposit API and hook exist, but there is no deposit form/UI anywhere |
| Stray TODO shipped in a type file | [feature/common/type.ts:34](../feature/common/type.ts#L34) — `// TODO: finishes this patter` | Typo aside, it signals an intentionally half-finished pattern (`PaginationParams`) that new code is copying without knowing it's unfinished |

These aren't bugs, but they're worth either finishing (transfer, deposit) or
deleting (empty stub files) — right now a new contributor can't tell "not
built yet" apart from "built but broken" just by reading the tree.

---

## 3. Duplication that will drift

### 3.1 Two competing "API error shape" types

- [lib/api/types.ts](../lib/api/types.ts): `ApiErrorResponse` / `ApiErrorBody` / `ApiFieldError` — this is the one actually used, by [lib/api/error.ts](../lib/api/error.ts).
- [feature/common/type.ts:12-22](../feature/common/type.ts#L12-L22): `ErrorResponse` / `ErrorDetail` — same shape, different names, **unused** (no import of either name found outside this file).

Pick one, delete the other — otherwise the next person to touch error
handling won't know which is canonical.

### 3.2 Non-namespaced `queryKeys` export, repeated per feature

`feature/account/hooks/query.ts`, `feature/account-manage/hooks/query.ts`,
and `feature/account-transaction/hooks/query.ts` **all** export a binding
literally named `queryKeys`. Only `feature/auth` breaks the pattern with
`authQueryKeys` ([auth/hooks/query.ts:14](../feature/auth/hooks/query.ts#L14)).

The consequence shows up at every call site that needs more than one of
them — they're forced to alias-import to avoid a collision:

```ts
// app/[locale]/(protected)/account/[id]/page.tsx:9,11
import { queryKeys as accountQueryKeys } from "@/feature/account-manage/hooks/query";
import { queryKeys as accountTransactionQueryKeys } from "@/feature/account-transaction/hooks/query";
```

Renaming each export to `accountQueryKeys`, `accountManageQueryKeys`,
`accountTransactionQueryKeys` at the source removes the aliasing burden from
every future call site, not just this one.

### 3.3 SSR prefetch duplicates the hook's `queryFn` by hand

[app/[locale]/(protected)/layout.tsx:31-34](../app/%5Blocale%5D/(protected)/layout.tsx#L31-L34)
and
[app/[locale]/(protected)/account/[id]/page.tsx:45-57](<../app/[locale]/(protected)/account/[id]/page.tsx#L45-L57>)
both re-type the exact `queryFn` body that already lives in
`useAccounts`/`useAccountEntries` in the corresponding `hooks/query.ts`,
instead of sharing one definition. TanStack Query's `queryOptions()` helper
exists for exactly this — export `accountsQueryOptions(params)` once, use it
in both the hook and the server-side `prefetchQuery` call, and a future
change to the query key or fetch shape can't silently drift between the two
copies.

### 3.4 Naming typos (cosmetic, but load-bearing once public)

- `AccountTranscationClient` — [feature/account-transaction/client.ts:12](../feature/account-transaction/client.ts#L12)
- `RequestAccountbody` — [feature/account/type.ts:29](../feature/account/type.ts#L29)

Both are exported and imported in several places already, so fixing them now
(vs. after more call sites accumulate) is cheaper.

---

## 4. i18n coverage is inconsistent

The app has real i18n infra — `react-i18next`, `messages/en/*.json` and
`messages/id/*.json`, and it's used correctly in places like
[account-list.tsx](../components/dashboard/account-list.tsx) (`useTranslation("common")`)
and the auth forms. But it's bypassed entirely in the account-detail flow:

- [account-detail-view.tsx:31-53](../components/account/account-detail-view.tsx#L31-L53) — hardcoded English: "Loading account", "Unable to load account", "Please try again in a moment.", "Account not found", "This account may have been removed or is unavailable.", "Back to dashboard".
- [feature/account-transaction/utils.ts:28-36](../feature/account-transaction/utils.ts#L28-L36) — `getEntryLabel()` hardcodes `"Deposit"`, `"Withdrawal"`, `"To "`, `"From "`, rendered directly in [account-entry-row.tsx:38](../components/account/account-entry-row.tsx#L38).

There is no `messages/{en,id}/account.json` namespace at all — the account
feature was built before/outside the i18n convention the rest of the app
follows. Since `id` (Bahasa Indonesia) is a first-class supported locale
here, this isn't a nice-to-have — it's a visible gap for a real chunk of
users on the account detail page specifically.

---

## 5. No automated verification

- **Zero test files** in the repo (no `*.test.*`/`*.spec.*`, no Vitest/Jest
  config). Several pieces here are pure functions begging for unit tests
  with no rendering/mocking needed: `formatAccountAmount`, `isIncomingEntry`,
  `getEntryLabel` ([feature/account-transaction/utils.ts](../feature/account-transaction/utils.ts)),
  `parseIntParam`/`parseArrayParam` ([feature/common/params.ts](../feature/common/params.ts)),
  and the Zod schemas.
- **No CI config** (`find . -iname "*.yml"` outside `node_modules` returns
  nothing) — `yarn lint` / `tsc --noEmit` / `next build` aren't enforced
  before merge, so the `strict: true` in [tsconfig.json](../tsconfig.json)
  can be silently defeated by a PR that doesn't type-check.

Given the existing `docs/SETUP_*` series is clearly written to onboard new
contributors quickly, a `docs/SETUP_TESTING.md` in the same style plus a
one-file GitHub Actions workflow (`lint` → `typecheck` → `build`) would fit
right in and catch regressions the docs alone can't.

---

## 6. Project README is still the create-next-app default

[README.md](../README.md) is unedited boilerplate — no mention of what this
app is, that `docs/` exists, the required `NEXT_PUBLIC_API_URL` env var
([.env.local](../.env.local)), or how it relates to the backend/core-service.
Given the actual onboarding material already lives in `docs/DOC_STRUCTURE.md`
and friends, the fix is small: replace the README with a short project
summary + a link into `docs/`, not a rewrite.

---

## Suggested order of attack

1. **1.2** (response interceptor / 401 handling) — small, isolated, fixes a
   real stuck-user bug, and the doc already tells you exactly where it goes.
2. **4** (i18n gaps in account-detail) — mechanical, high user-visible impact.
3. **3.1 / 3.2 / 3.4** — cheap renames/dedup, best done before more code
   imports the wrong name.
4. **2** — decide per stub: finish transfer/deposit, or delete the scaffolding.
5. **1.1**, **5**, **6** — larger or process-level changes; worth a
   deliberate discussion rather than a drive-by fix.
