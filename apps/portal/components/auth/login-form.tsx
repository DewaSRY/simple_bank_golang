"use client";

import { useMemo } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Link, useRouter } from "@/i18n/navigation";
import { InputField } from "@/components/form/input-field";
import { PasswordField } from "@/components/form/password-field";
import { Button } from "@/components/ui/button";
import { useLoginMutation } from "@/feature/auth/hooks/query";
import {
  createLoginSchema,
  type LoginFormValues,
} from "@/feature/auth/schemas";
import { setClientSessionCookie } from "@/feature/auth/session-client";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";

export function LoginForm() {
  const { t } = useTranslation("auth");
  const router = useRouter();
  const loginMutation = useLoginMutation();

  const form = useForm<LoginFormValues>({
    resolver: zodResolver(useMemo(() => createLoginSchema(t), [t])),
    defaultValues: { email: "", password: "" },
  });

  const onSubmit = form.handleSubmit((values) => {
    loginMutation.mutate(values, {
      onSuccess: ({ data }) => {
        setClientSessionCookie(data.access_token, data.expires_in);
        router.push("/dashboard");
      },
      onError: (error) => {
        const fieldErrors = getApiFieldErrors(error);
        if (fieldErrors) {
          for (const [field, message] of Object.entries(fieldErrors)) {
            form.setError(field as keyof LoginFormValues, { message });
          }
          return;
        }
        form.setError("root", {
          message: getApiErrorMessage(error, t("loginError")),
        });
      },
    });
  });

  return (
    <form onSubmit={onSubmit} className="h-full flex w-full flex-col gap-4">
      <div className="h-8/12 space-y-4">
        <InputField
          control={form.control}
          name="email"
          label={t("email")}
          autoComplete="email"
        />

        <PasswordField
          control={form.control}
          name="password"
          label={t("password")}
          autoComplete="current-password"
        />
      </div>

      {form.formState.errors.root?.message && (
        <p className="text-sm text-red-600">
          {form.formState.errors.root.message}
        </p>
      )}

      <Button type="submit" disabled={loginMutation.isPending}>
        {loginMutation.isPending ? t("loggingIn") : t("login")}
      </Button>

      <p className="text-sm text-zinc-600 dark:text-zinc-400">
        {t("noAccount")}{" "}
        <Link
          href="/register"
          className="font-medium text-zinc-950 dark:text-zinc-50"
        >
          {t("createAccount")}
        </Link>
      </p>
    </form>
  );
}
