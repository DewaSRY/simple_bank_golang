import { useEffect } from "react";
import { useForm } from "react-hook-form";

import { Button } from "@/components/ui/button";
import { DialogClose, DialogFooter } from "@/components/ui/dialog";
import { InputField } from "@/components/form/input-field";
import { TextareaField } from "@/components/form/textarea-field";
import { createAccountSchema } from "@/feature/account/schema";
import {
  scrollToFirstError,
  zodResolverTranslate,
} from "@/feature/account/form";
import type { AccountWithUserName } from "@/feature/account/type";

import { useEditAccountStore } from "./store";
import type { EditAccountFormValues } from "./type";

const FIELD_MESSAGES: Record<string, string> = {
  nameRequired: "Name is required",
  descriptionRequired: "Description is required",
};

function translateFieldMessage(key: string) {
  return FIELD_MESSAGES[key] ?? key;
}

interface Props {
  account: AccountWithUserName;
}

export function FormStep({ account }: Props) {
  const { values, fieldErrors, setValues, setFieldErrors, setStep } =
    useEditAccountStore();

  const form = useForm({
    resolver: zodResolverTranslate(createAccountSchema, translateFieldMessage),
    defaultValues: {
      name: values?.name ?? account.name,
      description: values?.description ?? account.description,
    },
  });

  useEffect(() => {
    if (!fieldErrors) return;

    for (const [field, message] of Object.entries(fieldErrors)) {
      form.setError(field as keyof EditAccountFormValues, { message });
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
          label="Name"
          control={form.control}
          autoComplete="name"
        />
        <TextareaField
          name="description"
          label="Description"
          control={form.control}
          cols={40}
          maxLength={400}
          autoComplete="description"
          counter
          counterPosition="bottom"
        />
      </div>

      <DialogFooter>
        <DialogClose render={<Button variant="outline">Cancel</Button>} />
        <Button type="submit">Continue</Button>
      </DialogFooter>
    </form>
  );
}
