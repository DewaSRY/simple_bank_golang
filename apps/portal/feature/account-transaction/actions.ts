"use server";

import { revalidatePath } from "next/cache";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type {
  CommonSuccessResponse,
  PaginationParams,
} from "@/feature/common";
import { accountTransactionClient } from "./client";
import type {
  AccountEntriesResponse,
  AccountPublicResponse,
  DepositRequestBody,
} from "./type";

export async function getAccountEntriesAction(
  accountId: number,
  params: Partial<PaginationParams>,
): Promise<ActionResult<CommonSuccessResponse<AccountEntriesResponse[]>>> {
  return runServerAction(async () => {
    const response = await accountTransactionClient.getAccountEntries(
      accountId,
      params,
    );
    return response.data;
  });
}

export async function getRecentTransactionsAction(
  accountId: number,
  params: Partial<PaginationParams>,
): Promise<ActionResult<CommonSuccessResponse<AccountPublicResponse[]>>> {
  return runServerAction(async () => {
    const response = await accountTransactionClient.getRecentTransactions(
      accountId,
      params,
    );
    return response.data;
  });
}

export async function depositAction(
  accountId: number,
  body: DepositRequestBody,
): Promise<ActionResult<CommonSuccessResponse<AccountEntriesResponse>>> {
  return runServerAction(async () => {
    const response = await accountTransactionClient.deposit(accountId, body);
    revalidatePath("/[locale]/(protected)", "layout");
    return response.data;
  });
}
