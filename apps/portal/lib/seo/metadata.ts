import { locales, type AppLocale } from "@/i18n/settings";

export const SITE_URL = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3000";

/** Builds `alternates.languages` for a given locale-agnostic path, e.g. "/login". */
export function buildLanguageAlternates(path: string): Record<string, string> {
  return Object.fromEntries(
    locales.map((locale) => [locale, `${SITE_URL}/${locale}${path}`]),
  );
}

export function canonicalFor(locale: AppLocale, path: string): string {
  return `${SITE_URL}/${locale}${path}`;
}
