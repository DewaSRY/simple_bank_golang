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

export interface AccountPublicResponse {
  id: number;
  name: string;
  number: string;
  user_id: number;
  username: string;
}
