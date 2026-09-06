import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  accountClient,
  AccountEntriesResponse,
  type AccountResponse,
  type AccountWithUserName,
  type ListAccountsParams,
  type RequestAccountbody,
} from "@/feature/account/client";
import type {
  CommonSuccessResponse,
  PaginationParams,
} from "@/feature/common/type";

/**
 * Centralized query keys for the account feature.
 */
export const accountKeys = {
  all: ["accounts"] as const,
  list: (params: ListAccountsParams) =>
    [...accountKeys.all, "list", params] as const,
  searchByNumber: (params: ListAccountsParams) =>
    [...accountKeys.all, "search-by-number", params] as const,
  entries: (accountId: number, params: ListAccountsParams) =>
    [...accountKeys.all, "entries", accountId, params] as const,
};

export function fetchAccounts(
  params: ListAccountsParams = {},
): Promise<AccountWithUserName[]> {
  return accountClient
    .listAccounts(params)
    .then((response) => response.data.data);
}

/**
 * List accounts for the current user.
 */
export function useAccounts(params: ListAccountsParams = {}) {
  return useQuery({
    queryKey: accountKeys.list(params),
    queryFn: () => fetchAccounts(params),
  });
}

/**
 * Create a new account.
 */
export const useCreateAccountMutation = () => {
  const queryClient = useQueryClient();
  return useMutation<
    CommonSuccessResponse<AccountResponse>,
    Error,
    RequestAccountbody
  >({
    mutationFn: (body) =>
      accountClient.createAccount(body).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: accountKeys.all,
      });
    },
  });
};

/**
 * Update an existing account.
 */
export const useUpdateAccountMutation = () => {
  return useMutation<
    CommonSuccessResponse<AccountResponse>,
    Error,
    { id: number; body: RequestAccountbody }
  >({
    mutationFn: ({ id, body }) =>
      accountClient.updateAccount(id, body).then((response) => response.data),
  });
};

/**
 * Fetch account entries for a specific account.
 */
export function fetchAccountEntries(
  accountId: number,
  params: Partial<PaginationParams> = { page: 1, limit: 10 },
): Promise<AccountEntriesResponse[]> {
  return accountClient
    .getAccountEntries(accountId, params)
    .then((response) => response.data.data);
}

/**
 * Hook to fetch account entries for a specific account.
 */
export function useAccountEntries(
  accountId: number,
  params: Partial<PaginationParams> = { page: 1, limit: 10 },
) {
  return useQuery({
    queryKey: accountKeys.entries(accountId, params),
    queryFn: () => fetchAccountEntries(accountId, params),
  });
}
