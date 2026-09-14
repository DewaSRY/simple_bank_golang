import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { InputField } from "@/components/form/input-field";
import { TextareaField } from "@/components/form/textarea-field";
import { scrollToFirstError, zodResolverTranslate } from "@/feature/account/form";
import {
  transferDetailsSchema,
  type TransferDetailsFormValues,
} from "@/feature/transfer/schema";

import { useTransferStore } from "./store";

export function DetailsStep() {
  const { t } = useTranslation("transfer");
  const { t: tCommon } = useTranslation("common");
  const { details, setDetails, setStep } = useTransferStore();

  const form = useForm<TransferDetailsFormValues>({
    resolver: zodResolverTranslate(transferDetailsSchema, t),
    defaultValues: {
      amount: details ? String(details.amount) : "",
      description: details?.description ?? "",
    },
  });

  const onSubmit = form.handleSubmit(
    (data) => {
      setDetails({ amount: Number(data.amount), description: data.description });
      setStep("preview");
    },
    (errors) => scrollToFirstError(errors),
  );

  return (
    <form onSubmit={onSubmit}>
      <div className="space-y-4 mb-6">
        <InputField
          name="amount"
          label={t("amount")}
          control={form.control}
          type="number"
          step="0.01"
          min="0"
          inputMode="decimal"
          placeholder={t("amountPlaceholder")}
          autoComplete="off"
        />
        <TextareaField
          name="description"
          label={t("description")}
          control={form.control}
          maxLength={200}
          counter
          counterPosition="bottom"
          placeholder={t("descriptionPlaceholder")}
        />
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("destination")}
        >
          {tCommon("back")}
        </Button>
        <Button type="submit">{tCommon("continue")}</Button>
      </DialogFooter>
    </form>
  );
}
