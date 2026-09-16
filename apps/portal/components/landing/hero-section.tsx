"use client";

import { useTranslation } from "react-i18next";
import { motion } from "motion/react";
import { Wallet, ArrowLeftRight, Star } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { Tagline } from "@/components/tagline";

export function HeroSection() {
  const { t } = useTranslation("common");
  const { t: tAuth } = useTranslation("auth");

  return (
    <section className="relative overflow-hidden px-4 pt-16 pb-20 sm:px-6 sm:pt-24 sm:pb-28">
      <div className="mx-auto grid w-full max-w-6xl items-center gap-12 lg:grid-cols-2 lg:gap-8">
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, ease: [0.22, 1, 0.36, 1] }}
          className="flex flex-col items-center gap-6 text-center lg:items-start lg:text-left"
        >
          <span className="rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground ring-1 ring-foreground/10">
            {t("landing.hero.eyebrow")}
          </span>
          <h1 className="max-w-lg text-4xl leading-tight font-semibold tracking-tight text-balance sm:text-5xl sm:leading-tight">
            <Tagline />
          </h1>
          <p className="max-w-md text-lg leading-8 text-muted-foreground text-balance">
            {t("cta")}
          </p>
          <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row">
            <Button
              size="lg"
              className="h-12 px-6 text-base"
              nativeButton={false}
              render={<Link href="/register" />}
            >
              {t("landing.nav.getStarted")}
            </Button>
            <Button
              size="lg"
              variant="outline"
              className="h-12 px-6 text-base"
              nativeButton={false}
              render={<Link href="/login" />}
            >
              {tAuth("login")}
            </Button>
          </div>
          <p className="text-sm text-muted-foreground">
            {t("landing.hero.ctaHint")}
          </p>
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 30, scale: 0.96 }}
          animate={{ opacity: 1, y: 0, scale: 1 }}
          transition={{ duration: 0.7, delay: 0.15, ease: [0.22, 1, 0.36, 1] }}
          className="relative mx-auto w-full max-w-md"
        >
          <div className="relative rounded-3xl bg-card p-3 ring-1 ring-foreground/10 shadow-xl">
            <div className="overflow-hidden rounded-2xl bg-primary p-6 text-primary-foreground">
              <div className="flex items-center justify-between gap-4">
                <div className="flex items-center gap-3">
                  <span className="flex size-10 items-center justify-center rounded-xl bg-primary-foreground/15">
                    <Wallet className="size-5" aria-hidden />
                  </span>
                  <div>
                    <p className="text-sm font-medium">
                      {t("featureAccountsTitle")}
                    </p>
                    <p className="text-xs text-primary-foreground/65">
                      SB-000482193
                    </p>
                  </div>
                </div>
                <span className="rounded-full bg-primary-foreground/15 px-2.5 py-1 text-[11px] font-medium">
                  USD
                </span>
              </div>
              <p className="mt-6 text-xs text-primary-foreground/65">
                {t("availableBalance")}
              </p>
              <p className="mt-1 font-mono text-3xl font-semibold tracking-tight">
                24,650.00
              </p>
            </div>

            <motion.div
              initial={{ opacity: 0, x: -16 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.5 }}
              className="mt-3 flex items-center gap-3 rounded-2xl bg-muted/60 p-3 ring-1 ring-foreground/5"
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-background text-primary ring-1 ring-foreground/10">
                <ArrowLeftRight className="size-4" aria-hidden />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {t("featureTransfersTitle")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("featureTransfersDesc")}
                </p>
              </div>
            </motion.div>

            <motion.div
              initial={{ opacity: 0, x: -16 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ duration: 0.5, delay: 0.65 }}
              className="mt-2 flex items-center gap-3 rounded-2xl p-3"
            >
              <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-warning/15 text-warning">
                <Star className="size-4 fill-current" aria-hidden />
              </span>
              <div className="min-w-0 flex-1">
                <p className="truncate text-sm font-medium">
                  {t("featureSecurityTitle")}
                </p>
                <p className="text-xs text-muted-foreground">
                  {t("featureSecurityDesc")}
                </p>
              </div>
            </motion.div>
          </div>
        </motion.div>
      </div>
    </section>
  );
}
