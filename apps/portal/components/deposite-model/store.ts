import { create } from "zustand";

import type { AccountList } from "./type";

type State = {
  accountList: AccountList[];
  selectedAccount: AccountList | null;

  setAccountList: (accountList: AccountList[]) => void;
  setSelectedAccount: (account: AccountList | null) => void;
};

export const useDepositeStore = create<State>((set) => {
  function setAccountList(accountList: AccountList[]) {
    set({ accountList });
  }

  function setSelectedAccount(account: AccountList | null) {
    set({ selectedAccount: account });
  }

  return {
    accountList: [],
    selectedAccount: null,
    setAccountList,
    setSelectedAccount,
  };
});
