# Internationalization Setup

This app uses [`next-intl`](https://next-intl.dev) for i18n, with translations split **per feature** rather than one giant dictionary per language.

Supported locales: `en` (default), `id`.

## Important: this repo runs a modified Next.js (16.3.3)

Per `AGENTS.md`, this Next.js build has breaking changes vs. stock Next.js. The one that matters for i18n:

> `middleware.ts` is **deprecated and renamed to `proxy.ts`** (Next.js 16). The exported function must be named `proxy` (or be the default export) — `middleware.ts` is not picked up.

`next-intl`'s own docs still reference `middleware.ts`. We instead created `proxy.ts` at the project root and used the default export, which Next.js picks up as the proxy/middleware file. See [Step 4](#step-4-proxy-file-not-middleware).

## File structure

```text
i18n/
├── routing.ts       # locales, default locale
├── navigation.ts     # locale-aware Link, useRouter, usePathname, redirect
└── request.ts        # loads + merges per-feature message files per request

messages/
├── en/
│   ├── common.json
│   ├── auth.json
│   └── transfer.json
└── id/
    ├── common.json
    ├── auth.json
    └── transfer.json

proxy.ts               # locale detection + routing (replaces middleware.ts)
next.config.ts          # wrapped with createNextIntlPlugin

app/
├── globals.css
├── favicon.ico
└── [locale]/
    ├── layout.tsx     # root layout now lives here, validates locale, provides messages
    └── page.tsx
```

Because routing is locale-prefixed (`/en/...`, `/id/...`), every route in `app/` had to move under a `[locale]` dynamic segment. There is no top-level `app/layout.tsx` anymore — `app/[locale]/layout.tsx` is the effective root layout (it's the first layout with no other layout above it, so it still owns `<html>`/`<body>`).

## Step 1: Dependency

Already present in `package.json`:

```bash
yarn add next-intl
```

## Step 2: Translation files, split per feature

Each locale directory holds one JSON file per feature/namespace:

`messages/en/auth.json`
```json
{
  "login": "Login",
  "logout": "Logout",
  "email": "Email address",
  "password": "Password",
  "forgotPassword": "Forgot password?",
  "loginError": "Invalid email or password"
}
```

`messages/id/auth.json`
```json
{
  "login": "Masuk",
  "logout": "Keluar",
  "email": "Alamat email",
  "password": "Kata sandi",
  "forgotPassword": "Lupa kata sandi?",
  "loginError": "Email atau kata sandi salah"
}
```

Same pattern for `common.json` (shared UI strings) and `transfer.json` (the transfer feature). Ownership stays clean: a feature team only touches its own file, in both locales.

Rich text (bold, links, etc. embedded in a translated sentence) uses `next-intl`'s tag syntax, **not** plain `{placeholder}` interpolation — a placeholder can only hold a value (string/number), not a function/JSX. This mistake fails at render with `Functions are not valid as a child of Client Components`, which is what happened during initial implementation until the messages were switched to tags:

```json
// messages/en/common.json
"tagline": "To get started, edit the <file>page.tsx</file> file."
```

```tsx
t.rich("tagline", {
  file: (chunks) => <code>{chunks}</code>,
});
```

## Step 3: `i18n/routing.ts` — define locales once

```ts
import { defineRouting } from "next-intl/routing";

export const routing = defineRouting({
  locales: ["en", "id"],
  defaultLocale: "en",
});

export type AppLocale = (typeof routing.locales)[number];
```

`i18n/navigation.ts` wraps `next/link` and `next/navigation` so links and redirects automatically get the right locale prefix:

```ts
import { createNavigation } from "next-intl/navigation";
import { routing } from "./routing";

export const { Link, redirect, usePathname, useRouter, getPathname } =
  createNavigation(routing);
```

Use these (`@/i18n/navigation`) instead of `next/link` / `next/navigation` anywhere a link should stay in the current locale.

## Step 4: `i18n/request.ts` — load + merge per-feature files

```ts
import { hasLocale } from "next-intl";
import { getRequestConfig } from "next-intl/server";
import { routing } from "./routing";

export default getRequestConfig(async ({ requestLocale }) => {
  const requested = await requestLocale;
  const locale = hasLocale(routing.locales, requested)
    ? requested
    : routing.defaultLocale;

  const [common, auth, transfer] = await Promise.all([
    import(`../messages/${locale}/common.json`),
    import(`../messages/${locale}/auth.json`),
    import(`../messages/${locale}/transfer.json`),
  ]);

  return {
    locale,
    messages: {
      Common: common.default,
      Auth: auth.default,
      Transfer: transfer.default,
    },
  };
});
```

Translation files stay split on disk, but `next-intl` receives one merged object per request, namespaced by feature (`Common`, `Auth`, `Transfer`).

Adding a new feature = add `messages/{en,id}/<feature>.json` + one more entry here (see [Adding a new feature namespace](#adding-a-new-feature-namespace)).

## Step 5: Proxy file (not `middleware.ts`)

```ts
// proxy.ts (project root)
import createMiddleware from "next-intl/middleware";
import { routing } from "./i18n/routing";

export default createMiddleware(routing);

export const config = {
  matcher: ["/((?!api|_next|_vercel|.*\\..*).*)"],
};
```

`next-intl/middleware` is just the name of the package export — what matters to Next.js is the **file name** (`proxy.ts`) and that the middleware function is the **default export**, since Next 16 no longer recognizes a `middleware.ts` file convention. Confirmed in the build output:

```text
ƒ Proxy (Middleware)
```

This is what performs locale detection (`Accept-Language` header, `NEXT_LOCALE` cookie) and redirects `/` → `/en` (or `/id`), and rewrites `/en/...` internally so `app/[locale]` can render it.

## Step 6: Wire the plugin in `next.config.ts`

```ts
import type { NextConfig } from "next";
import createNextIntlPlugin from "next-intl/plugin";

const withNextIntl = createNextIntlPlugin("./i18n/request.ts");

const nextConfig: NextConfig = {
  /* config options here */
};

export default withNextIntl(nextConfig);
```

## Step 7: `app/[locale]/layout.tsx` — validate locale, provide messages

```tsx
import { hasLocale, NextIntlClientProvider } from "next-intl";
import { getMessages } from "next-intl/server";
import { notFound } from "next/navigation";
import { routing } from "@/i18n/routing";
import "../globals.css";

export function generateStaticParams() {
  return routing.locales.map((locale) => ({ locale }));
}

export default async function RootLayout({
  children,
  params,
}: LayoutProps<"/[locale]">) {
  const { locale } = await params;

  if (!hasLocale(routing.locales, locale)) {
    notFound();
  }

  const messages = await getMessages();

  return (
    <html lang={locale}>
      <body>
        <NextIntlClientProvider messages={messages}>
          {children}
        </NextIntlClientProvider>
      </body>
    </html>
  );
}
```

- `generateStaticParams` lets `/en` and `/id` be statically generated at build time.
- `hasLocale` + `notFound()` guards against an invalid `[locale]` segment (e.g. someone hitting `/fr` directly without going through the proxy).
- `NextIntlClientProvider` makes messages available to Client Components (`"use client"`) that call `useTranslations`. Server Components can just call `getTranslations` directly and don't need the provider.

## Step 8: Using translations

**Server Component** (default — preferred, zero client JS cost):

```tsx
import { getTranslations, setRequestLocale } from "next-intl/server";

export default async function Page({ params }: PageProps<"/[locale]">) {
  const { locale } = await params;
  setRequestLocale(locale);

  const t = await getTranslations("Common");
  return <h1>{t("appName")}</h1>;
}
```

**Client Component** (needed for interactivity, e.g. the locale switcher):

```tsx
"use client";
import { useTranslations, useLocale } from "next-intl";

export function LocaleSwitcher() {
  const t = useTranslations("Common");
  const locale = useLocale();
  // ...
}
```

A working example of both lives in `app/[locale]/page.tsx` (server) and `components/locale-switcher.tsx` (client, uses `@/i18n/navigation`'s `useRouter`/`usePathname` to switch locale while staying on the same page).

## Adding a new locale

1. Add the locale code to `routing.ts`'s `locales` array.
2. Create `messages/<locale>/{common,auth,transfer}.json` with translated values for every existing key (same keys as `en`).
3. No other code changes needed — `request.ts`, the proxy, and `generateStaticParams` all read from `routing.locales`.

## Adding a new feature namespace

1. Create `messages/en/<feature>.json` and `messages/id/<feature>.json`.
2. Add it to the `Promise.all` + `messages` object in `i18n/request.ts`, using a `PascalCase` namespace key (e.g. `Material`).
3. Consume with `useTranslations("Material")` / `getTranslations("Material")`.

Naming convention: keep feature file names singular or plural consistently (we use singular: `auth.json`, `transfer.json`).

## Troubleshooting: "Encountered a script tag while rendering React component"

This console warning showed up while building the locale switcher, even though no code in this repo writes a literal `<script>` tag. It's a symptom of a hydration break elsewhere in the tree, not a script tag you wrote:

1. `LocaleSwitcher` (`components/locale-switcher.tsx`) originally rendered a native `<select>`, but its options were `DropdownMenuItem`s from `@base-ui/react`'s `Menu` primitive — which render as `<div role="menuitem">`. A `<div>` is not valid inside `<select>`, so the browser logged `In HTML, <div> cannot be a child of <select>. This will cause a hydration error.`
2. That invalid nesting broke hydration for the page, which made Next.js Fast Refresh fall back to a client-only "full reload" of the React tree (`⚠ Fast Refresh had to perform a full reload due to a runtime error`) instead of a real server round-trip.
3. `providers/theme-provider.tsx` wraps `next-themes`, which always renders an inline `<script>` (`dangerouslySetInnerHTML`, no `type` attribute) as its no-flash-of-unstyled-theme mechanism. This is invisible during a normal hydration because it's part of the server-rendered HTML. But when React had to rebuild the tree purely on the client instead of hydrating server markup, it created that `<script>` node via `document.createElement` — and any `<script>` element created this way is inert (browsers only execute `<script>` tags present in the originally parsed HTML), so React's DOM renderer warns: "Encountered a script tag while rendering React component. Scripts inside React components are never executed when rendering on the client."

**Fix:** rewrite `LocaleSwitcher` to use the same `DropdownMenu`/`Button` primitives as `ThemeToggle` (`components/theme-toggle.tsx`) instead of a raw `<select>`. That removes the invalid `<select>` > `<div>` nesting, so hydration no longer breaks and Fast Refresh no longer has to force a client-only remount — which is what was surfacing the (otherwise harmless) `next-themes` script tag as a warning.

The lesson generalizes: if you see this specific warning without having written a `<script>` tag yourself, look for a hydration mismatch upstream (invalid HTML nesting is a common cause) rather than trying to silence the warning at its source.

## Verification performed

- `yarn tsc --noEmit` — clean.
- `yarn lint` — clean.
- `yarn build` — succeeds, `proxy.ts` correctly recognized as `ƒ Proxy (Middleware)` in route output.
- `yarn start` (production server) — `GET /` → `307` redirect to `/en`; `GET /en` and `GET /id` → `200` with correctly localized, rich-text-rendered content.
