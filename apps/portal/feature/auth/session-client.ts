import { SESSION_COOKIE_NAME } from "./constants";

// Browser-side counterpart to session.ts's setSessionCookie: login/register
// are client-driven mutations now, not Server Actions, so there's no
// next/headers cookies() to write through. The cookie itself is not
// httpOnly (see session.ts), so document.cookie can set it directly with
// the same path/SameSite/Secure shape the server side uses.
export function setClientSessionCookie(token: string, maxAgeSeconds: number) {
  const secure = process.env.NODE_ENV === "production" ? "; Secure" : "";
  document.cookie = `${SESSION_COOKIE_NAME}=${encodeURIComponent(
    token,
  )}; path=/; max-age=${maxAgeSeconds}; SameSite=Lax${secure}`;
}
