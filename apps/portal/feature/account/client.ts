import { BaseClient } from "../../lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";

export interface Account {
  id: number;
  owner: string;
  balance: string;
  currency: string;
  created_at: string;
  updated_at: string;
}

export interface ListAccountsParams {
  page?: number;
  limit?: number;
}

export class AccountClient extends BaseClient {
  listAccounts({ page = 1, limit = 10 }: ListAccountsParams = {}) {
    return this.get<CommonSuccessResponse<Account[]>>({
      endpoint: "/accounts",
      params: { page, limit },
    });
  }
}

export const accountClient = new AccountClient();
