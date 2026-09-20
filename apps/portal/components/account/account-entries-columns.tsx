"use client";

import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import {
  createColumnHelper,
  metaHelper,
  tableFeatures,
} from "@tanstack/react-table";
import type { DisplayLedgerEntry } from "@/feature/account-transaction";
import { getEntryLabel } from "@/feature/account-transaction";
import { formatAccountAmount } from "@/lib/number";

type AccountEntryColumnMeta = {
  align?: "left" | "right";
};

export const accountEntriesTableFeatures = tableFeatures({
  columnMeta: metaHelper<AccountEntryColumnMeta>(),
});

const columnHelper = createColumnHelper<
  typeof accountEntriesTableFeatures,
  DisplayLedgerEntry
>();

export function getAccountEntriesColumns({
  accountId,
  currency,
  t,
}: {
  accountId: number;
  currency: string;
  t: (key: string, params?: Record<string, unknown>) => string;
}) {
  return columnHelper.columns([
    columnHelper.accessor("id", {
      header: t("entryColumn"),
      cell: ({ getValue }) => (
        <span className="text-sm text-muted-foreground">#{getValue()}</span>
      ),
    }),
    columnHelper.display({
      id: "description",
      header: t("transactionType"),
      cell: ({ row }) => {
        const entry = row.original;
        const isIncoming = entry.direction === "incoming";
        const Icon = isIncoming ? ArrowDownLeft : ArrowUpRight;

        return (
          <div className="flex items-center gap-3">
            <span
              className={
                isIncoming
                  ? "flex size-8 items-center justify-center rounded-full bg-success/10 text-success"
                  : "flex size-8 items-center justify-center rounded-full bg-destructive/10 text-destructive"
              }
            >
              <Icon className="size-4" aria-hidden />
            </span>
            <div>
              <p className="font-medium">
                {getEntryLabel(entry, accountId, t)}
              </p>
              <p className="text-xs capitalize text-muted-foreground">
                {entry.type}
              </p>
            </div>
          </div>
        );
      },
    }),
    columnHelper.display({
      id: "details",
      header: t("description"),
      cell: ({ row }) => {
        const entry = row.original;
        const isIncoming = entry.direction === "incoming";
        const isTransfer = !["deposit", "withdraw"].includes(
          entry.type.toLowerCase(),
        );
        const counterpartyNumber = isIncoming
          ? entry.account_number
          : entry.to_account_number;

        if (!isTransfer && !entry.description) {
          return <span className="text-xs text-muted-foreground">—</span>;
        }

        return (
          <div className="space-y-0.5">
            {isTransfer && counterpartyNumber ? (
              <p className="font-mono text-xs font-semibold text-muted-foreground">
                {t(isIncoming ? "fromAccountNumber" : "toAccountNumber", {
                  number: counterpartyNumber,
                })}
              </p>
            ) : null}
            {entry.description ? (
              <p className="text-xs text-muted-foreground">
                {entry.description}
              </p>
            ) : null}
          </div>
        );
      },
    }),
    columnHelper.display({
      id: "amount",
      header: t("amountColumn"),
      meta: { align: "right" },
      cell: ({ row }) => {
        const entry = row.original;
        const isIncoming = entry.direction === "incoming";

        return (
          <span
            className={
              isIncoming
                ? "font-mono font-medium text-green-500"
                : "font-mono font-medium text-red-500"
            }
          >
            {formatAccountAmount(entry.amount, currency, isIncoming)}
          </span>
        );
      },
    }),
  ]);
}
