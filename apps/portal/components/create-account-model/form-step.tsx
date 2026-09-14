import { useEffect } from "react";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";

import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { InputField } from "@/components/form/input-field";
import { TextareaField } from "@/components/form/textarea-field";
import {
  createAccountSchema,
  CreateAccountFormValues,
} from "@/feature/account/schema";
import {
  scrollToFirstError,
  zodResolverTranslate,
} from "@/feature/account/form";

import { useCreateAccountStore } from "./store";

export function FormStep() {
  const { t } = useTranslation("account");
  const { t: tCommon } = useTranslation("common");
  const { values, fieldErrors, setValues, setFieldErrors, setStep } =
    useCreateAccountStore();

  const form = useForm({
    resolver: zodResolverTranslate(createAccountSchema, t),
    defaultValues: {
      name: values?.name ?? "",
      description: values?.description ?? "",
    },
  });

  useEffect(() => {
    if (!fieldErrors) return;

    for (const [field, message] of Object.entries(fieldErrors)) {
      form.setError(field as keyof CreateAccountFormValues, { message });
    }
    setFieldErrors(null);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fieldErrors]);

  const onSubmit = form.handleSubmit(
    (data) => {
      setValues(data);
      setStep("preview");
    },
    (errors) => scrollToFirstError(errors),
  );

  return (
    <form onSubmit={onSubmit}>
      <div className="space-y-4 mb-6">
        <InputField
          name="name"
          label={t("name")}
          control={form.control}
          autoComplete="name"
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
        />
      </div>

      <DialogFooter>
        <DialogClose
          render={<Button variant="outline">{tCommon("cancel")}</Button>}
        />
        <Button type="submit">{tCommon("continue")}</Button>
      </DialogFooter>
    </form>
  );
}
