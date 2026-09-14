import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { DialogFooter } from "@/components/ui/dialog";
import { useDeposit } from "@/feature/account-transaction/hooks/query";
import { formatAccountAmount } from "@/feature/account-transaction/utils";
import { getApiErrorMessage } from "@/lib/api/error";

import { useDepositeStore } from "./store";

interface Props {
  onSuccess: () => void;
}

export function PreviewStep({ onSuccess }: Props) {
  const { selectedAccount, details, setStep } = useDepositeStore();
  const [error, setError] = useState<string | null>(null);

  const { mutateAsync, isPending } = useDeposit(selectedAccount?.id ?? 0);

  async function handleConfirm() {
    if (!selectedAccount || !details) return;

    setError(null);
    try {
      await mutateAsync(details);
      onSuccess();
    } catch (err) {
      setError(getApiErrorMessage(err, "Failed to deposit. Please try again."));
    }
  }

  if (!selectedAccount || !details) return null;

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">Account</span>
            <div className="text-right">
              <p className="font-medium">{selectedAccount.name}</p>
              <p className="text-sm text-muted-foreground">
                {selectedAccount.number}
              </p>
            </div>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">Amount</span>
            <p className="font-medium">
              {formatAccountAmount(String(details.amount))}
            </p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">Description</span>
            <p className="text-right font-medium">{details.description}</p>
          </div>
        </CardContent>
      </Card>

      {error && <p className="mb-2 text-sm text-destructive">{error}</p>}

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("details")}
          disabled={isPending}
        >
          Back
        </Button>
        <Button type="button" onClick={handleConfirm} disabled={isPending}>
          {isPending ? "Depositing..." : "Confirm Deposit"}
        </Button>
      </DialogFooter>
    </div>
  );
}
