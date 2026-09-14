import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";

import { AccountList } from "./account-list";
import { useDepositeStore } from "./store";

export function AccountStep() {
  const { selectedAccount, setStep } = useDepositeStore();

  return (
    <div>
      <div className="max-h-80 overflow-y-auto">
        <AccountList />
      </div>

      <DialogFooter>
        <DialogClose render={<Button variant="outline">Cancel</Button>} />
        <Button
          type="button"
          disabled={!selectedAccount}
          onClick={() => setStep("details")}
        >
          Continue
        </Button>
      </DialogFooter>
    </div>
  );
}
