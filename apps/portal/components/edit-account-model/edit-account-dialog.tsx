"use client";

import { useTranslation } from "react-i18next";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import type { AccountWithUserName } from "@/feature/account/type";

import { FormStep } from "./form-step";
import { PreviewStep } from "./preview-step";
import { useEditAccountStore } from "./store";

interface Props {
  account: AccountWithUserName;
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

export function EditAccountDialog({ account, open, setOpen }: Props) {
  const { t } = useTranslation("account");
  const { step, reset } = useEditAccountStore();

  const STEP_COPY = {
    form: {
      title: t("editAccountTitle"),
      description: t("editAccountDescription"),
    },
    preview: {
      title: t("reviewChangesTitle"),
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

        {step === "form" && <FormStep account={account} />}
        {step === "preview" && (
          <PreviewStep
            accountId={account.id}
            onSuccess={() => handleOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  );
}
