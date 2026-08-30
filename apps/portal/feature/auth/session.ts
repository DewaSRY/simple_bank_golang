import { cookies } from "next/headers";
import { SESSION_COOKIE_NAME } from "./constants";

// Not httpOnly: lib/api/api-interceptor.ts reads this cookie via
// `document.cookie` for client-side requests, so browser JS must see it.
// Login/register write it client-side instead — see session-client.ts.
export async function clearSessionCookie() {
  const cookieStore = await cookies();
  cookieStore.delete(SESSION_COOKIE_NAME);
}
