import type { AccountEntriesResponse } from "../account/type";

export function formatAccountAmount(
  amount: string,
  currency: string = "IDR",
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
  t: (key: string, params?: Record<string, unknown>) => string,
) {
  if (entry.type.toLowerCase() === "deposit") return t("deposit");
  if (entry.type.toLowerCase() === "withdraw") return t("withdrawal");

  return entry.account_id === accountId
    ? t("toEntry", { name: entry.to_account_name || entry.to_account_number })
    : t("fromEntry", { name: entry.account_name || entry.account_number });
}
