"use server";

import { redirect } from "@/i18n/redirect";
import type { AppLocale } from "@/i18n/settings";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type { CommonSuccessResponse } from "@/feature/common";
import { authClient } from "./client";
import type { LoginRequest, ProfileResponse, RegisterRequest } from "./type";
import { clearSessionCookie, setSessionCookie } from "./session";

export async function logoutAction(locale: AppLocale): Promise<void> {
  await clearSessionCookie();
  return redirect({ href: "/login", locale });
}

// Returns ActionResult<null>, not the backend's AuthResponse: the session
// cookie is already set server-side (httpOnly) above, and nothing
// client-side reads the token (see feature/auth/session.ts) — forwarding
// the raw access_token across the Server Action boundary would only leave
// it sitting in the browser's React Query cache/devtools unnecessarily.
export async function loginAction(
  body: LoginRequest,
): Promise<ActionResult<null>> {
  return runServerAction(async () => {
    const response = await authClient.login(body);
    await setSessionCookie(
      response.data.data.access_token,
      response.data.data.expires_in,
    );
    return null;
  });
}

export async function registerAction(
  body: RegisterRequest,
): Promise<ActionResult<null>> {
  return runServerAction(async () => {
    const response = await authClient.register(body);
    await setSessionCookie(
      response.data.data.access_token,
      response.data.data.expires_in,
    );
    return null;
  });
}

export async function getProfileAction(): Promise<
  ActionResult<CommonSuccessResponse<ProfileResponse>>
> {
  return runServerAction(async () => {
    const response = await authClient.getProfile();
    return response.data;
  });
}
