"use client";

import { useTranslation } from "react-i18next";
import { ExternalLink } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { ScrollReveal } from "./scroll-reveal";

const REPO_URL = "https://github.com/DewaSRY/simple_bank_golang";

export function FinalCtaSection() {
  const { t } = useTranslation("landing");

  return (
    <section className="px-4 py-16 sm:px-6 sm:py-24">
      <ScrollReveal className="mx-auto flex w-full max-w-4xl flex-col items-center gap-6 rounded-3xl bg-brand-surface px-6 py-14 text-center text-brand-surface-foreground shadow-lg sm:px-16">
        <h2 className="max-w-xl text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
          {t("finalCta.title")}
        </h2>
        <p className="text-brand-surface-foreground/75 text-balance">
          {t("finalCta.description")}
        </p>
        <div className="flex w-full flex-col gap-3 sm:w-auto sm:flex-row">
          <Button
            size="lg"
            variant="secondary"
            className="h-12 px-8 text-base"
            nativeButton={false}
            render={<a href="#journey" />}
          >
            {t("finalCta.ctaExplore")}
          </Button>
          <Button
            size="lg"
            variant="outline"
            className="h-12 border-brand-surface-foreground/30 bg-transparent px-8 text-base text-brand-surface-foreground shadow-none hover:bg-brand-surface-foreground/10 hover:text-brand-surface-foreground dark:border-brand-surface-foreground/30 dark:bg-transparent dark:hover:bg-brand-surface-foreground/10"
            nativeButton={false}
            render={<Link href="/onboarding" />}
          >
            {t("finalCta.ctaDemo")}
          </Button>
          <Button
            size="lg"
            variant="ghost"
            className="h-12 gap-2 px-8 text-base text-brand-surface-foreground hover:bg-brand-surface-foreground/10 hover:text-brand-surface-foreground dark:hover:bg-brand-surface-foreground/10"
            nativeButton={false}
            render={
              <a href={REPO_URL} target="_blank" rel="noopener noreferrer" />
            }
          >
            {t("finalCta.ctaSource")}
            <ExternalLink className="size-4" aria-hidden />
          </Button>
        </div>
      </ScrollReveal>
    </section>
  );
}
