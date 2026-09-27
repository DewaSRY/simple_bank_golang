import { useState } from "react";
import { useTranslation } from "react-i18next";

import { PreviewList, PreviewRow } from "@/components/common/preview-row";
import { InlineAlert } from "@/components/common/inline-alert";
import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { useDeposit } from "@/feature/account-transaction";
import { formatAccountAmount } from "@/lib/number";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";

import type { DepositeForm } from "./deposite-dialog";
import { useDepositeStore } from "./store";

interface Props {
  form: DepositeForm;
  onSuccess: () => void;
}

export function PreviewStep({ form, onSuccess }: Props) {
  const { t } = useTranslation("deposit");
  const { t: tCommon } = useTranslation("common");
  const { selectedAccount, details, setStep, setFieldErrors, reset } =
    useDepositeStore();
  const [error, setError] = useState<string | null>(null);

  const { mutateAsync, isPending } = useDeposit(selectedAccount?.id ?? 0);

  async function handleConfirm() {
    if (!selectedAccount || !details) return;

    setError(null);
    try {
      await mutateAsync(details);
      onSuccess();
      form.reset();
      reset();
    } catch (err) {
      const fieldErrors = getApiFieldErrors(err);
      if (fieldErrors) {
        setFieldErrors(fieldErrors);
        setStep("details");
        return;
      }
      setError(getApiErrorMessage(err, t("depositError")));
    }
  }

  if (!selectedAccount || !details) return null;

  return (
    <div className="space-y-4">
      <PreviewList>
        <PreviewRow
          label={t("account")}
          value={selectedAccount.name}
          subValue={selectedAccount.number}
        />
        <PreviewRow label={t("description")} value={details.description} />
        <PreviewRow
          label={t("amount")}
          emphasis
          value={formatAccountAmount(
            String(details.amount),
            selectedAccount.currency,
          )}
        />
      </PreviewList>

      {error && <InlineAlert>{error}</InlineAlert>}

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("details")}
          disabled={isPending}
        >
          {tCommon("back")}
        </Button>
        <Button type="button" onClick={handleConfirm} loading={isPending}>
          {isPending ? t("depositing") : t("confirmDeposit")}
        </Button>
      </DialogFooter>
    </div>
  );
}
