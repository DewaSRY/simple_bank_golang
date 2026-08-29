"use client";

import { useLocale, useTranslations } from "next-intl";
import { routing } from "@/i18n/routing";
import { usePathname, useRouter } from "@/i18n/navigation";

import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";

export function LocaleSwitcher() {
  const t = useTranslations("Common");
  const locale = useLocale();
  const router = useRouter();
  const pathname = usePathname();

  function handleLocaleChange(nextLocale: string) {
    router.replace(pathname, { locale: nextLocale });
  }

  return (
    <>
      <DropdownMenu>
        <DropdownMenuTrigger>
          <span>{t("language")}</span>
        </DropdownMenuTrigger>
        <DropdownMenuContent align="end">
          {routing.locales.map((loc) => (
            <DropdownMenuItem key={loc}>
              <Button
                variant="ghost"
                onClick={() => handleLocaleChange(loc)}
                className="w-full justify-end"
              >
                {loc.toUpperCase()}
              </Button>
            </DropdownMenuItem>
          ))}
        </DropdownMenuContent>
      </DropdownMenu>
    </>
  );
}
