"use client";

import { useEffect } from "react";
import { useTranslation } from "react-i18next";

import { Card } from "@/components/ui/card";
import ErrorState from "@/components/ui/error-state";
import { PageContainer } from "@/components/common/page-header";

export default function DashboardError({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  const { t } = useTranslation("common");

  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <PageContainer>
      <Card>
        <ErrorState
          title={t("error.title")}
          description={t("error.dashboardDescription")}
          onRetry={reset}
        />
      </Card>
    </PageContainer>
  );
}
