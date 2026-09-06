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

export interface AccountEntriesResponse {
  account_id: number;
  account_name: string;
  account_number: string;
  amount: string;
  id: number;
  is_main: boolean;
  to_account_id: number;
  to_account_name: string;
  to_account_number: string;
  type: string;
  user_id: number;
}

export type DepositRequestBody = {
  amount: number;
  description: string;
};
