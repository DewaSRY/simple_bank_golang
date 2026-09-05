import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { useForm } from "react-hook-form";
import { useState } from "react";
import {
  createAccountSchema,
  CreateAccountFormValues,
} from "@/feature/account/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { InputField } from "@/components/form/input-field";
import { TextareaField } from "@/components/form/textarea-field";

import { useCreateAccountMutation } from "@/feature/account/hooks/query";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";

export interface props {
  open?: boolean;
  setOpen?: (open: boolean) => void;
}

export function CreateAccountDialog({ open, setOpen }: props) {
  const { t } = useTranslation("account");

  const { t: tCommon } = useTranslation("common");
  const { mutateAsync } = useCreateAccountMutation();
  const [isLoading, setIsLoading] = useState(false);

  const form = useForm({
    resolver: zodResolver(useMemo(() => createAccountSchema(t), [t])),
    defaultValues: {
      name: "",
      description: "",
    },
  });

  const onSubmit = form.handleSubmit(async (data) => {
    try {
      setIsLoading(true);
      await mutateAsync(data, {
        onSuccess: () => {
          setOpen?.(false);
        },
      });
      setIsLoading(false);
    } catch (error) {
      setIsLoading(false);
      const fieldErrors = getApiFieldErrors(error);
      if (fieldErrors) {
        for (const [field, message] of Object.entries(fieldErrors)) {
          form.setError(field as keyof CreateAccountFormValues, { message });
        }
        return;
      }
      form.setError("root", {
        message: getApiErrorMessage(error, t("registerError")),
      });
    }
  });

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogContent className="sm:max-w-sm px-4">
        <DialogHeader>
          <DialogTitle>{t("createAccountTitle")}</DialogTitle>
          <DialogDescription>{t("createAccountDescription")}</DialogDescription>
        </DialogHeader>

        <form onSubmit={onSubmit}>
          <div className="space-y-4 mb-6">
            <InputField
              name="name"
              label={t("name")}
              control={form.control}
              autoComplete="name"
              disabled={isLoading}
            />
            <TextareaField
              name="description"
              label={t("description")}
              control={form.control}
              cols={40}
              maxLength={400}
              autoComplete="description"
              counter
              counterPosition="bottom"
              disabled={isLoading}
            />
          </div>

          <DialogFooter>
            <DialogClose render={<Button variant="outline">Cancel</Button>} />
            <Button type="submit" disabled={isLoading}>
              Save changes
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
