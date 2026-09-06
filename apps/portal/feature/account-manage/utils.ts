import type { AccountEntriesResponse } from "@/feature/account/client";

export function formatAccountAmount(
  amount: string,
  currency: string,
  signed = false,
) {
  const value = Number(amount);
  if (Number.isNaN(value)) return amount;

  return `${signed && value > 0 ? "+" : ""}${new Intl.NumberFormat(undefined, {
    style: "currency",
    currency,
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value)}`;
}

export function isIncomingEntry(
  entry: AccountEntriesResponse,
  accountId: number,
) {
  return entry.to_account_id === accountId && entry.account_id !== accountId;
}

export function getEntryLabel(
  entry: AccountEntriesResponse,
  accountId: number,
) {
  if (entry.type.toLowerCase() === "deposit") return "Deposit";
  if (entry.type.toLowerCase() === "withdraw") return "Withdrawal";

  return entry.account_id === accountId
    ? `To ${entry.to_account_name || entry.to_account_number}`
    : `From ${entry.account_name || entry.account_number}`;
}
