"use client";

import { useTranslation } from "react-i18next";
import { UserPlus, PiggyBank, Send } from "lucide-react";
import { ScrollReveal } from "./scroll-reveal";

const ICONS = [UserPlus, PiggyBank, Send];

export function HowItWorksSection() {
  const { t } = useTranslation("common");
  const steps = t("landing.howItWorks.steps", {
    returnObjects: true,
  }) as Array<{ title: string; description: string }>;

  return (
    <section className="bg-muted/30 px-4 py-16 sm:px-6 sm:py-24">
      <div className="mx-auto w-full max-w-6xl">
        <ScrollReveal className="mx-auto max-w-2xl text-center">
          <span className="text-sm font-medium text-muted-foreground">
            {t("landing.howItWorks.eyebrow")}
          </span>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {t("landing.howItWorks.title")}
          </h2>
        </ScrollReveal>

        <div className="relative mt-12 grid gap-8 sm:grid-cols-3 sm:gap-6">
          {steps.map((step, index) => {
            const Icon = ICONS[index];
            return (
              <ScrollReveal
                key={step.title}
                delay={index * 0.12}
                className="relative flex flex-col items-center text-center sm:items-start sm:text-left"
              >
                <div className="flex items-center gap-3">
                  <span className="flex size-11 shrink-0 items-center justify-center rounded-xl bg-primary text-primary-foreground">
                    <Icon className="size-5" aria-hidden />
                  </span>
                  <span className="font-mono text-sm text-muted-foreground">
                    0{index + 1}
                  </span>
                </div>
                <h3 className="mt-4 text-base font-semibold">{step.title}</h3>
                <p className="mt-1.5 text-sm text-muted-foreground">
                  {step.description}
                </p>
              </ScrollReveal>
            );
          })}
        </div>
      </div>
    </section>
  );
}
