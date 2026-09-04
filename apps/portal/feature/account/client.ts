import { BaseClient } from "../../lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";

export type AccountResponse = {
  balance: string;
  created_at: string;
  currency: string;
  description: string;
  id: number;
  is_main: boolean;
  name: string;
  number: string;
  user_id: number;
};

export interface AccountWithUserName {
  id: number;
  balance: string;
  currency: string;
  created_at: string;
  updated_at: string;
  description: string;
  is_main: boolean;
  name: string;
  number: string;
  user_id: number;
  username: string;
}

export interface RequestAccountbody {
  description: string;
  name: string;
}

export interface SearchAccountsParams {
  number: string;
  page?: number;
  limit?: number;
}

export interface DeleteAccountResponse {
  account: AccountResponse;
  balance_swept_to_account_id: number;
}

export interface ListAccountsParams {
  page?: number;
  limit?: number;
}

export class AccountClient extends BaseClient {
  listAccounts({ page = 1, limit = 10 }: ListAccountsParams = {}) {
    return this.get<CommonSuccessResponse<AccountWithUserName[]>>({
      endpoint: "/accounts",
      params: { page, limit },
    });
  }

  createAccount(body: RequestAccountbody) {
    return this.post<CommonSuccessResponse<AccountResponse>>({
      endpoint: "/accounts",
      body,
    });
  }

  searchAccountByNumber({
    number,
    page = 1,
    limit = 10,
  }: SearchAccountsParams) {
    return this.get<CommonSuccessResponse<AccountWithUserName[]>>({
      endpoint: "/accounts/search-by-number",
      params: { number, page, limit },
    });
  }

  updateAccount(id: number, body: RequestAccountbody) {
    return this.post<CommonSuccessResponse<AccountResponse>>({
      endpoint: `/accounts/${id}`,
      body,
    });
  }

  // /accounts/{id}/deposit
}

export const accountClient = new AccountClient();
