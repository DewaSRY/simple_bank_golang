"use server";

import { revalidatePath } from "next/cache";
import { runServerAction, type ActionResult } from "@/lib/api/action-result";
import type { CommonSuccessResponse } from "@/feature/common";
import { transferClient } from "./client";
import type { CreateTransferBody, TransferResponse } from "./type";

export async function createTransferAction(
  body: CreateTransferBody,
): Promise<ActionResult<CommonSuccessResponse<TransferResponse>>> {
  return runServerAction(async () => {
    const response = await transferClient.createTransfer(body);
    revalidatePath("/[locale]/(protected)", "layout");
    return response.data;
  });
}
