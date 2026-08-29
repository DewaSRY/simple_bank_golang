import { cookies } from "next/headers";
import { SESSION_COOKIE_NAME } from "./constants";

export async function setSessionCookie(token: string, maxAgeSeconds: number) {
  const cookieStore = await cookies();

  // Not httpOnly: lib/api/api-interceptor.ts reads this cookie via
  // `document.cookie` for client-side requests, so browser JS must see it.
  cookieStore.set(SESSION_COOKIE_NAME, token, {
    secure: process.env.NODE_ENV === "production",
    sameSite: "lax",
    path: "/",
    maxAge: maxAgeSeconds,
  });
}

export async function clearSessionCookie() {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE_NAME);
}
