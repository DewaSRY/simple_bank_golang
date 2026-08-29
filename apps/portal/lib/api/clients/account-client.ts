import { BaseClient } from "../base-client";
import type { ApiSuccessResponse } from "../types";

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
    return this.get<ApiSuccessResponse<Account[]>>({
      endpoint: "/accounts",
      params: { page, limit },
    });
  }
}

export const accountClient = new AccountClient();
