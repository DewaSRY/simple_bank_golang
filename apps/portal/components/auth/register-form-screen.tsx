"use client";

import { useEffect, useState } from "react";
import { zodResolverTranslate } from "@/lib/form";
import { useForm } from "react-hook-form";
import { useTranslation } from "react-i18next";
import { Check } from "lucide-react";
import { useRouter } from "@/i18n/navigation";
import { InputField } from "@/components/form/input-field";
import { PasswordField } from "@/components/form/password-field";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader } from "@/components/ui/card";
import { GuardedLink } from "@/components/common/navigation-guard/guarded-link";
import { useNavigationGuardStore } from "@/components/common/navigation-guard/store";
import {
  useRegisterMutation,
  createRegisterSchema,
  type RegisterFormScreenValues,
} from "@/feature/auth";
import { getApiErrorMessage, getApiFieldErrors } from "@/lib/api/error";
import { BrandBanner } from "./brand-banner";

type RegisterStep = "form" | "preview";

export function RegisterFormScreen() {
  const { t } = useTranslation("auth");
  const { t: tCommon } = useTranslation("common");
  const router = useRouter();
  const registerMutation = useRegisterMutation();

  const [step, setStep] = useState<RegisterStep>("form");

  const form = useForm<RegisterFormScreenValues>({
    resolver: zodResolverTranslate(createRegisterSchema(t), t),
    defaultValues: {
      username: "",
      email: "",
      password: "",
      confirmPassword: "",
    },
  });

  const isPending = registerMutation.isPending;

  const setGuard = useNavigationGuardStore((s) => s.setGuard);
  const isDirty = form.formState.isDirty;

  useEffect(() => {
    setGuard(isDirty, { onAbort: () => form.reset() });
    return () => setGuard(false);
  }, [isDirty, setGuard, form]);

  const goToPreview = form.handleSubmit(async () => {
    const isValid = await form.trigger();
    if (isValid) {
      setStep("preview");
    }
  });

  const onConfirmRegister = async () => {
    const values = form.getValues();
    try {
      await registerMutation.mutateAsync(
        {
          username: values.username,
          email: values.email,
          password: values.password,
          password_confirm: values.confirmPassword,
        },
        {
          onSuccess: () => {
            setGuard(false);
            router.push("/auth-success");
          },
        },
      );
    } catch (error) {
      const fieldErrors = getApiFieldErrors(error);
      if (fieldErrors) {
        setStep("form");
        for (const [field, message] of Object.entries(fieldErrors)) {
          form.setError(field as keyof RegisterFormScreenValues, { message });
        }
        return;
      }
      form.setError("root", {
        message: getApiErrorMessage(error, t("registerError")),
      });
      return;
    }
  };

  const previewValues = form.getValues();

  return (
    <div className="w-full flex items-center justify-center">
      <Card className="w-full z-1  sm:px-4 py-10 sm:min-w-150 space-y-4">
        <CardHeader>
          <div className="flex flex-col space-y-1 border border-brand-100 p-4 rounded-sm ">
            <div className=" mb-2">
              <BrandBanner />
            </div>

            <div className="">
              <h1 className="text-2xl font-semibold tracking-tight">
                {step === "preview" ? t("reviewTitle") : t("registerTitle")}
              </h1>
            </div>
          </div>
        </CardHeader>
        <CardContent className="flex min-h-[50vh] flex-col">
          {step === "form" && (
            <form
              onSubmit={goToPreview}
              className="flex w-full flex-1 flex-col gap-4 justify-between"
            >
              <div className="h-8/12 space-y-4 mb-4">
                <InputField
                  control={form.control}
                  name="username"
                  label={t("username")}
                  autoComplete="username"
                  placeholder={t("username")}
                />

                <InputField
                  control={form.control}
                  name="email"
                  type="email"
                  label={t("email")}
                  autoComplete="email"
                  placeholder={t("email")}
                />

                <PasswordField
                  control={form.control}
                  name="password"
                  label={t("password")}
                  autoComplete="new-password"
                  placeholder={t("password")}
                />

                <PasswordField
                  control={form.control}
                  name="confirmPassword"
                  label={t("confirmPassword")}
                  autoComplete="new-password"
                  placeholder={t("confirmPassword")}
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
                  className="h-11 "
                  disabled={isPending || !form.formState.isDirty}
                >
                  {tCommon("continue")}
                </Button>

                <p className="text-center text-sm text-muted-foreground">
                  {t("alreadyHaveAccount")}
                  <GuardedLink
                    href="/login"
                    className="font-medium text-foreground hover:underline"
                  >
                    {t("loginHere")}
                  </GuardedLink>
                </p>
              </div>
            </form>
          )}
          {step === "preview" && (
            <div className="h-full flex w-full flex-col gap-4 justify-between">
              <Card>
                <CardContent className="space-y-3">
                  <div className="flex items-center justify-between gap-4">
                    <span className="text-sm text-muted-foreground">
                      {t("username")}
                    </span>
                    <p className="font-medium break-all text-right">
                      {previewValues.username}
                    </p>
                  </div>
                  <div className="flex items-center justify-between gap-4">
                    <span className="text-sm text-muted-foreground">
                      {t("email")}
                    </span>
                    <p className="font-medium break-all text-right">
                      {previewValues.email}
                    </p>
                  </div>
                  <div className="flex items-center justify-between gap-4">
                    <span className="text-sm text-muted-foreground">
                      {t("password")}
                    </span>
                    <p className="font-medium flex items-center gap-1">
                      <Check className="size-4 text-brand-600" />
                      {t("passwordSet")}
                    </p>
                  </div>
                </CardContent>
              </Card>
              <div>
                <div className="flex flex-col gap-4">
                  {form.formState.errors.root?.message && (
                    <p className="text-sm text-destructive">
                      {form.formState.errors.root.message}
                    </p>
                  )}
                  <div className="flex gap-2 w-full">
                    <Button
                      type="button"
                      variant="outline"
                      size="lg"
                      className="h-11 flex-1"
                      onClick={() => setStep("form")}
                      disabled={isPending}
                    >
                      {tCommon("back")}
                    </Button>
                    <Button
                      type="button"
                      size="lg"
                      className="h-11 flex-1"
                      onClick={onConfirmRegister}
                      disabled={isPending}
                    >
                      {isPending ? t("registering") : t("confirmAndRegister")}
                    </Button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
