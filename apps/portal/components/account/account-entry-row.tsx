"use client";

import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { AccountEntriesResponse } from "@/feature/account-transaction/type";
import {
  formatAccountAmount,
  getEntryLabel,
  isIncomingEntry,
} from "@/feature/account-transaction/utils";

export function AccountEntryRow({
  entry,
  accountId,
  currency,
}: {
  entry: AccountEntriesResponse;
  accountId: number;
  currency: string;
}) {
  const { t } = useTranslation("account");
  const isIncoming = isIncomingEntry(entry, accountId);
  const Icon = isIncoming ? ArrowDownLeft : ArrowUpRight;

  return (
    <tr className="border-b last:border-0 hover:bg-muted/30">
      <td className="px-4 py-4 text-sm text-muted-foreground">#{entry.id}</td>
      <td className="px-4 py-4">
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
            <p className="font-medium">{getEntryLabel(entry, accountId, t)}</p>
            <p className="text-xs capitalize text-muted-foreground">
              {entry.type}
            </p>
          </div>
        </div>
      </td>
      <td
        className={
          isIncoming
            ? "px-4 py-4 text-right font-mono font-medium text-success"
            : "px-4 py-4 text-right font-mono font-medium text-destructive"
        }
      >
        {formatAccountAmount(entry.amount, currency, isIncoming)}
      </td>
    </tr>
  );
}
