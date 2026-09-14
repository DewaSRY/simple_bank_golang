import { useState } from "react";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { DialogFooter } from "@/components/ui/dialog";
import { useCreateAccountMutation } from "@/feature/account/hooks/query";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";

import { useCreateAccountStore } from "./store";

interface Props {
  onSuccess: () => void;
}

export function PreviewStep({ onSuccess }: Props) {
  const { t } = useTranslation("account");
  const { values, setStep, setFieldErrors } = useCreateAccountStore();
  const { mutateAsync, isPending } = useCreateAccountMutation();
  const [error, setError] = useState<string | null>(null);

  async function handleConfirm() {
    if (!values) return;

    setError(null);
    try {
      await mutateAsync(values);
      onSuccess();
    } catch (err) {
      const fieldErrors = getApiFieldErrors(err);
      if (fieldErrors) {
        setFieldErrors(fieldErrors);
        setStep("form");
        return;
      }
      setError(getApiErrorMessage(err, t("registerError")));
    }
  }

  if (!values) return null;

  return (
    <div>
      <Card className="mb-4">
        <CardContent className="space-y-3">
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">
              {t("name")}
            </span>
            <p className="text-right font-medium">{values.name}</p>
          </div>
          <div className="flex items-center justify-between gap-4">
            <span className="text-sm text-muted-foreground">
              {t("description")}
            </span>
            <p className="text-right font-medium">{values.description}</p>
          </div>
        </CardContent>
      </Card>

      {error && <p className="mb-2 text-sm text-destructive">{error}</p>}

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("form")}
          disabled={isPending}
        >
          Back
        </Button>
        <Button type="button" onClick={handleConfirm} disabled={isPending}>
          {isPending ? "Submitting..." : "Confirm"}
        </Button>
      </DialogFooter>
    </div>
  );
}
