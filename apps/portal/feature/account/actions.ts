"use server";

import { revalidatePath } from "next/cache";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type { CommonSuccessResponse } from "@/feature/common";
import { accountClient } from "./client";
import type {
  AccountResponse,
  AccountWithUserName,
  RequestAccountbody,
  SearchAccountsParams,
  SearchMeAccountsParams,
} from "./type";

export async function listMeAccountsAction(
  params: SearchMeAccountsParams,
): Promise<ActionResult<CommonSuccessResponse<AccountWithUserName[]>>> {
  return runServerAction(async () => {
    const response = await accountClient.listMeAccounts(params);
    return response.data;
  });
}

export async function searchAccountByNumberAction(
  params: SearchAccountsParams,
): Promise<ActionResult<CommonSuccessResponse<AccountWithUserName[]>>> {
  return runServerAction(async () => {
    const response = await accountClient.searchAccountByNumber(params);
    return response.data;
  });
}

export async function createAccountAction(
  body: RequestAccountbody,
): Promise<ActionResult<CommonSuccessResponse<AccountResponse>>> {
  return runServerAction(async () => {
    const response = await accountClient.createAccount(body);
    revalidatePath("/[locale]/(protected)", "layout");
    return response.data;
  });
}
