"use client";

import { useLocale, useTranslations } from "next-intl";
import { routing } from "@/i18n/routing";
import { usePathname, useRouter } from "@/i18n/navigation";

export function LocaleSwitcher() {
  const t = useTranslations("Common");
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  return (
    <label className="flex items-center gap-2 text-sm text-zinc-600 dark:text-zinc-400">
      <span>{t("language")}</span>
      <select
        className="rounded border border-black/8 bg-transparent px-2 py-1 dark:border-white/[.145]"
        value={locale}
        onChange={(event) => {
          const nextLocale = event.target.value;
          router.replace(pathname, { locale: nextLocale });
        }}
      >
        {routing.locales.map((loc) => (
          <option key={loc} value={loc}>
            {loc.toUpperCase()}
          </option>
        ))}
      </select>
    </label>
  );
}
