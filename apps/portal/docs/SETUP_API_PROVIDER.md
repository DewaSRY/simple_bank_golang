# API Provider Setup

How this app talks to the backend: a single shared Axios instance, wrapped by
per-resource client classes, with cross-cutting concerns (auth, timezone,
build-safety) handled once in an interceptor.

Files:
- [lib/api/base-client.ts](../lib/api/base-client.ts) — the shared Axios
  instance and the `BaseClient` base class every API client extends.
- [lib/api/api-interceptor.ts](../lib/api/api-interceptor.ts) — request
  interceptor logic (auth header, timezone header, build-phase guard).
- [feature/auth/client.ts](../feature/auth/client.ts),
  [feature/account/client.ts](../feature/account/client.ts) — real
  `BaseClient` subclasses to copy the "Extending it" pattern from.

## Why this shape

Next.js App Router code runs in three different contexts — the build step,
the server (RSC / route handlers), and the browser — and each one has a
different way of reading the session token. If every API call site had to
know which context it was in, that logic would leak into every feature. So
it's centralized once:

```
Feature code
   │  calls a typed method, e.g. userClient.getProfile()
   ▼
Resource client (extends BaseClient)
   │  knows the endpoint shape, not the transport concerns
   ▼
BaseClient
   │  thin wrapper over axios.get/post/put/patch/delete
   ▼
apiClient (shared axios.create() instance)
   │  one instance, one interceptor attached
   ▼
ApiInterceptor
   │  injects Authorization + X-Timezone, blocks build-time requests
   ▼
Backend
```

Feature code never touches axios directly, and never has to think about
"am I on the server or the client right now?" — that question is answered
in exactly one place.

## `BaseClient`

`BaseClient` is not meant to be used directly — it's a base class that gives
every resource-specific client (`UserClient`, `OrderClient`, etc.) typed
`get`/`post`/`put`/`patch`/`delete` helpers over a shared `AxiosInstance`.

```ts
export const apiClient = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080/api/v1",
  headers: { "Content-Type": "application/json" },
  timeout: 10_000,
});
```

The `/api/v1` prefix is baked into the fallback default (and into
`NEXT_PUBLIC_API_URL` in `.env.local`) rather than repeated in every
endpoint string — see the "Missing `/api/v1` prefix" gotcha in
`SETUP_REACT_QUERY.md`.

`apiClient` is a module-level singleton — created once, imported everywhere.
That matters for the interceptor (see below): there is exactly one Axios
instance in the app unless a caller explicitly passes a different one.

Each protected method takes a single options object (`{ endpoint, body,
params, config }`) instead of positional args, so call sites read as intent
rather than as an argument-order puzzle, and adding a new option later never
becomes a breaking change:

```ts
protected post<TResponse = unknown>(
  options: RequestOptions,
): Promise<AxiosResponse<TResponse>>
```

### Attaching the interceptor exactly once

The constructor attaches `ApiInterceptor` to whatever instance it's given:

```ts
const interceptedInstances = new WeakSet<AxiosInstance>();

export class BaseClient {
  protected readonly instance: AxiosInstance;

  constructor(instance: AxiosInstance = apiClient) {
    this.instance = instance;

    if (!interceptedInstances.has(instance)) {
      new ApiInterceptor(instance);
      interceptedInstances.add(instance);
    }
  }
}
```

The `WeakSet` guard exists because `BaseClient` is subclassed once per
resource, but `apiClient` is shared by all of them. Without the guard, every
`new UserClient()`, `new OrderClient()`, etc. would call
`instance.interceptors.request.use(...)` again on the *same* instance —
stacking up N identical interceptors that each redundantly re-check auth and
timezone on every request. The guard makes "attach interceptor" idempotent
per Axios instance, so it doesn't matter how many resource clients exist or
in what order they're constructed.

### Extending it

Resource clients live next to the feature they belong to —
`feature/<name>/client.ts` (e.g. [feature/auth/client.ts](../feature/auth/client.ts),
[feature/account/client.ts](../feature/account/client.ts)) — not under
`lib/api/`, which only holds the transport-level pieces (`BaseClient`,
`ApiInterceptor`) shared by all of them:

