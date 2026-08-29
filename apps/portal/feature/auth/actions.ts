"use server";

import { getTranslations } from "next-intl/server";
import { redirect } from "@/i18n/navigation";
import type { AppLocale } from "@/i18n/routing";
import { authClient } from "@/lib/api/clients/auth-client";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";
import { clearSessionCookie, setSessionCookie } from "./session";
import type { AuthFormState } from "./types";

export async function loginAction(
  locale: AppLocale,
  _prevState: AuthFormState,
  formData: FormData,
): Promise<AuthFormState> {
  const username = String(formData.get("username") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  const t = await getTranslations({ locale, namespace: "Auth" });

  if (!username || !password) {
    return { status: "error", message: t("loginError") };
  }

  let accessToken: string;
  let expiresIn: number;
  try {
    const response = await authClient.login({ username, password });
    accessToken = response.data.data.access_token;
    expiresIn = response.data.data.expires_in;
  } catch (error) {
    return {
      status: "error",
      message: getApiErrorMessage(error, t("loginError")),
      fieldErrors: getApiFieldErrors(error),
    };
  }

  await setSessionCookie(accessToken, expiresIn);
  return redirect({ href: "/dashboard", locale });
}

export async function registerAction(
  locale: AppLocale,
  _prevState: AuthFormState,
  formData: FormData,
): Promise<AuthFormState> {
  const username = String(formData.get("username") ?? "").trim();
  const email = String(formData.get("email") ?? "").trim();
  const password = String(formData.get("password") ?? "");
  const confirmPassword = String(formData.get("confirmPassword") ?? "");
  const t = await getTranslations({ locale, namespace: "Auth" });

  if (!username || !email || !password) {
    return { status: "error", message: t("registerError") };
  }

  if (password !== confirmPassword) {
    return {
      status: "error",
      fieldErrors: { confirmPassword: t("passwordMismatch") },
    };
  }

  try {
    await authClient.register({ username, email, password });
  } catch (error) {
    return {
      status: "error",
      message: getApiErrorMessage(error, t("registerError")),
      fieldErrors: getApiFieldErrors(error),
    };
  }

  try {
    const response = await authClient.login({ username, password });
    await setSessionCookie(
      response.data.data.access_token,
      response.data.data.expires_in,
    );
  } catch {
    return redirect({ href: "/login", locale });
  }

  return redirect({ href: "/dashboard", locale });
}

export async function logoutAction(locale: AppLocale): Promise<void> {
  await clearSessionCookie();
  return redirect({ href: "/login", locale });
}
