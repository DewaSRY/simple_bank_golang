import { create } from "zustand";

import type { AccountList, DepositeDetails, DepositeStep } from "./type";

type State = {
  step: DepositeStep;
  accountList: AccountList[];
  selectedAccount: AccountList | null;
  details: DepositeDetails | null;

  setAccountList: (accountList: AccountList[]) => void;
  setSelectedAccount: (account: AccountList | null) => void;
  setStep: (step: DepositeStep) => void;
  setDetails: (details: DepositeDetails) => void;
  reset: () => void;
};

const initialState = {
  step: "account" as DepositeStep,
  accountList: [] as AccountList[],
  selectedAccount: null as AccountList | null,
  details: null as DepositeDetails | null,
};

export const useDepositeStore = create<State>((set) => {
  function setAccountList(accountList: AccountList[]) {
    set({ accountList });
  }

  function setSelectedAccount(account: AccountList | null) {
    set({ selectedAccount: account });
  }

  function setStep(step: DepositeStep) {
    set({ step });
  }

  function setDetails(details: DepositeDetails) {
    set({ details });
  }

  function reset() {
    set(initialState);
  }

  return {
    ...initialState,
    setAccountList,
    setSelectedAccount,
    setStep,
    setDetails,
    reset,
  };
});
