# TanStack Query Setup

This app uses [`@tanstack/react-query`](https://tanstack.com/query) for
client-side data fetching: caching, background refetch, and a browser cache
that's shared across components — on top of (not instead of) Server
Component data fetching.

Files:
- [providers/query-provider.tsx](../providers/query-provider.tsx) — the
  `QueryClientProvider`, mounted once in the root layout.
- [feature/account/hooks/query.ts](../feature/account/hooks/query.ts) — query
  keys + the `useAccounts` hook, the pattern to copy for a new feature.
- [feature/account/account-list.tsx](../feature/account/account-list.tsx) —
  a Client Component consuming that hook.
- [app/[locale]/dashboard/page.tsx](../app/%5Blocale%5D/dashboard/page.tsx) —
  a Server Component that prefetches and hands off via `HydrationBoundary`.

## Why this shape

The dashboard's accounts list used to be fetched entirely in the Server
Component (`accountClient.listAccounts()` inside `page.tsx`, `try`/`catch`
around it). That's simple, but it means the data can never be refetched,
cached, or invalidated without a full page reload — every account mutation
elsewhere in the app would need its own bespoke "reload the page" logic.

TanStack Query is the same shared browser cache used across `SETUP_API_PROVIDER.md`'s
API client layer, but one level up: instead of every component re-fetching
independently, components subscribing to the same query key share one
in-flight request and one cached result, and any of them can trigger a
refetch (mutation `onSuccess`, window refocus, a manual "refresh" button)
that every subscriber picks up.

The Server Component still owns the *initial* fetch — this app doesn't drop
that just to add a client library:

```
Server Component (dashboard/page.tsx)
   │  queryClient.prefetchQuery(...) — same accountClient call as before
   ▼
dehydrate(queryClient) → <HydrationBoundary state={...}>
   │  ships the already-fetched data down in the initial HTML/RSC payload
   ▼
Client Component (AccountList)
   │  useAccounts() reads the hydrated cache — no loading flash on first paint
   ▼
TanStack Query
   │  now owns the browser copy: refetch on focus, manual invalidation, etc.
   ▼
accountClient.listAccounts() (unchanged)
```

The `Account` type, `accountClient`, and the interceptor that attaches the
auth header are untouched — TanStack Query wraps the existing API layer, it
doesn't replace it.

## Step 1: Dependencies

Already present in `package.json`:

```bash
yarn add @tanstack/react-query @tanstack/react-query-devtools
```

## Step 2: The provider

```tsx
// providers/query-provider.tsx
"use client";

import type { ReactNode } from "react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { ReactQueryDevtools } from "@tanstack/react-query-devtools";

let browserQueryClient: QueryClient | undefined;

function getQueryClient() {
  if (typeof window === "undefined") return new QueryClient();
  browserQueryClient ??= new QueryClient();
  return browserQueryClient;
}

export function QueryProvider({ children }: { children: ReactNode }) {
  const queryClient = getQueryClient();

  return (
    <QueryClientProvider client={queryClient}>
      {children}
      <ReactQueryDevtools initialIsOpen={false} />
    </QueryClientProvider>
  );
}
```

A fresh `QueryClient` per server render keeps requests isolated between
users; the module-level `browserQueryClient` singleton keeps one cache alive
across client re-renders in the browser. This is the same
server-vs-browser split `BaseClient`/`ApiInterceptor` already do for the
Axios instance — see `SETUP_API_PROVIDER.md`.

`ReactQueryDevtools` no-ops itself out of production bundles internally
(gated on `NODE_ENV`), so it's safe to render unconditionally rather than
wrapping it in an env check.

Mounted once, above `NextIntlClientProvider`'s children, in
`app/[locale]/layout.tsx`:

```tsx
<NextIntlClientProvider messages={messages}>
  <QueryProvider>{children}</QueryProvider>
</NextIntlClientProvider>
```

## Step 3: Query keys + hook, per feature

Each feature that fetches data gets a `feature/<name>/hooks/query.ts`
exporting a query-key factory and the hook(s) built on it:

```ts
// feature/account/hooks/query.ts
import { useQuery } from "@tanstack/react-query";
import {
  accountClient,
  type Account,
  type ListAccountsParams,
} from "@/lib/api/clients/account-client";

export const accountKeys = {
  all: ["accounts"] as const,
  list: (params: ListAccountsParams) =>
    [...accountKeys.all, "list", params] as const,
};

export function fetchAccounts(
  params: ListAccountsParams = {},
): Promise<Account[]> {
  return accountClient
    .listAccounts(params)
    .then((response) => response.data.data);
}

export function useAccounts(params: ListAccountsParams = {}) {
  return useQuery({
    queryKey: accountKeys.list(params),
    queryFn: () => fetchAccounts(params),
  });
}
```

`fetchAccounts` is exported on its own (not just wrapped in the hook)
because the Server Component needs the exact same function for
`prefetchQuery` — see Step 5. Keeping the key factory (`accountKeys`)
separate from the hook is what lets both the server prefetch and the client
hook agree on the same cache identity without duplicating the key shape.

## Step 4: The Client Component

```tsx
// feature/account/account-list.tsx
"use client";

import { useTranslations } from "next-intl";
import { getApiErrorMessage } from "@/lib/api/error";
import { useAccounts } from "./hooks/query";

export function AccountList() {
  const t = useTranslations("Common");
  const {
    data: accounts = [],
    error,
    isPending,
  } = useAccounts({ page: 1, limit: 10 });

  if (isPending) return <p>{t("loadingAccounts")}</p>;
  if (error) return <p>{getApiErrorMessage(error, t("loadAccountsError"))}</p>;
  if (accounts.length === 0) return <p>{t("noAccounts")}</p>;

  return (
    <ul>
      {accounts.map((account) => (
        <li key={account.id}>
          {account.owner} — {account.balance} {account.currency}
        </li>
      ))}
    </ul>
  );
}
```

This is a plain `useQuery` (not `useSuspenseQuery`): the component renders
its own loading/error state inline, matching the UX the page already had
before this change. No Suspense boundary or error boundary needed.

## Step 5: Prefetch from the Server Component

```tsx
// app/[locale]/dashboard/page.tsx
import { QueryClient, HydrationBoundary, dehydrate } from "@tanstack/react-query";
import { AccountList } from "@/feature/account/account-list";
import { accountKeys, fetchAccounts } from "@/feature/account/hooks/query";

export default async function DashboardPage({ params }: PageProps<"/[locale]/dashboard">) {
  // ...locale/translation setup...

  const accountParams = { page: 1, limit: 10 };
  const queryClient = new QueryClient();
  await queryClient.prefetchQuery({
    queryKey: accountKeys.list(accountParams),
    queryFn: () => fetchAccounts(accountParams),
  });

  return (
    // ...
    <HydrationBoundary state={dehydrate(queryClient)}>
      <AccountList />
    </HydrationBoundary>
    // ...
  );
}
```

`prefetchQuery` is **awaited** here (not fire-and-forget with a Suspense
fallback) so the first paint has the data already — same blocking behavior
the old inline `try`/`catch` fetch had. If `accountClient.listAccounts`
throws (e.g. a 401), `prefetchQuery` catches it internally and stores the
error in the query cache; `dehydrate` ships that error state down too, so
`AccountList`'s `error` branch handles it — no `try`/`catch` needed in the
page itself anymore.

The query key (`accountKeys.list(accountParams)`) must be built the exact
same way on both sides — that's why it's a shared function in
`hooks/query.ts` rather than an inline array literal in each call site. A
mismatched key (e.g. different param order or an extra field) means the
client mounts a *different* cache entry than the one that was hydrated, and
you get a loading flash instead of instant data.

## Adding a new feature's query hooks

1. Create `feature/<name>/hooks/query.ts`: a key factory + a plain
   `fetch<Name>` function + a `use<Name>` hook wrapping it in `useQuery`.
2. If the initial view needs the data (not just an on-interaction fetch),
   prefetch it in the owning Server Component with `queryClient.prefetchQuery`
   using the same key + fetch function, and wrap the Client Component in
   `<HydrationBoundary state={dehydrate(queryClient)}>`.
3. If it's purely interaction-driven (e.g. an autocomplete), skip the
   prefetch/hydration step entirely and just call the hook from a Client
   Component — `enabled` can gate it on user input.

## Gotchas hit while wiring this up

None of these are TanStack Query issues — they're gaps in how the portal
and `core-service` already talked to each other, which client-side fetching
was the first thing to actually exercise (the dashboard's fetch had always
run server-side until now, so a browser never made this request before):

- **Missing `/api/v1` prefix.** Every API client call
  (`authClient.login`, `authClient.register`, `accountClient.listAccounts`)
  was missing the `/api/v1` prefix that `core-service`'s router actually
  mounts every route under (see `apps/core-service/internal/api/router.go`).
  Fixed by baking the prefix into `NEXT_PUBLIC_API_URL`
  (`.env.local`) and the `BaseClient` fallback default, rather than
  prefixing every endpoint string individually.
- **CORS disabled by default.** `core-service`'s CORS middleware treats an
  empty `CORS_ALLOWED_ORIGINS` as "disable CORS entirely," which is correct
  for server-to-server calls but blocks any request made *from the
  browser* — exactly what client-side TanStack Query does. Fixed by setting
  `CORS_ALLOWED_ORIGINS=http://localhost:3000` on the `core-services`
  service in the repo-root `docker-compose.yaml` (the value already existed
  in `apps/core-service/app.env`, it just wasn't wired into the compose
  file the container actually runs from).
- **`X-Timezone` not in the CORS allowlist.** `ApiInterceptor` attaches a
  custom `X-Timezone` header on every client-side request (see
  `SETUP_API_PROVIDER.md`), but the CORS middleware's `AllowHeaders` only
  listed `Origin`, `Content-Type`, `Authorization` — so the preflight
  request failed before the real request was ever sent. Added
  `X-Timezone` to `AllowHeaders` in
  `apps/core-service/internal/api/server.go`.

If a future client-side call fails with a CORS or 404 error against a route
that works fine from `curl`, check these three first.

## Verification performed

- `yarn tsc --noEmit` — clean.
- `yarn lint` — clean.
- `yarn build` — succeeds.
- Registered a real user and created a real account against the running
  `core-service` + Postgres containers, then loaded `/en/dashboard` with a
  valid session cookie in a headless browser: the account rendered
  correctly from the server-hydrated cache, the client-side background
  refetch completed with **zero console errors**, and the React Query
  Devtools toggle rendered in the corner — confirming the full
  prefetch → hydrate → client-refetch loop works end-to-end, not just the
  initial server render.
