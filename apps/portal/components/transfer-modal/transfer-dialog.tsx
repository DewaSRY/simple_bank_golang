import { useEffect } from "react";
import { useTranslation } from "react-i18next";

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

export function TransferDialog({ open, setOpen }: Props) {
  const { t } = useTranslation("transfer");
  const { step, reset } = useTransferStore();

  const STEP_COPY = {
    source: {
      title: t("transferFundsTitle"),
      description: t("selectSourceDescription"),
    },
    destination: {
      title: t("transferFundsTitle"),
      description: t("selectDestinationDescription"),
    },
    details: {
      title: t("transferFundsTitle"),
      description: t("enterDetailsDescription"),
    },
    preview: {
      title: t("reviewTransferTitle"),
      description: t("reviewTransferDescription"),
    },
  } as const;

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
