"use client";

import { useTranslation } from "react-i18next";
import { Star, Wallet } from "lucide-react";
import { ScrollReveal } from "./scroll-reveal";

const MOCK_ACCOUNTS = [
  { name: "Main account", number: "SB-000482193", balance: "24,650.00", currency: "USD", main: true },
  { name: "Savings", number: "SB-000482207", balance: "8,120.50", currency: "USD", main: false },
  { name: "Travel", number: "SB-000482318", balance: "1,340.00", currency: "EUR", main: false },
];

export function SolutionSection() {
  const { t } = useTranslation("common");

  return (
    <section className="bg-muted/30 px-4 py-16 sm:px-6 sm:py-24">
      <div className="mx-auto grid w-full max-w-6xl items-center gap-12 lg:grid-cols-2 lg:gap-16">
        <ScrollReveal>
          <span className="text-sm font-medium text-muted-foreground">
            {t("landing.solution.eyebrow")}
          </span>
          <h2 className="mt-3 text-3xl font-semibold tracking-tight text-balance sm:text-4xl">
            {t("landing.solution.title")}
          </h2>
          <p className="mt-4 max-w-md text-lg text-muted-foreground text-balance">
            {t("landing.solution.description")}
          </p>
        </ScrollReveal>

        <ScrollReveal delay={0.1} y={24} className="w-full">
          <div className="rounded-2xl bg-card p-4 ring-1 ring-foreground/10 shadow-lg sm:p-5">
            <div className="mb-3 flex items-center justify-between">
              <p className="text-sm font-semibold">{t("yourAccounts")}</p>
              <Wallet className="size-4 text-muted-foreground" aria-hidden />
            </div>
            <div className="flex flex-col gap-2">
              {MOCK_ACCOUNTS.map((account) => (
                <div
                  key={account.number}
                  className="flex items-center justify-between gap-3 rounded-xl bg-muted/40 p-3 ring-1 ring-foreground/5"
                >
                  <div className="flex min-w-0 items-center gap-3">
                    <span className="flex size-9 shrink-0 items-center justify-center rounded-lg bg-primary/10 text-xs font-semibold text-primary">
                      {account.name.slice(0, 2).toUpperCase()}
                    </span>
                    <div className="min-w-0">
                      <p className="flex items-center gap-1.5 truncate text-sm font-medium">
                        {account.name}
                        {account.main && (
                          <Star
                            className="size-3 shrink-0 fill-warning text-warning"
                            aria-hidden
                          />
                        )}
                      </p>
                      <p className="truncate text-xs text-muted-foreground">
                        {account.number}
                      </p>
                    </div>
                  </div>
                  <div className="flex shrink-0 items-baseline gap-1">
                    <span className="font-mono text-sm font-semibold">
                      {account.balance}
                    </span>
                    <span className="text-xs text-muted-foreground">
                      {account.currency}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          </div>
        </ScrollReveal>
      </div>
    </section>
  );
}
