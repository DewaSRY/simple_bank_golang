"use client";

import { useTranslation } from "react-i18next";
import {
  MousePointerClick,
  RefreshCw,
  Code2,
  Waypoints,
  Server,
} from "lucide-react";
import { motion } from "motion/react";
import { ScrollReveal } from "./scroll-reveal";
import { FlowDiagram, type FlowDiagramStep } from "./flow-diagram";

const ICONS = [MousePointerClick, RefreshCw, Code2, Waypoints, Server];

const TECH_STACK = [
  "Next.js 16",
  "React 19",
  "TanStack Query v5",
  "Zustand",
  "Axios",
  "React Hook Form + Zod",
  "shadcn/ui on Base UI",
  "Tailwind v4",
  "react-i18next",
  "Go",
  "PostgreSQL",
  "Docker",
  "Docker Compose",
  "Nginx",
  "Terraform",
  "AWS",
  "EC2",
];

export function ArchitectureSection() {
  const { t } = useTranslation("common");

  const steps = t("landing.architecture.steps", {
    returnObjects: true,
  }) as Array<{ title: string; description: string }>;
  const callouts = t("landing.architecture.callouts", {
    returnObjects: true,
  }) as string[];

  const diagramSteps: FlowDiagramStep[] = steps.map((step, index) => ({
    ...step,
    icon: ICONS[index],
  }));

  return (
    <section id="architecture" className="px-4 py-16 sm:px-6 sm:py-24">
      <div className="mx-auto w-full max-w-6xl">
        <ScrollReveal className="mx-auto max-w-2xl text-center">
          <span className="text-sm font-medium text-muted-foreground">
            {t("landing.architecture.eyebrow")}
          </span>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {t("landing.architecture.title")}
          </h2>
          <p className="mt-4 text-lg text-muted-foreground text-balance">
            {t("landing.architecture.subtitle")}
          </p>
        </ScrollReveal>

        <div className="mt-12">
          <FlowDiagram steps={diagramSteps} />

          <ScrollReveal
            delay={0.15}
            className="mx-auto mt-5 flex w-fit items-center gap-2 rounded-full border border-dashed border-foreground/20 px-4 py-2 text-xs font-medium text-muted-foreground"
          >
            <span className="size-1.5 rounded-full bg-primary" aria-hidden />
            {t("landing.architecture.prefetch.label")}
          </ScrollReveal>
          <ScrollReveal
            delay={0.2}
            className="mx-auto mt-2 max-w-lg text-center text-sm text-muted-foreground"
          >
            {t("landing.architecture.prefetch.description")}
          </ScrollReveal>
        </div>

        <div className="mt-12 grid gap-4 sm:grid-cols-2">
          {callouts.map((callout, index) => (
            <ScrollReveal key={callout.slice(0, 32)} delay={index * 0.1}>
              <motion.div
                whileHover={{ y: -2 }}
                transition={{ duration: 0.2 }}
                className="h-full rounded-2xl bg-muted/40 p-5 text-sm leading-relaxed text-muted-foreground ring-1 ring-foreground/5"
              >
                {callout}
              </motion.div>
            </ScrollReveal>
          ))}
        </div>

        <ScrollReveal delay={0.1} className="mt-12 text-center">
          <p className="text-xs font-medium tracking-wide text-muted-foreground uppercase">
            {t("landing.architecture.techStackLabel")}
          </p>
          <div className="mt-4 flex flex-wrap justify-center gap-2">
            {TECH_STACK.map((tech) => (
              <span
                key={tech}
                className="rounded-full bg-muted px-3 py-1 text-xs font-medium text-muted-foreground ring-1 ring-foreground/10"
              >
                {tech}
              </span>
            ))}
          </div>
        </ScrollReveal>
      </div>
    </section>
  );
}
