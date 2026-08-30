"use server";

import { redirect } from "@/i18n/navigation";
import type { AppLocale } from "@/i18n/routing";
import { clearSessionCookie } from "./session";

export async function logoutAction(locale: AppLocale): Promise<void> {
  await clearSessionCookie();
  return redirect({ href: "/login", locale });
}
