"use client";

import { useTranslation } from "react-i18next";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { ScrollReveal } from "./scroll-reveal";

export function FinalCtaSection() {
  const { t } = useTranslation("common");

  return (
    <section className="px-4 py-16 sm:px-6 sm:py-24">
      <ScrollReveal className="mx-auto flex w-full max-w-4xl flex-col items-center gap-6 rounded-3xl bg-primary px-6 py-14 text-center text-primary-foreground sm:px-16">
        <h2 className="max-w-xl text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
          {t("landing.finalCta.title")}
        </h2>
        <p className="max-w-md text-primary-foreground/70 text-balance">
          {t("landing.finalCta.description")}
        </p>
        <Button
          size="lg"
          variant="secondary"
          className="h-12 px-8 text-base"
          nativeButton={false}
          render={<Link href="/register" />}
        >
          {t("landing.nav.getStarted")}
        </Button>
      </ScrollReveal>
    </section>
  );
}
