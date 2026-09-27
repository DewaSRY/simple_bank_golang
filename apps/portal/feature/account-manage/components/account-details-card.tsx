"use client";

import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AccountWithUserName } from "@/feature/account";

function formatDate(value: string, locale: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return null;
  return new Intl.DateTimeFormat(locale, { dateStyle: "medium" }).format(date);
}

export function AccountDetailsCard({
  account,
}: {
  account: AccountWithUserName;
}) {
  const { t, i18n } = useTranslation("account");
  const openedOn = formatDate(account.created_at, i18n.language);

  const rows = [
    { label: t("currency"), value: account.currency },
    ...(openedOn ? [{ label: t("openedOn"), value: openedOn }] : []),
  ];

  return (
    <Card>
      <CardHeader>
        <CardTitle>{t("accountDetailsTitle")}</CardTitle>
        <CardDescription>{t("accountDetailsDescription")}</CardDescription>
      </CardHeader>
      <CardContent>
        <dl className="space-y-4 text-sm">
          {rows.map((row) => (
            <div key={row.label} className="flex items-center justify-between gap-4">
              <dt className="text-muted-foreground">{row.label}</dt>
              <dd className="font-medium">{row.value}</dd>
            </div>
          ))}
          <div className="space-y-1 border-t pt-4">
            <dt className="text-muted-foreground">{t("description")}</dt>
            <dd
              className={
                account.description
                  ? "font-medium wrap-break-word"
                  : "text-muted-foreground italic"
              }
            >
              {account.description || t("noDescription")}
            </dd>
          </div>
        </dl>
      </CardContent>
    </Card>
  );
}
