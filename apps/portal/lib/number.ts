export function formatBalance(balance: string) {
  const value = Number(balance);
  if (Number.isNaN(value)) return balance;
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}

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
