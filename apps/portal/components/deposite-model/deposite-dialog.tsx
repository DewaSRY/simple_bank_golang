import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { AccountStep } from "./account-step";
import { DetailsStep } from "./details-step";
import { PreviewStep } from "./preview-step";
import { useDepositeStore } from "./store";

interface Props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

const STEP_COPY = {
  account: {
    title: "Deposite Funds",
    description: "Select the account you want to deposite into.",
  },
  details: {
    title: "Deposite Funds",
    description: "Enter the amount and a description for this deposite.",
  },
  preview: {
    title: "Review Deposite",
    description: "Confirm the details before submitting your deposite.",
  },
} as const;

export function DepositeDialog({ open, setOpen }: Props) {
  const { step, reset } = useDepositeStore();
  const { title, description } = STEP_COPY[step];

  function handleOpenChange(nextOpen: boolean) {
    setOpen?.(nextOpen);
    if (nextOpen) reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="p-4 lg:min-w-4xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {step === "account" && <AccountStep />}
        {step === "details" && <DetailsStep />}
        {step === "preview" && (
          <PreviewStep onSuccess={() => handleOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  );
}
