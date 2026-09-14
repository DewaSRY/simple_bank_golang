import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { DialogFooter } from "@/components/ui/dialog";
import { InputField } from "@/components/form/input-field";
import { TextareaField } from "@/components/form/textarea-field";
import { scrollToFirstError } from "@/feature/account/form";
import {
  transferDetailsSchema,
  type TransferDetailsFormValues,
} from "@/feature/transfer/schema";

import { useTransferStore } from "./store";

export function DetailsStep() {
  const { details, setDetails, setStep } = useTransferStore();

  const form = useForm<TransferDetailsFormValues>({
    resolver: zodResolver(transferDetailsSchema),
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
          label="Amount"
          control={form.control}
          type="number"
          step="0.01"
          min="0"
          inputMode="decimal"
          placeholder="0.00"
          autoComplete="off"
        />
        <TextareaField
          name="description"
          label="Description"
          control={form.control}
          maxLength={200}
          counter
          counterPosition="bottom"
          placeholder="What's this transfer for?"
        />
      </div>

      <DialogFooter>
        <Button
          type="button"
          variant="outline"
          onClick={() => setStep("destination")}
        >
          Back
        </Button>
        <Button type="submit">Continue</Button>
      </DialogFooter>
    </form>
  );
}
