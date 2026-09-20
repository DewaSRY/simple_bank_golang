import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type { RequestAccountbody } from "@/feature/account";
import { queryKeys as accountQueryKeys } from "@/feature/account/hooks/query";
import { unwrapActionResult } from "@/lib/api/action-result";
import {
  deleteAccountAction,
  detailAccountAction,
  updateAccountAction,
} from "../actions";

export const queryKeys = {
  manageAccount: (number: number) =>
    ["accounts", "list", "manage-account", number] as const,
};

export function useAccountDetail(number: number) {
  return useQuery({
    queryKey: queryKeys.manageAccount(number),
    queryFn: () => detailAccountAction(number).then(unwrapActionResult),
  });
}

export function useUpdateAccount(number: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: RequestAccountbody) =>
      updateAccountAction(number, body).then(unwrapActionResult),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.manageAccount(number),
      });
      queryClient.invalidateQueries({ queryKey: accountQueryKeys.all });
    },
  });
}

export const useDeleteAccount = (number: number) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () => deleteAccountAction(number).then(unwrapActionResult),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.manageAccount(number),
      });
      queryClient.invalidateQueries({ queryKey: accountQueryKeys.all });
    },
  });
};
