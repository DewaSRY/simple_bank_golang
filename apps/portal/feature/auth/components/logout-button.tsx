import Link from "next/link";
import { LogOut } from "lucide-react";
import { getTranslation } from "@/i18n/server";
import type { AppLocale } from "@/i18n/settings";
import { buttonVariants } from "@/components/ui/button";

export async function LogoutButton({ locale }: { locale: AppLocale }) {
  const { t } = await getTranslation(locale, "auth");

  // Styled link rather than <Link><Button/></Link>: nesting a button inside
  // an anchor is invalid and gives keyboard users two tab stops.
  return (
    <Link
      href={`/${locale}/logout`}
      className={buttonVariants({ variant: "ghost", size: "sm" })}
    >
      <LogOut aria-hidden />
      <span className="hidden sm:inline">{t("logout")}</span>
      <span className="sr-only sm:hidden">{t("logout")}</span>
    </Link>
  );
}
