import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { queryKeys as accountQueryKeys } from "@/feature/account/hooks/query";
import type { PaginationParams } from "@/feature/common/type";
import { unwrapActionResult } from "@/lib/api/action-result";
import { DepositRequestBody } from "../type";
import {
  depositAction,
  getAccountEntriesAction,
  getRecentTransactionsAction,
} from "../actions";

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
      getAccountEntriesAction(accountId, params).then(unwrapActionResult),
  });
}

export function useRecentTransactions(
  accountId: number,
  params: Partial<PaginationParams> = { page: 1, limit: 10 },
) {
  return useQuery({
    queryKey: queryKeys.recentTransactions(accountId, params),
    queryFn: () =>
      getRecentTransactionsAction(accountId, params).then(unwrapActionResult),
  });
}

export function useDeposit(accountId: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: DepositRequestBody) =>
      depositAction(accountId, body).then(unwrapActionResult),
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
