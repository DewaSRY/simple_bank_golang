"use client";

import { useEffect } from "react";
import { CheckCircle2 } from "lucide-react";
import { motion } from "motion/react";
import { useTranslation } from "react-i18next";
import { useRouter } from "@/i18n/navigation";

const REDIRECT_DELAY_MS = 1000;

export function AuthSuccessBanner() {
  const { t } = useTranslation("auth");
  const router = useRouter();

  useEffect(() => {
    const timer = setTimeout(() => {
      router.replace("/dashboard");
    }, REDIRECT_DELAY_MS);
    return () => clearTimeout(timer);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <div className="relative flex min-h-screen flex-1 items-center justify-center overflow-hidden bg-background">
      <div
        aria-hidden
        className="pointer-events-none absolute inset-0 bg-[radial-gradient(circle_at_50%_40%,var(--brand-soft),transparent_60%)]"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -left-24 top-1/3 size-72 animate-pulse rounded-full bg-brand/20 blur-3xl"
      />
      <div
        aria-hidden
        className="pointer-events-none absolute -right-24 bottom-1/3 size-72 animate-pulse rounded-full bg-brand-300/20 blur-3xl [animation-delay:300ms]"
      />

      <div className="relative flex flex-col items-center gap-6 px-6 text-center">
        <motion.div
          initial={{ scale: 0.5, opacity: 0 }}
          animate={{ scale: 1, opacity: 1 }}
          transition={{ duration: 0.35, ease: [0.22, 1, 0.36, 1] }}
          className="relative flex size-20 items-center justify-center"
        >
          <span className="absolute inset-0 rounded-full border-4 border-brand/20" />
          <CheckCircle2
            className="size-10 text-brand-700 dark:text-brand"
            aria-hidden
          />
        </motion.div>

        <div className="space-y-1" role="status" aria-live="polite">
          <p className="text-lg font-semibold text-foreground">
            {t("authSuccessTitle")}
          </p>
          <p className="text-sm text-muted-foreground">
            {t("authSuccessSubtitle")}
          </p>
        </div>

        <div className="h-1 w-40 overflow-hidden rounded-full bg-brand/10">
          <div className="h-full w-full origin-left animate-[logout-progress_600ms_ease-out_forwards] rounded-full bg-brand" />
        </div>
      </div>
    </div>
  );
}
