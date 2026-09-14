import { useTranslation } from "react-i18next";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { FormStep } from "./form-step";
import { PreviewStep } from "./preview-step";
import { useCreateAccountStore } from "./store";

export interface props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

export function CreateAccountDialog({ open, setOpen }: props) {
  const { t } = useTranslation("account");
  const { step, reset } = useCreateAccountStore();

  const STEP_COPY = {
    form: {
      title: t("createAccountTitle"),
      description: t("createAccountDescription"),
    },
    preview: {
      title: t("reviewAccountTitle"),
      description: t("reviewAccountDescription"),
    },
  } as const;

  const { title, description } = STEP_COPY[step];

  function handleOpenChange(nextOpen: boolean) {
    setOpen?.(nextOpen);
    if (nextOpen) reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleOpenChange}>
      <DialogContent className="sm:max-w-sm px-4">
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {step === "form" && <FormStep />}
        {step === "preview" && (
          <PreviewStep onSuccess={() => handleOpenChange(false)} />
        )}
      </DialogContent>
    </Dialog>
  );
}
