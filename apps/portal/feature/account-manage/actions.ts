"use server";

import { revalidatePath } from "next/cache";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type { CommonSuccessResponse } from "@/feature/common";
import type {
  AccountResponse,
  AccountWithUserName,
  RequestAccountbody,
} from "@/feature/account";
import { accountManageClient } from "./client";
import type { DeleteAccountResponse } from "./type";

export async function detailAccountAction(
  id: number,
): Promise<ActionResult<CommonSuccessResponse<AccountWithUserName>>> {
  return runServerAction(async () => {
    const response = await accountManageClient.detailAccount(id);
    return response.data;
  });
}

export async function updateAccountAction(
  id: number,
  body: RequestAccountbody,
): Promise<ActionResult<CommonSuccessResponse<AccountResponse>>> {
  return runServerAction(async () => {
    const response = await accountManageClient.updateAccount(id, body);
    revalidatePath("/[locale]/(protected)", "layout");
    return response.data;
  });
}

export async function deleteAccountAction(
  id: number,
): Promise<ActionResult<CommonSuccessResponse<DeleteAccountResponse>>> {
  return runServerAction(async () => {
    const response = await accountManageClient.deleteAccount(id);
    revalidatePath("/[locale]/(protected)", "layout");
    return response.data;
  });
}
