import { useQuery } from "@tanstack/react-query";
import {
  accountClient,
  type Account,
  type ListAccountsParams,
} from "@/lib/api/clients/account-client";

export const accountKeys = {
  all: ["accounts"] as const,
  list: (params: ListAccountsParams) =>
    [...accountKeys.all, "list", params] as const,
};

export function fetchAccounts(
  params: ListAccountsParams = {},
): Promise<Account[]> {
  return accountClient
    .listAccounts(params)
    .then((response) => response.data.data);
}

export function useAccounts(params: ListAccountsParams = {}) {
  return useQuery({
    queryKey: accountKeys.list(params),
    queryFn: () => fetchAccounts(params),
  });
}
