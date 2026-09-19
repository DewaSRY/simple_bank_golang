import { notFound } from "next/navigation";
import { Wallet } from "lucide-react";
import { isAppLocale } from "@/i18n/settings";
import { getTranslation } from "@/i18n/server";
import { RegisterFormScreen } from "@/components/auth/register-form-screen";
import { Link } from "@/i18n/navigation";
import { useTranslation } from "react-i18next";

export function BrandBanner() {
  const { t: tCommon } = useTranslation("common");

  return (
    <Link
      href="/"
      className="flex items-center gap-2 text-lg font-semibold tracking-tight"
    >
      <span className="flex size-8 items-center justify-center rounded-lg bg-primary text-primary-foreground">
        <Wallet className="size-4" aria-hidden />
      </span>
      {tCommon("appName")}
    </Link>
  );
}
