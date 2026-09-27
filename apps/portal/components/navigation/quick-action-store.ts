import { create } from "zustand";

import { useCreateAccountStore } from "@/feature/account";
import {
  useDepositeStore,
  type DepositAccount,
} from "@/feature/account-transaction";
import { useTransferStore, type SourceAccount } from "@/feature/transfer";

export type QuickAction = "transfer" | "deposit" | "create-account";

interface State {
  active: QuickAction | null;
  /** Opens the transfer wizard; with `from`, skips straight to picking a recipient. */
  openTransfer: (from?: SourceAccount) => void;
  /** Opens the deposit wizard; with `into`, skips straight to the amount step. */
  openDeposit: (into?: DepositAccount) => void;
  openCreateAccount: () => void;
  close: () => void;
}

/**
 * Which app-wide action dialog is open. The dialogs are mounted once (see
 * `QuickActionDialogs`) so the sidebar, dashboard and account page can all
 * open the same flow instead of each owning a private copy. Each opener
 * resets that wizard's step store first, so it always starts fresh.
 */
export const useQuickActionStore = create<State>((set) => ({
  active: null,
  openTransfer(from) {
    const transfer = useTransferStore.getState();
    transfer.reset();
    if (from) {
      transfer.setSourceAccount(from);
      transfer.setStep("destination");
    }
    set({ active: "transfer" });
  },
  openDeposit(into) {
    const deposit = useDepositeStore.getState();
    deposit.reset();
    if (into) {
      deposit.setSelectedAccount(into);
      deposit.setStep("details");
    }
    set({ active: "deposit" });
  },
  openCreateAccount() {
    useCreateAccountStore.getState().reset();
    set({ active: "create-account" });
  },
  close() {
    set({ active: null });
  },
}));
