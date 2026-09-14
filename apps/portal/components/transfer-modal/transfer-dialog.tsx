import { useEffect } from "react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { DestinationStep } from "./destination-step";
import { DetailsStep } from "./details-step";
import { PreviewStep } from "./preview-step";
import { SourceStep } from "./source-step";
import { useTransferStore } from "./store";

interface Props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

const STEP_COPY = {
  source: {
    title: "Transfer Funds",
    description: "Select the account you want to transfer from.",
  },
  destination: {
    title: "Transfer Funds",
    description: "Choose a recent recipient or search by account number.",
  },
  details: {
    title: "Transfer Funds",
    description: "Enter the amount and a description for this transfer.",
  },
  preview: {
    title: "Review Transfer",
    description: "Confirm the details before submitting your transfer.",
  },
} as const;

export function TransferDialog({ open, setOpen }: Props) {
  const { step, reset } = useTransferStore();
  const { title, description } = STEP_COPY[step];

  useEffect(() => {
    if (open) reset();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [open]);

  function handleOpenChange(nextOpen: boolean) {
    setOpen?.(nextOpen);
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="p-4 lg:min-w-4xl">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {step === "source" && <SourceStep />}
        {step === "destination" && <DestinationStep />}
        {step === "details" && <DetailsStep />}
        {step === "preview" && (
          <PreviewStep onSuccess={() => handleOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  );
}
