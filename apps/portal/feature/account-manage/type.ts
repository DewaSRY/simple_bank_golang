import type { AccountResponse } from "@/feature/account/type";

export interface DeleteAccountResponse {
  account: AccountResponse;
  balance_swept_to_account_id: number;
}
