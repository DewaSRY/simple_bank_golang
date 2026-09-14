"use client";

import { useTranslation } from "react-i18next";
import { Layers, Clock, ShieldAlert } from "lucide-react";
import { ScrollReveal } from "./scroll-reveal";

const ICONS = [Layers, Clock, ShieldAlert];

export function ProblemSection() {
  const { t } = useTranslation("common");
  const points = t("landing.problem.points", {
    returnObjects: true,
  }) as Array<{ title: string; description: string }>;

  return (
    <section className="px-4 py-16 sm:px-6 sm:py-24">
      <div className="mx-auto w-full max-w-6xl">
        <ScrollReveal className="mx-auto max-w-2xl text-center">
          <span className="text-sm font-medium text-muted-foreground">
            {t("landing.problem.eyebrow")}
          </span>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {t("landing.problem.title")}
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-balance">
            {t("landing.problem.description")}
          </p>
        </ScrollReveal>

        <div className="mt-12 grid gap-4 sm:grid-cols-3">
          {points.map((point, index) => {
            const Icon = ICONS[index];
            return (
              <ScrollReveal
                key={point.title}
                delay={index * 0.1}
                className="rounded-2xl bg-muted/40 p-5 ring-1 ring-foreground/5"
              >
                <Icon className="size-5 text-muted-foreground" aria-hidden />
                <h3 className="mt-3 text-sm font-semibold">{point.title}</h3>
                <p className="mt-1 text-sm text-muted-foreground">
                  {point.description}
                </p>
              </ScrollReveal>
            );
          })}
        </div>
      </div>
    </section>
  );
}
