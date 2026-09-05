export function formatBalance(balance: string) {
  const value = Number(balance);
  if (Number.isNaN(value)) return balance;
  return new Intl.NumberFormat(undefined, {
    minimumFractionDigits: 2,
    maximumFractionDigits: 2,
  }).format(value);
}
