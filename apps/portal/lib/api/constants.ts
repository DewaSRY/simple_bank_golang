// Cookie the browser writes on first render (see lib/timezone-sync.tsx) so
// lib/api/api-interceptor.ts can read the visitor's timezone server-side —
// every request now originates from the Next.js server, so there's no
// `document`/`Intl` available at request time to read it directly.
export const TIMEZONE_COOKIE_NAME = "x-timezone";
