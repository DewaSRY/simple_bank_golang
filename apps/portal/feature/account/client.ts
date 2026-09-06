import { BaseClient } from "../../lib/api/base-client";
import type { CommonSuccessResponse } from "@/feature/common/type";
import type {
  AccountResponse,
  AccountWithUserName,
  RequestAccountbody,
  SearchAccountsParams,
} from "./type";
import type { PaginationParams } from "@/feature/common/type";

export class AccountClient extends BaseClient {
  listAccounts({ page = 1, limit = 10 }: Partial<PaginationParams> = {}) {
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
}

export const accountClient = new AccountClient();
