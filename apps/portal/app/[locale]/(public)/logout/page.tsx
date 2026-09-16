"use client";

import { useEffect } from "react";
import { Loader2 } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useRouter } from "@/i18n/navigation";
import { clearClientSessionCookie } from "@/feature/auth/session-client";

const MIN_VISIBLE_MS = 300;

export default function LogoutPage() {
  const router = useRouter();
  const { t } = useTranslation("common");

  useEffect(() => {
    const start = Date.now();
    clearClientSessionCookie();

    const remaining = Math.max(MIN_VISIBLE_MS - (Date.now() - start), 0);
    const timer = setTimeout(() => {
      router.replace("/login");
    }, remaining);

    return () => clearTimeout(timer);
  }, [router]);

  return (
    <div className="flex min-h-screen flex-1 items-center justify-center">
      <Loader2 className="size-6 animate-spin text-muted-foreground" aria-hidden />
      <span className="sr-only">{t("loading")}</span>
    </div>
  );
}
