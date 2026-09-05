"use server";

import { redirect } from "@/i18n/redirect";
import type { AppLocale } from "@/i18n/settings";
import { clearSessionCookie } from "./session";

export async function logoutAction(locale: AppLocale): Promise<void> {
  await clearSessionCookie();
  return redirect({ href: "/login", locale });
}
