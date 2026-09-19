"use server";

import { redirect } from "@/i18n/redirect";
import type { AppLocale } from "@/i18n/settings";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type { CommonSuccessResponse } from "@/feature/common/type";
import { authClient } from "./client";
import type {
  AuthResponse,
  LoginRequest,
  ProfileResponse,
  RegisterRequest,
} from "./client";
import { clearSessionCookie, setSessionCookie } from "./session";

export async function logoutAction(locale: AppLocale): Promise<void> {
  await clearSessionCookie();
  return redirect({ href: "/login", locale });
}

export async function loginAction(
  body: LoginRequest,
): Promise<ActionResult<CommonSuccessResponse<AuthResponse>>> {
  return runServerAction(async () => {
    const response = await authClient.login(body);
    await setSessionCookie(
      response.data.data.access_token,
      response.data.data.expires_in,
    );
    return response.data;
  });
}

export async function registerAction(
  body: RegisterRequest,
): Promise<ActionResult<CommonSuccessResponse<AuthResponse>>> {
  return runServerAction(async () => {
    const response = await authClient.register(body);
    await setSessionCookie(
      response.data.data.access_token,
      response.data.data.expires_in,
    );
    return response.data;
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
