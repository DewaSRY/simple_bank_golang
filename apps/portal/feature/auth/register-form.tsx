"use client";

import { useActionState } from "react";
import { useTranslations } from "next-intl";
import { Link } from "@/i18n/navigation";
import type { AppLocale } from "@/i18n/routing";
import { registerAction } from "./actions";
import { initialAuthFormState } from "./types";

export function RegisterForm({ locale }: { locale: AppLocale }) {
  const t = useTranslations("Auth");
  const [state, formAction, isPending] = useActionState(
    registerAction.bind(null, locale),
    initialAuthFormState,
  );

  return (
    <form action={formAction} className="flex w-full max-w-sm flex-col gap-4">
      <div className="flex flex-col gap-1">
        <label htmlFor="username" className="text-sm font-medium">
          {t("username")}
        </label>
        <input
          id="username"
          name="username"
          type="text"
          required
          autoComplete="username"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.username && (
          <p className="text-sm text-red-600">{state.fieldErrors.username}</p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="email" className="text-sm font-medium">
          {t("email")}
        </label>
        <input
          id="email"
          name="email"
          type="email"
          required
          autoComplete="email"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.email && (
          <p className="text-sm text-red-600">{state.fieldErrors.email}</p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="password" className="text-sm font-medium">
          {t("password")}
        </label>
        <input
          id="password"
          name="password"
          type="password"
          required
          minLength={8}
          autoComplete="new-password"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.password && (
          <p className="text-sm text-red-600">{state.fieldErrors.password}</p>
        )}
      </div>

      <div className="flex flex-col gap-1">
        <label htmlFor="confirmPassword" className="text-sm font-medium">
          {t("confirmPassword")}
        </label>
        <input
          id="confirmPassword"
          name="confirmPassword"
          type="password"
          required
          minLength={8}
          autoComplete="new-password"
          className="rounded border border-black/[.08] bg-transparent px-3 py-2 dark:border-white/[.145]"
        />
        {state.fieldErrors?.confirmPassword && (
          <p className="text-sm text-red-600">
            {state.fieldErrors.confirmPassword}
          </p>
        )}
      </div>

      {state.status === "error" && state.message && (
        <p className="text-sm text-red-600">{state.message}</p>
      )}

      <button
        type="submit"
        disabled={isPending}
        className="rounded-full bg-foreground px-5 py-2 font-medium text-background transition-colors hover:bg-[#383838] disabled:opacity-60 dark:hover:bg-[#ccc]"
      >
        {isPending ? t("registering") : t("register")}
      </button>

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
