"use client";

import { ArrowLeft, SearchX, type LucideIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Link } from "@/i18n/navigation";
import { Card } from "@/components/ui/card";
import { buttonVariants } from "@/components/ui/button";
import EmptyState from "@/components/ui/empty-state";
import ErrorState from "@/components/ui/error-state";
import { PageContainer } from "@/components/common/page-header";

export function AccountStateMessage({
  title,
  description,
  variant = "empty",
  icon = SearchX,
  onRetry,
}: {
  title: string;
  description: string;
  variant?: "empty" | "error";
  icon?: LucideIcon;
  onRetry?: () => void;
}) {
  const { t } = useTranslation("account");

  const backLink = (
    <Link
      href="/dashboard"
      className={buttonVariants({ variant: "outline" })}
    >
      <ArrowLeft aria-hidden />
      {t("returnToDashboard")}
    </Link>
  );

  return (
    <PageContainer className="max-w-lg py-8">
      <Card>
        {variant === "error" ? (
          <ErrorState title={title} description={description} onRetry={onRetry}>
            {backLink}
          </ErrorState>
        ) : (
          <EmptyState icon={icon} title={title} description={description}>
            {backLink}
          </EmptyState>
        )}
      </Card>
    </PageContainer>
  );
}
