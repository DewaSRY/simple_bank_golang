"use client";

import { useTranslation } from "react-i18next";
import { Wallet } from "lucide-react";
import { Link } from "@/i18n/navigation";

export function LandingFooter() {
  const { t } = useTranslation("common");
  const { t: tAuth } = useTranslation("auth");

  return (
    <footer className="border-t px-4 py-10 sm:px-6">
      <div className="mx-auto flex w-full max-w-6xl flex-col items-center gap-6 sm:flex-row sm:items-start sm:justify-between">
        <div className="flex flex-col items-center gap-2 sm:items-start">
          <Link
            href="/"
            className="flex items-center gap-2 text-sm font-semibold tracking-tight"
          >
            <span className="flex size-6 items-center justify-center rounded-md bg-primary text-primary-foreground">
              <Wallet className="size-3.5" aria-hidden />
            </span>
            {t("appName")}
          </Link>
          <p className="text-sm text-muted-foreground">
            {t("landing.footer.tagline")}
          </p>
        </div>

        <div className="flex items-center gap-6 text-sm">
          <Link
            href="/register"
            className="text-muted-foreground hover:text-foreground"
          >
            {t("landing.footer.account")}
          </Link>
          <Link
            href="/login"
            className="text-muted-foreground hover:text-foreground"
          >
            {tAuth("login")}
          </Link>
        </div>
      </div>
      <p className="mt-8 text-center text-xs text-muted-foreground sm:text-left">
        © {new Date().getFullYear()} {t("appName")}. {t("landing.footer.rights")}
      </p>
    </footer>
  );
}
