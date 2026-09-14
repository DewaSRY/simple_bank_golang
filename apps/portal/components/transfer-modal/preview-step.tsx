import { useState } from "react";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { DialogFooter } from "@/components/ui/dialog";
import { formatAccountAmount } from "@/feature/account-transaction/utils";
import { useCreateTransfer } from "@/feature/transfer/hooks/query";
import { getApiErrorMessage } from "@/lib/api/error";

import { useTransferStore } from "./store";

interface Props {
  onSuccess: () => void;
}

export function PreviewStep({ onSuccess }: Props) {
  const { sourceAccount, destinationAccount, details, setStep } = useTransferStore();
  const [error, setError] = useState<string | null>(null);

  const { mutateAsync, isPending } = useCreateTransfer();

  async function handleConfirm() {
    if (!sourceAccount || !destinationAccount || !details) return;

    setError(null);
    try {
      await mutateAsync({
        from_account_id: sourceAccount.id,
        to_account_id: destinationAccount.id,
        amount: details.amount,
        description: details.description,
      });
      onSuccess();
    } catch (err) {
      setError(getApiErrorMessage(err, "Failed to transfer. Please try again."));
    }
  }

  if (!sourceAccount || !destinationAccount || !details) return null;

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">From</span>
            <div className="text-right">
              <p className="font-medium">{sourceAccount.name}</p>
              <p className="text-sm text-muted-foreground">{sourceAccount.number}</p>
            </div>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">To</span>
            <div className="text-right">
              <p className="font-medium">{destinationAccount.name}</p>
              <p className="text-sm text-muted-foreground">
                {destinationAccount.number}
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
          {isPending ? "Transferring..." : "Confirm Transfer"}
        </Button>
      </DialogFooter>
    </div>
  );
}
