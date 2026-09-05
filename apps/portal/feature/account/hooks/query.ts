import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import {
  accountClient,
  type AccountResponse,
  type AccountWithUserName,
  type ListAccountsParams,
  type RequestAccountbody,
} from "@/feature/account/client";
import type { CommonSuccessResponse } from "@/feature/common/type";

/**
 * Centralized query keys for the account feature.
 */
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
