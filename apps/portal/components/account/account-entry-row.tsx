"use client";

import { ArrowDownLeft, ArrowUpRight } from "lucide-react";
import type { AccountEntriesResponse } from "@/feature/account/client";
import {
  formatAccountAmount,
  getEntryLabel,
  isIncomingEntry,
} from "@/feature/account/utils";

export function AccountEntryRow({
  entry,
  accountId,
  currency,
}: {
  entry: AccountEntriesResponse;
  accountId: number;
  currency: string;
}) {
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
                ? "flex size-8 items-center justify-center rounded-full bg-emerald-500/10 text-emerald-600"
                : "flex size-8 items-center justify-center rounded-full bg-rose-500/10 text-rose-600"
            }
          >
            <Icon className="size-4" aria-hidden />
          </span>
          <div>
            <p className="font-medium">{getEntryLabel(entry, accountId)}</p>
            <p className="text-xs capitalize text-muted-foreground">
              {entry.type}
            </p>
          </div>
        </div>
      </td>
      <td
        className={
          isIncoming
            ? "px-4 py-4 text-right font-mono font-medium text-emerald-600"
            : "px-4 py-4 text-right font-mono font-medium text-rose-600"
        }
      >
        {formatAccountAmount(entry.amount, currency, isIncoming)}
      </td>
    </tr>
  );
}
