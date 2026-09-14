"use client";

import { useTranslation } from "react-i18next";
import { Wallet, ArrowLeftRight, ShieldCheck } from "lucide-react";
import { motion } from "motion/react";
import { ScrollReveal } from "./scroll-reveal";

export function FeaturesSection() {
  const { t } = useTranslation("common");

  const features = [
    {
      icon: Wallet,
      title: t("featureAccountsTitle"),
      description: t("featureAccountsDesc"),
    },
    {
      icon: ArrowLeftRight,
      title: t("featureTransfersTitle"),
      description: t("featureTransfersDesc"),
    },
    {
      icon: ShieldCheck,
      title: t("featureSecurityTitle"),
      description: t("featureSecurityDesc"),
    },
  ];

  return (
    <section id="features" className="px-4 py-16 sm:px-6 sm:py-24">
      <div className="mx-auto w-full max-w-6xl">
        <ScrollReveal className="mx-auto max-w-2xl text-center">
          <span className="text-sm font-medium text-muted-foreground">
            {t("landing.features.eyebrow")}
          </span>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {t("landing.features.title")}
          </h2>
        </ScrollReveal>

        <div className="mt-12 grid gap-5 sm:grid-cols-3">
          {features.map((feature, index) => (
            <ScrollReveal key={feature.title} delay={index * 0.1}>
              <motion.div
                whileHover={{ y: -4 }}
                transition={{ duration: 0.2 }}
                className="h-full rounded-2xl bg-card p-6 ring-1 ring-foreground/10"
              >
                <span className="flex size-11 items-center justify-center rounded-xl bg-primary/10 text-primary">
                  <feature.icon className="size-5" aria-hidden />
                </span>
                <h3 className="mt-4 text-base font-semibold">
                  {feature.title}
                </h3>
                <p className="mt-1.5 text-sm text-muted-foreground">
                  {feature.description}
                </p>
              </motion.div>
            </ScrollReveal>
          ))}
        </div>
      </div>
    </section>
  );
}