```ts
// feature/user/client.ts
import { BaseClient } from "@/lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";

export type UserProfile = {
  id: string;
  name: string;
};

export class UserClient extends BaseClient {
  getProfile(userId: string) {
    return this.get<CommonSuccessResponse<UserProfile>>({
      endpoint: `/users/${userId}`,
    });
  }

  updateProfile(userId: string, body: Partial<UserProfile>) {
    return this.patch<CommonSuccessResponse<UserProfile>>({
      endpoint: `/users/${userId}`,
      body,
    });
  }
}

export const userClient = new UserClient();
```

Export a singleton instance per resource (as above), the same way `apiClient`
itself is a singleton — that's what lets the `WeakSet` guard do its job, and
it avoids reconstructing the client (and its typed methods) on every import.

Every method returns the raw `Promise<AxiosResponse<TResponse>>` — callers
(typically a feature's `hooks/query.ts`) unwrap it with
`.then((response) => response.data)` to get to `TResponse` itself. See
"Mutations: the auth feature" in `SETUP_REACT_QUERY.md` for why skipping
that unwrap is a type error, not a runtime bug.

Only pass a custom instance to the constructor when you deliberately need
isolation from the shared instance — e.g. a client for a *different* backend
that shouldn't get the session-cookie auth header at all. Anything hitting
the same API should go through the default `apiClient`.

## `ApiInterceptor`

`ApiInterceptor` wraps one `AxiosInstance` and registers a single request
interceptor that runs three checks, in order, before every request leaves:

```ts
this.instance.interceptors.request.use(async (config) => {
  if (this.isBuildPhase()) {
    throw new BuildPhaseSkippedError(config.url);
  }

  await this.addAuthorizationHeader(config);
  this.addClientTimezoneHeader(config);
  return config;
});
```

### 1. Build-phase guard

`next build` executes Server Components at build time to statically generate
pages. If a component fetches from your own backend during that phase, the
request either hits a backend that isn't running in the build environment or
bakes stale data into the static output. `isBuildPhase()` detects this via
`NEXT_PHASE === "phase-production-build"` and throws
`BuildPhaseSkippedError` before the request is even sent.

This is a signal, not a failure to swallow silently — callers (or a
higher-level fetch wrapper) can catch `BuildPhaseSkippedError` specifically
to render a fallback/loading state during static generation, while letting
any other error propagate normally.

### 2. Authorization header

The session token lives in a cookie, but *which* cookie API differs by
context:

- **Server** (RSC, route handlers): `next/headers`' `cookies()` is the only
  way to read cookies, and it's async.
- **Browser**: there's no `cookies()` API — the token is parsed out of
  `document.cookie` with a regex.

`isServer()` (`typeof window === "undefined"`) picks the right path. The
`try/catch` around `cookies()` isn't for a real auth failure — it's because
Next.js signals "this route can't be statically rendered" by *throwing* a
special error (`digest === "DYNAMIC_SERVER_USAGE"`) the moment `cookies()` is
called during static generation. That specific error is re-thrown so Next
can correctly bail the route out of static rendering; anything else calling
into `cookies()` unexpectedly failing is treated as "no token available"
rather than crashing the request, since a logged-out request is a perfectly
valid state (the backend will reject it with 401, not the interceptor).

### 3. Timezone header

`X-Timezone` is only set client-side (`Intl.DateTimeFormat()` needs the
browser/runtime's local timezone, which is meaningless to compute on the
server where the process' timezone isn't the *user's*). Server-rendered
requests simply omit the header; the backend should treat a missing
`X-Timezone` as "unknown" rather than assuming UTC or erroring.

## Adding a new cross-cutting concern

To add another interceptor behavior (e.g. request ID tracing, a
retry-on-401-refresh flow), add a private method to `ApiInterceptor` and call
it from `setupRequestInterceptors`, in the same style as
`addAuthorizationHeader`/`addClientTimezoneHeader`. Keep it here rather than
in a resource client — anything that should apply to *every* request belongs
in the interceptor, not duplicated per client.

For response-side concerns (e.g. centralized 401 → redirect-to-login, error
normalization), add a `setupResponseInterceptors()` method following the
same pattern and call it from the constructor alongside
`setupRequestInterceptors()`.
