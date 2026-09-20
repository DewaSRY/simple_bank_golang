"use client";

import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Link, useRouter } from "@/i18n/navigation";
import { InputField } from "@/components/form/input-field";
import { PasswordField } from "@/components/form/password-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import {
  useLoginMutation,
  createLoginSchema,
  type LoginFormScreenValues,
} from "@/feature/auth";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";
import { zodResolverTranslate } from "@/lib/form";

import { BrandBanner } from "./brand-banner";

export function LoginFormScreen() {
  const { t } = useTranslation("auth");
  const router = useRouter();
  const loginMutation = useLoginMutation();

  const form = useForm<LoginFormScreenValues>({
    resolver: zodResolverTranslate(createLoginSchema(t), t),
    defaultValues: { email: "" },
  });

  const onSubmit = form.handleSubmit((values) => {
    loginMutation.mutate(values, {
      onSuccess: () => {
        router.push("/auth-success");
      },
      onError: (error) => {
        const fieldErrors = getApiFieldErrors(error);
        if (fieldErrors) {
          for (const [field, message] of Object.entries(fieldErrors)) {
            form.setError(field as keyof LoginFormScreenValues, { message });
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
    <div className="w-full flex items-center justify-center">
      <Card className="w-full z-1 sm:px-4 py-10 sm:min-w-150 space-y-4">
        <CardHeader>
          <div className="flex flex-col space-y-1 border border-brand-100 p-4 rounded-sm ">
            <div className=" mb-2">
              <BrandBanner />
            </div>

            <div className="">
              <h1 className="text-2xl font-semibold tracking-tight">
                {t("loginTitle")}
              </h1>
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex min-h-[50vh] flex-col">
          <form
            onSubmit={onSubmit}
            className="flex w-full flex-1 flex-col gap-4 justify-between"
          >
            <div className=" space-y-4">
              <InputField
                control={form.control}
                name="email"
                label={t("email")}
                autoComplete="email"
                placeholder={t("email")}
              />
            </div>
            <div className="flex flex-col gap-4">
              {form.formState.errors.root?.message && (
                <p className="text-sm text-destructive">
                  {form.formState.errors.root.message}
                </p>
              )}

              <Button
                type="submit"
                size="lg"
                className="h-11"
                disabled={loginMutation.isPending}
              >
                {loginMutation.isPending ? t("loggingIn") : t("login")}
              </Button>

              <p className="text-center text-sm text-muted-foreground">
                {t("noAccount")}{" "}
                <Link
                  href="/register"
                  className="font-medium text-foreground hover:underline"
                >
                  {t("createAccount")}
                </Link>
              </p>
            </div>
          </form>
        </CardContent>
      </Card>
    </div>
  );
}
