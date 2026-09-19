import { useTranslation } from "react-i18next";
import { useForm } from "react-hook-form";
import { zodResolverTranslate } from "@/lib/form";
import { createAccountSchema } from "@/feature/account/schema";
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
import type { CreateAccountFormValues } from "./type";

export interface props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

export function useCreateAccountForm(
  values: CreateAccountFormValues | null,
  t: (key: string) => string,
) {
  return useForm({
    resolver: zodResolverTranslate(createAccountSchema, t),
    defaultValues: {
      name: "",
      description: "",
    },
  });
}

export type CreateForm = ReturnType<typeof useCreateAccountForm>;

export function CreateAccountDialog({ open, setOpen }: props) {
  const { t } = useTranslation("account");
  const { step, reset, values } = useCreateAccountStore();

  const form = useCreateAccountForm(values, t);

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
      <DialogContent className="lg:min-w-4xl px-4 space-y-4">
        <DialogHeader>
          <DialogTitle className="text-xl font-bold">{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>

        {step === "form" && <FormStep form={form} />}
        {step === "preview" && (
          <PreviewStep onSuccess={() => handleOpenChange(false)} form={form} />
        )}
      </DialogContent>
    </Dialog>
  );
}
