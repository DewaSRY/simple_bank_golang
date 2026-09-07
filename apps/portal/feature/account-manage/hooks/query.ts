import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { accountManageClient } from "@/feature/account-manage/client";
import type { RequestAccountbody } from "@/feature/account/type";

export const queryKeys = {
  manageAccount: (number: number) => ["manage-account", number] as const,
};

export function useAccountDetail(number: number) {
  return useQuery({
    queryKey: queryKeys.manageAccount(number),
    queryFn: () =>
      accountManageClient
        .detailAccount(number)
        .then((response) => response.data.data),
  });
}

export function useUpdateAccount(number: number) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: (body: RequestAccountbody) =>
      accountManageClient
        .updateAccount(number, body)
        .then((response) => response.data),

    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.manageAccount(number),
      });
    },
  });
}

export const useDeleteAccount = (number: number) => {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: () =>
      accountManageClient
        .deleteAccount(number)
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.manageAccount(number),
      });
    },
  });
};
