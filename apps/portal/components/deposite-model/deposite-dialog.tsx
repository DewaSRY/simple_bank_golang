import { useTranslation } from "react-i18next";

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

export function DepositeDialog({ open, setOpen }: Props) {
  const { t } = useTranslation("deposit");
  const { step, reset } = useDepositeStore();

  const STEP_COPY = {
    account: {
      title: t("depositFundsTitle"),
      description: t("selectAccountDescription"),
    },
    details: {
      title: t("depositFundsTitle"),
      description: t("enterDetailsDescription"),
    },
    preview: {
      title: t("reviewDepositTitle"),
      description: t("reviewDepositDescription"),
    },
  } as const;

  const { title, description } = STEP_COPY[step];

  function handleOpenChange(nextOpen: boolean) {
    setOpen?.(nextOpen);
    if (nextOpen) reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="p-4 lg:min-w-4xl space-y-4">
        <DialogHeader>
          <DialogTitle className="text-xl font-bold">{title}</DialogTitle>
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
