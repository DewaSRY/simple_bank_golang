"use client";

import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import type { AccountEntriesResponse } from "@/feature/account/client";
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
  return (
    <Card>
      <CardHeader className="border-b">
        <CardTitle>Account entries</CardTitle>
        <CardDescription>Recent activity for {accountName}</CardDescription>
      </CardHeader>
      <CardContent className="p-0">
        {entries.length === 0 ? (
          <div className="px-6 py-14 text-center">
            <p className="font-medium">No entries yet</p>
            <p className="mt-1 text-sm text-muted-foreground">
              Transactions for this account will appear here.
            </p>
          </div>
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full min-w-155 text-left">
              <thead className="bg-muted/40 text-xs uppercase tracking-wide text-muted-foreground">
                <tr>
                  <th className="px-4 py-3 font-medium">Entry</th>
                  <th className="px-4 py-3 font-medium">Description</th>
                  <th className="px-4 py-3 text-right font-medium">Amount</th>
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
