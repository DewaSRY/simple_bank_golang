import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";

import { AccountList } from "./account-list";
import { useDepositeStore } from "./store";

export function AccountStep() {
  const { t: tCommon } = useTranslation("common");
  const { selectedAccount, setStep } = useDepositeStore();

  return (
    <div>
      <div className="max-h-80 overflow-y-auto">
        <AccountList />
      </div>

      <DialogFooter>
        <DialogClose
          render={<Button variant="outline">{tCommon("cancel")}</Button>}
        />
        <Button
          type="button"
          disabled={!selectedAccount}
          onClick={() => setStep("details")}
        >
          {tCommon("continue")}
        </Button>
      </DialogFooter>
    </div>
  );
}
