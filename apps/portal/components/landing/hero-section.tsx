"use client";

import { useTranslation } from "react-i18next";
import { motion } from "motion/react";
import { ArrowRight, ChevronDown } from "lucide-react";
import { Link } from "@/i18n/navigation";
import { Button } from "@/components/ui/button";
import { AUTHOR } from "@/components/landing/author";
import { GithubIcon, LinkedinIcon } from "@/components/landing/social-icons";
import { SparkleIconLink } from "@/components/landing/sparkle-icon-link";

export function HeroSection() {
  const { t } = useTranslation("landing");

  const pipeline = t("hero.pipeline", { returnObjects: true }) as string[];

  return (
    <section className="relative overflow-hidden px-4 pt-16 pb-20 sm:px-6 sm:pt-24 sm:pb-28">
      <div className="mx-auto flex w-full max-w-[70vw] flex-col items-center gap-6 text-center">
        <motion.span
          initial={{ opacity: 0, y: 12 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, ease: [0.22, 1, 0.36, 1] }}
          className="rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground ring-1 ring-foreground/10"
        >
          {t("hero.eyebrow")}
        </motion.span>

        <motion.h1
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.05, ease: [0.22, 1, 0.36, 1] }}
          className=" text-4xl leading-tight font-semibold tracking-tight text-balance sm:text-5xl sm:leading-tight"
        >
          <span className="mr-0.5">{t("hero.titleBefore")} </span>
          <span className="text-primary">{t("hero.titleBrand")}</span>
        </motion.h1>

        <motion.p
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.1, ease: [0.22, 1, 0.36, 1] }}
          className=" text-lg leading-8 text-muted-foreground text-balance"
        >
          {t("hero.description")}
        </motion.p>

        <motion.div
          initial={{ opacity: 0, y: 16 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5, delay: 0.2 }}
          className="mt-2 flex w-full flex-wrap items-center justify-center gap-1.5 sm:gap-2"
        >
          {pipeline.map((step, index) => (
            <span key={step} className="flex items-center gap-1.5 sm:gap-2">
              <span className="rounded-full bg-card px-3 py-1.5 text-xs font-medium ring-1 ring-foreground/10 sm:text-sm">
                {step}
              </span>
              {index < pipeline.length - 1 && (
                <ArrowRight
                  className="size-3.5 shrink-0 text-muted-foreground/50"
                  aria-hidden
                />
              )}
            </span>
          ))}
        </motion.div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3, ease: [0.22, 1, 0.36, 1] }}
          className="mt-4 flex w-full flex-col gap-3 sm:w-auto sm:flex-row"
        >
          <Button
            size="lg"
            className="h-12 px-6 text-base"
            nativeButton={false}
            render={<a href="#journey" />}
          >
            {t("hero.ctaPrimary")}
          </Button>
          <Button
            size="lg"
            variant="outline"
            className="h-12 px-6 text-base"
            nativeButton={false}
            render={<Link href="/onboarding" />}
          >
            {t("hero.ctaSecondary")}
          </Button>
        </motion.div>

        <motion.p
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.45 }}
          className="text-sm text-muted-foreground"
        >
          {t("hero.note")}
        </motion.p>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          transition={{ duration: 0.5, delay: 0.5 }}
          className="flex items-center gap-4"
        >
          <SparkleIconLink
            href={AUTHOR.githubUrl}
            ariaLabel={t("hero.social.github")}
            icon={<GithubIcon className="size-5" />}
            hoverRotate="left"
          />
          <SparkleIconLink
            href={AUTHOR.linkedinUrl}
            ariaLabel={t("hero.social.linkedin")}
            icon={<LinkedinIcon className="size-5" />}
            hoverRotate="right"
          />
        </motion.div>
      </div>

      <motion.a
        href="#journey"
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.5, delay: 0.6 }}
        className="absolute bottom-4 left-1/2 flex -translate-x-1/2 flex-col items-center gap-1.5 text-xs font-medium text-muted-foreground transition-colors hover:text-foreground sm:bottom-6"
        aria-label={t("hero.scrollHint")}
      >
        <span>{t("hero.scrollHint")}</span>
        <motion.span
          animate={{ y: [0, 6, 0] }}
          transition={{ duration: 1.6, repeat: Infinity, ease: "easeInOut" }}
        >
          <ChevronDown className="size-4" aria-hidden />
        </motion.span>
      </motion.a>
    </section>
  );
}
