import { BaseClient } from "../../lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";
// import type { PaginationParams } from "@/feature/common/type";
import type { CreateTransferBody, TransferResponse } from "./type";

export class TransferClient extends BaseClient {
  createTransfer(body: CreateTransferBody) {
    return this.post<CommonSuccessResponse<TransferResponse>>({
      endpoint: "/transfers",
      body,
    });
  }
}

export const transferClient = new TransferClient();
