import { useState } from "react";
import { useTranslation } from "react-i18next";

import { PreviewRow } from "@/components/common/preview-row";
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
  const { t } = useTranslation("transfer");
  const { t: tCommon } = useTranslation("common");
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
      setError(getApiErrorMessage(err, t("transferError")));
    }
  }

  if (!sourceAccount || !destinationAccount || !details) return null;

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="space-y-3">
          <PreviewRow
            label={t("from")}
            value={sourceAccount.name}
            subValue={sourceAccount.number}
          />
          <PreviewRow
            label={t("to")}
            value={destinationAccount.name}
            subValue={destinationAccount.number}
          />
          <PreviewRow
            label={t("amount")}
            value={formatAccountAmount(String(details.amount))}
          />
          <PreviewRow label={t("description")} value={details.description} />
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
          {tCommon("back")}
        </Button>
        <Button type="button" onClick={handleConfirm} disabled={isPending}>
          {isPending ? t("transferring") : t("confirmTransfer")}
        </Button>
      </DialogFooter>
    </div>
  );
}
