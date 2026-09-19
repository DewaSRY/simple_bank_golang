"use client";

import { motion, type Variants } from "motion/react";
import type { LucideIcon } from "lucide-react";
import { cn } from "@/lib/utils";

export interface FlowDiagramStep {
  icon: LucideIcon;
  title: string;
  description: string;
}

interface FlowDiagramProps {
  steps: FlowDiagramStep[];
  className?: string;
}

const listVariants: Variants = {
  hidden: {},
  visible: { transition: { staggerChildren: 0.12 } },
};

const nodeVariants: Variants = {
  hidden: { opacity: 0, y: 16 },
  visible: {
    opacity: 1,
    y: 0,
    transition: { duration: 0.4, ease: [0.22, 1, 0.36, 1] },
  },
};

export function FlowDiagram({ steps, className }: FlowDiagramProps) {
  return (
    <motion.div
      variants={listVariants}
      initial="hidden"
      whileInView="visible"
      viewport={{ once: true, margin: "-80px" }}
      className={cn("flex flex-col md:flex-row md:items-center", className)}
    >
      {steps.map((step, index) => (
        <div
          key={step.title}
          className="flex flex-1 flex-col md:flex-row md:items-center"
        >
          <motion.div
            variants={nodeVariants}
            className="flex flex-1 flex-col items-center gap-2 rounded-2xl bg-card p-4 text-center ring-1 ring-foreground/10"
          >
            <span className="flex size-10 items-center justify-center rounded-xl bg-primary/10 text-primary">
              <step.icon className="size-5" aria-hidden />
            </span>
            <p className="text-sm font-semibold">{step.title}</p>
            <p className="text-xs text-muted-foreground">
              {step.description}
            </p>
          </motion.div>

          {index < steps.length - 1 && <FlowConnector />}
        </div>
      ))}
    </motion.div>
  );
}

function FlowConnector() {
  return (
    <div className="flex shrink-0 items-center justify-center py-1 md:px-1 md:py-0">
      <svg
        className="h-6 w-4 rotate-90 text-muted-foreground/40 md:h-4 md:w-6 md:rotate-0"
        viewBox="0 0 24 16"
        fill="none"
        aria-hidden
      >
        <motion.path
          d="M1 8H21M21 8L15 2M21 8L15 14"
          stroke="currentColor"
          strokeWidth="1.5"
          strokeLinecap="round"
          strokeLinejoin="round"
          initial={{ pathLength: 0, opacity: 0 }}
          whileInView={{ pathLength: 1, opacity: 1 }}
          viewport={{ once: true }}
          transition={{ duration: 0.4, delay: 0.15 }}
        />
      </svg>
    </div>
  );
}
