"use client";

import { useTranslation } from "react-i18next";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AccountEntriesResponse } from "@/feature/account-transaction/type";
import { AccountEntryRow } from "@/components/account/account-entry-row";

export function AccountEntriesCard({
  accountName,
  accountId,
  currency,
  entries,
}: {
  accountName: string;
  accountId: number;
  currency: string;
  entries: AccountEntriesResponse[];
}) {
  const { t } = useTranslation("account");

  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>{t("accountEntriesTitle")}</CardTitle>
        <CardDescription>
          {t("accountEntriesDescription", { name: accountName })}
        </CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        {entries.length === 0 ? (
          <div className="px-6 py-14 text-center">
            <p className="font-medium">{t("noEntriesTitle")}</p>
            <p className="mt-1 text-sm text-muted-foreground">
              {t("noEntriesDescription")}
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-155 text-left">
              <thead className="bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th className="px-4 py-3 font-medium">
                    {t("entryColumn")}
                  </th>
                  <th className="px-4 py-3 font-medium">
                    {t("description")}
                  </th>
                  <th className="px-4 py-3 text-right font-medium">
                    {t("amountColumn")}
                  </th>
                </tr>
              </thead>
              <tbody>
                {entries.map((entry) => (
                  <AccountEntryRow
                    key={entry.id}
                    entry={entry}
                    accountId={accountId}
                    currency={currency}
                  />
                ))}
              </tbody>
            </table>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
