import { useQuery } from "@tanstack/react-query";
import {
  accountClient,
  type AccountWithUserName,
  type ListAccountsParams,
} from "@/feature/account/client";

export const accountKeys = {
  all: ["accounts"] as const,
  list: (params: ListAccountsParams) =>
    [...accountKeys.all, "list", params] as const,
};

export function fetchAccounts(
  params: ListAccountsParams = {},
): Promise<AccountWithUserName[]> {
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
