"use client";

import { useMemo } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useTranslations } from "next-intl";
import { Link, useRouter } from "@/i18n/navigation";
import { InputField } from "@/components/form/input-field";
import { PasswordField } from "@/components/form/password-field";
import { Button } from "@/components/ui/button";
import {
  useLoginMutation,
  useRegisterMutation,
} from "@/feature/auth/hooks/query";
import {
  createRegisterSchema,
  type RegisterFormValues,
} from "@/feature/auth/schemas";
import { setClientSessionCookie } from "@/feature/auth/session-client";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";

export function RegisterForm() {
  const t = useTranslations("Auth");
  const router = useRouter();
  const registerMutation = useRegisterMutation();
  const loginMutation = useLoginMutation();

  const form = useForm<RegisterFormValues>({
    resolver: zodResolver(useMemo(() => createRegisterSchema(t), [t])),
    defaultValues: {
      username: "",
      email: "",
      password: "",
      confirmPassword: "",
    },
  });

  const isPending = registerMutation.isPending || loginMutation.isPending;

  const onSubmit = form.handleSubmit(async (values) => {
    try {
      await registerMutation.mutateAsync(
        {
          username: values.username,
          email: values.email,
          password: values.password,
          password_confirm: values.confirmPassword,
        },
        {
          onSuccess: ({ data }) => {
            setClientSessionCookie(data.access_token, data.expires_in);
            router.push("/dashboard");
          },
        },
      );
    } catch (error) {
      const fieldErrors = getApiFieldErrors(error);
      if (fieldErrors) {
        for (const [field, message] of Object.entries(fieldErrors)) {
          form.setError(field as keyof RegisterFormValues, { message });
        }
        return;
      }
      form.setError("root", {
        message: getApiErrorMessage(error, t("registerError")),
      });
      return;
    }
  });

  return (
    <form onSubmit={onSubmit} className="h-full flex w-full flex-col gap-4">
      <div className="h-8/12 space-y-4">
        <InputField
          control={form.control}
          name="username"
          label={t("username")}
          autoComplete="username"
        />

        <InputField
          control={form.control}
          name="email"
          type="email"
          label={t("email")}
          autoComplete="email"
        />

        <PasswordField
          control={form.control}
          name="password"
          label={t("password")}
          autoComplete="new-password"
        />

        <PasswordField
          control={form.control}
          name="confirmPassword"
          label={t("confirmPassword")}
          autoComplete="new-password"
        />
      </div>

      {form.formState.errors.root?.message && (
        <p className="text-sm text-red-600">
          {form.formState.errors.root.message}
        </p>
      )}

      <Button type="submit" disabled={isPending}>
        {isPending ? t("registering") : t("register")}
      </Button>

      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("alreadyHaveAccount")}{" "}
        <Link
          href="/login"
          className="font-medium text-zinc-950 dark:text-zinc-50"
        >
          {t("login")}
        </Link>
      </p>
    </form>
  );
}
