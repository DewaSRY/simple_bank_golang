import { useMutation, useQueryClient } from "@tanstack/react-query";

import { queryKeys as accountQueryKeys } from "@/feature/account/hooks/query";
import { queryKeys as accountTransactionQueryKeys } from "@/feature/account-transaction/hooks/query";
import { unwrapActionResult } from "@/lib/api/action-result";

import { createTransferAction } from "../actions";
import type { CreateTransferBody } from "../type";

export function useCreateTransfer() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (body: CreateTransferBody) =>
      createTransferAction(body).then(unwrapActionResult),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: accountQueryKeys.all });
      queryClient.invalidateQueries({
        queryKey: accountTransactionQueryKeys.entries(variables.from_account_id, {
          page: 1,
          limit: 10,
        }),
      });
      queryClient.invalidateQueries({
        queryKey: accountTransactionQueryKeys.recentTransactions(
          variables.from_account_id,
          { page: 1, limit: 10 },
        ),
      });
    },
  });
}
