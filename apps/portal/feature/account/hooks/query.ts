import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { accountClient } from "@/feature/account/client";
import type {
  AccountResponse,
  RequestAccountbody,
  SearchAccountsParams,
} from "@/feature/account/type";
import type {
  CommonSuccessResponse,
  PaginationParams,
} from "@/feature/common/type";

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

export function useAccounts(params: Partial<PaginationParams> = {}) {
  return useQuery({
    queryKey: queryKeys.list(params),
    queryFn: () =>
      accountClient.listAccounts(params).then((response) => response.data.data),
  });
}

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
        queryKey: queryKeys.all,
      });
    },
  });
};

export function useSearchAccountByNumber(params: SearchAccountsParams) {
  return useQuery({
    queryKey: queryKeys.searchByNumber(params),
    queryFn: () =>
      accountClient
        .searchAccountByNumber(params)
        .then((response) => response.data.data),
  });
}
