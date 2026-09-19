import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  AccountResponse,
  RequestAccountbody,
  SearchAccountsParams,
  SearchMeAccountsParams,
} from "@/feature/account/type";
import type {
  CommonSuccessResponse,
  PaginationParams,
} from "@/feature/common/type";
import { unwrapActionResult } from "@/lib/api/action-result";
import {
  createAccountAction,
  listMeAccountsAction,
  searchAccountByNumberAction,
} from "../actions";

/**
 * Centralized query keys for the account feature.
 */
export const queryKeys = {
  all: ["accounts"] as const,
  list: (params: Partial<PaginationParams> = {}) =>
    [...queryKeys.all, "list", params] as const,
  searchByNumber: (params: SearchAccountsParams) =>
    [...queryKeys.all, "search-by-number", params] as const,
};

export function useAccounts(params: SearchMeAccountsParams) {
  return useQuery({
    queryKey: queryKeys.list(params),
    queryFn: () => listMeAccountsAction(params).then(unwrapActionResult),
  });
}

export const useCreateAccountMutation = () => {
  const queryClient = useQueryClient();
  return useMutation<
    CommonSuccessResponse<AccountResponse>,
    Error,
    RequestAccountbody
  >({
    mutationFn: (body) => createAccountAction(body).then(unwrapActionResult),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.all,
      });
    },
  });
};

export function useSearchAccountByNumber(
  params: SearchAccountsParams,
  options: { enabled?: boolean } = {},
) {
  return useQuery({
    queryKey: queryKeys.searchByNumber(params),
    queryFn: () =>
      searchAccountByNumberAction(params).then(unwrapActionResult),
    enabled: options.enabled ?? true,
  });
}
