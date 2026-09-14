import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { accountTransactionClient } from "@/feature/account-transaction/client";
import { queryKeys as accountQueryKeys } from "@/feature/account/hooks/query";
import type { PaginationParams } from "@/feature/common/type";
import { DepositRequestBody } from "../type";

export const queryKeys = {
  entries: (
    accountId: number,
    params: Partial<PaginationParams> = { page: 1, limit: 10 },
  ) => ["entries", accountId, params] as const,
  recentTransactions: (
    accountId: number,
    params: Partial<PaginationParams> = { page: 1, limit: 10 },
  ) => ["recentTransactions", accountId, params] as const,
};

export function useAccountEntries(
  accountId: number,
  params: Partial<PaginationParams> = { page: 1, limit: 10 },
) {
  return useQuery({
    queryKey: queryKeys.entries(accountId, params),
    queryFn: () =>
      accountTransactionClient
        .getAccountEntries(accountId, params)
        .then((res) => res.data),
  });
}

export function useRecentTransactions(
  accountId: number,
  params: Partial<PaginationParams> = { page: 1, limit: 10 },
) {
  return useQuery({
    queryKey: queryKeys.recentTransactions(accountId, params),
    queryFn: () =>
      accountTransactionClient
        .getRecentTransactions(accountId, params)
        .then((res) => res.data),
  });
}

export function useDeposit(accountId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: DepositRequestBody) =>
      accountTransactionClient.deposit(accountId, body),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.entries(accountId, {
          page: 1,
          limit: 10,
        }),
      });
      queryClient.invalidateQueries({ queryKey: accountQueryKeys.all });
    },
  });
}
